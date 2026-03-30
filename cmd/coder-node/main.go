package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/trungtran/coder/api/grpc/chatpb"
	"github.com/trungtran/coder/api/grpc/debugpb"
	"github.com/trungtran/coder/api/grpc/memorypb"
	"github.com/trungtran/coder/api/grpc/reviewpb"
	"github.com/trungtran/coder/api/grpc/skillpb"
	authdomain "github.com/trungtran/coder/internal/domain/auth"
	"github.com/trungtran/coder/internal/infra/embedding"
	"github.com/trungtran/coder/internal/infra/llm"
	"github.com/trungtran/coder/internal/infra/postgres"
	grpcinterceptor "github.com/trungtran/coder/internal/transport/grpc/interceptor"
	grpcserver "github.com/trungtran/coder/internal/transport/grpc/server"
	httpmiddleware "github.com/trungtran/coder/internal/transport/http/middleware"
	httpserver "github.com/trungtran/coder/internal/transport/http/server"
	dashboard "github.com/trungtran/coder/internal/transport/http/server/dashboard"
	ucauth "github.com/trungtran/coder/internal/usecase/auth"
	ucchat "github.com/trungtran/coder/internal/usecase/chat"
	ucdebug "github.com/trungtran/coder/internal/usecase/debug"
	ucmemory "github.com/trungtran/coder/internal/usecase/memory"
	ucreview "github.com/trungtran/coder/internal/usecase/review"
	ucskill "github.com/trungtran/coder/internal/usecase/skill"
	"github.com/trungtran/coder/internal/version"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "50051"
	}

	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		httpPort = "8080"
	}

	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		log.Fatalf("POSTGRES_DSN is required")
	}

	ollamaBase := os.Getenv("OLLAMA_BASE_URL")
	if ollamaBase == "" {
		log.Fatalf("OLLAMA_BASE_URL is required")
	}

	ollamaModel := os.Getenv("OLLAMA_EMBEDDING_MODEL")
	if ollamaModel == "" {
		ollamaModel = "mxbai-embed-large"
	}

	ollamaChatModel := os.Getenv("OLLAMA_CHAT_MODEL")
	if ollamaChatModel == "" {
		ollamaChatModel = "qwen3.5:0.8b"
	}

	// Initialize Postgres with shared DB handle
	memDB, rawDB, err := postgres.NewPostgresMemoryWithDB(dsn)
	if err != nil {
		log.Fatalf("Failed to initialize postgres: %v", err)
	}

	// Initialize Ollama embedding provider
	provider := &embedding.OllamaEmbeddingProvider{
		BaseURL: ollamaBase,
		Model:   ollamaModel,
	}

	// Initialize memory manager (use case)
	mgr := ucmemory.NewManager(memDB, provider)
	defer mgr.Close()

	// Initialize skill store (infrastructure, shares same DB connection)
	skillStore, err := postgres.NewPostgresSkillStore(rawDB)
	if err != nil {
		log.Fatalf("Failed to initialize skill store: %v", err)
	}

	// Initialize skill ingestor (use case)
	skillIngestor := ucskill.NewIngestor(skillStore, provider)

	// Initialize skill facade (use case — combines ingestor + store)
	skillFacade := ucskill.NewSkillFacade(skillIngestor, skillStore)

	// Secure mode setup
	ctx := context.Background()
	var authMgr authdomain.AuthManager

	if os.Getenv("SECURE_MODE") == "true" {
		authRepo, err := postgres.NewPostgresAuth(rawDB)
		if err != nil {
			log.Fatalf("Failed to initialize auth repository: %v", err)
		}
		authMgr = ucauth.NewManager(authRepo, true)

		// Print bootstrap token if this is the first startup
		bootstrapToken, err := authMgr.GetBootstrapToken(ctx)
		if err != nil {
			log.Printf("Warning: failed to generate bootstrap token: %v", err)
		} else if bootstrapToken != "" {
			fmt.Printf("\nBOOTSTRAP TOKEN (shown once): %s\n", bootstrapToken)
			fmt.Println("   Share this with clients so they can run: coder login")
			fmt.Println()
		}
	} else {
		authMgr = ucauth.NewManager(nil, false) // open mode — no-op auth
	}

	// Initialize LLM provider (Ollama /api/chat)
	llmProvider := llm.NewOllamaProvider(ollamaBase, ollamaChatModel)

	// Initialize chat repository
	chatRepo, err := postgres.NewPostgresChatRepo(rawDB)
	if err != nil {
		log.Fatalf("Failed to initialize chat repository: %v", err)
	}

	// Initialize chat manager (use case)
	chatMgr := ucchat.NewManager(chatRepo, llmProvider, mgr, skillFacade, ucchat.Config{
		Model: ollamaChatModel,
	})

	// Initialize review and debug managers (use cases)
	reviewMgr := ucreview.NewManager(llmProvider, mgr, skillFacade, ollamaChatModel)
	debugMgr := ucdebug.NewManager(llmProvider, mgr, skillFacade, ollamaChatModel)

	// 1. Setup gRPC server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", grpcPort))
	if err != nil {
		log.Fatalf("failed to listen on gRPC port: %v", err)
	}

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(grpcinterceptor.UnaryAuth(authMgr)),
		grpc.ChainStreamInterceptor(grpcinterceptor.StreamAuth(authMgr)),
	)
	memoryServer := grpcserver.NewMemoryServer(mgr)
	memorypb.RegisterMemoryServiceServer(grpcServer, memoryServer)

	skillServer := grpcserver.NewSkillServer(skillFacade)
	skillpb.RegisterSkillServiceServer(grpcServer, skillServer)

	grpcChatServer := grpcserver.NewChatServer(chatMgr)
	chatpb.RegisterChatServiceServer(grpcServer, grpcChatServer)

	grpcReviewServer := grpcserver.NewReviewServer(reviewMgr)
	reviewpb.RegisterReviewServiceServer(grpcServer, grpcReviewServer)

	grpcDebugServer := grpcserver.NewDebugServer(debugMgr)
	debugpb.RegisterDebugServiceServer(grpcServer, grpcDebugServer)

	reflection.Register(grpcServer)

	go func() {
		log.Printf("coder-node gRPC server listening at %v", lis.Addr())
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("gRPC server failed: %v", err)
		}
	}()

	// 2. Setup HTTP server
	httpMux := http.NewServeMux()

	// Health endpoint (always public)
	httpMux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","secure_mode":%v}`, authMgr.IsSecureMode())
	})

	httpMemoryServer := httpserver.NewMemoryServer(mgr)
	httpMemoryServer.RegisterHandlers(httpMux)

	httpSkillServer := httpserver.NewSkillServer(skillFacade)
	httpSkillServer.RegisterHandlers(httpMux)

	// Auth endpoints
	httpAuthServer := httpserver.NewAuthServer(authMgr)
	httpAuthServer.RegisterHandlers(httpMux)

	// Chat endpoints (Phase 1 — LLM Backbone)
	httpChatServer := httpserver.NewChatServer(chatMgr)
	httpChatServer.RegisterHandlers(httpMux)

	// Review endpoint (Phase 3)
	httpReviewServer := httpserver.NewReviewServer(reviewMgr)
	httpReviewServer.RegisterHandlers(httpMux)

	// Debug endpoint (Phase 6)
	httpDebugServer := httpserver.NewDebugServer(debugMgr)
	httpDebugServer.RegisterHandlers(httpMux)

	// Dashboard UI
	dashboardServer := dashboard.NewDashboardServer(authMgr, version.Version)
	dashboardServer.RegisterHandlers(httpMux)

	// Wrap entire mux with auth middleware
	var handler http.Handler = httpMux
	handler = httpmiddleware.Auth(authMgr)(handler)

	log.Printf("coder-node HTTP server listening at :%s (secure_mode=%v, chat_model=%s)", httpPort, authMgr.IsSecureMode(), ollamaChatModel)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", httpPort), handler); err != nil {
		log.Fatalf("HTTP server failed: %v", err)
	}
}
