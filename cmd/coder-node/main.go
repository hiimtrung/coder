package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	authpb "github.com/trungtran/coder/api/grpc/authpb/auth"
	"github.com/trungtran/coder/api/grpc/memorypb"
	"github.com/trungtran/coder/api/grpc/skillpb"
	authdomain "github.com/trungtran/coder/internal/domain/auth"
	"github.com/trungtran/coder/internal/infra/embedding"
	"github.com/trungtran/coder/internal/infra/postgres"
	grpcinterceptor "github.com/trungtran/coder/internal/transport/grpc/interceptor"
	grpcserver "github.com/trungtran/coder/internal/transport/grpc/server"
	httpmiddleware "github.com/trungtran/coder/internal/transport/http/middleware"
	httpserver "github.com/trungtran/coder/internal/transport/http/server"
	dashboard "github.com/trungtran/coder/internal/transport/http/server/dashboard"
	ucauth "github.com/trungtran/coder/internal/usecase/auth"
	ucmemory "github.com/trungtran/coder/internal/usecase/memory"
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

	// -------------------------------------------------------------------------
	// Embedding provider — powers semantic (vector) search for memory and skills.
	//
	// Supported providers (set EMBEDDING_PROVIDER):
	//   ollama  — Local Ollama embedding model (DEFAULT).
	//             OLLAMA_BASE_URL   default: http://localhost:11434
	//             OLLAMA_EMBEDDING_MODEL  default: mxbai-embed-large (dim 1024)
	//   openai  — OpenAI text-embedding-3-small or any OpenAI-compat endpoint.
	//             EMBEDDING_API_KEY  required
	//             EMBEDDING_BASE_URL optional (default: https://api.openai.com/v1)
	//             EMBEDDING_MODEL    optional (default: text-embedding-3-small)
	//   none    — FTS-only mode (no vectors, keyword search only)
	// -------------------------------------------------------------------------
	var embeddingProvider interface {
		GenerateEmbedding(ctx context.Context, text string) ([]float32, error)
	}

	embeddingProviderName := os.Getenv("EMBEDDING_PROVIDER")
	if embeddingProviderName == "" {
		embeddingProviderName = "ollama" // default: local Ollama
	}

	switch embeddingProviderName {
	case "ollama":
		baseURL := os.Getenv("OLLAMA_BASE_URL")
		if baseURL == "" {
			baseURL = "http://localhost:11434"
		}
		model := os.Getenv("OLLAMA_EMBEDDING_MODEL")
		if model == "" {
			model = "mxbai-embed-large"
		}
		embeddingProvider = &embedding.OllamaEmbeddingProvider{
			BaseURL: baseURL,
			Model:   model,
		}
		log.Printf("Embedding: ollama model=%s base=%s", model, baseURL)

	case "openai":
		model := os.Getenv("EMBEDDING_MODEL")
		if model == "" {
			model = "text-embedding-3-small"
		}
		dimensions := 1024 // match existing vector(1024) schema
		if d := os.Getenv("EMBEDDING_DIMENSIONS"); d != "" {
			fmt.Sscanf(d, "%d", &dimensions)
		}
		embeddingProvider = &embedding.OpenAIEmbeddingProvider{
			APIKey:     os.Getenv("EMBEDDING_API_KEY"),
			Model:      model,
			BaseURL:    os.Getenv("EMBEDDING_BASE_URL"),
			Dimensions: dimensions,
		}
		log.Printf("Embedding: openai model=%s dimensions=%d", model, dimensions)

	default:
		log.Printf("Embedding: FTS-only mode (set EMBEDDING_PROVIDER=ollama or openai to enable vector search)")
	}

	// Initialize Postgres with shared DB handle
	memDB, rawDB, err := postgres.NewPostgresMemoryWithDB(dsn)
	if err != nil {
		log.Fatalf("Failed to initialize postgres: %v", err)
	}

	// Initialize memory manager (use case)
	mgr := ucmemory.NewManager(memDB, embeddingProvider)
	defer mgr.Close()

	// Initialize skill store (infrastructure, shares same DB connection)
	skillStore, err := postgres.NewPostgresSkillStore(rawDB)
	if err != nil {
		log.Fatalf("Failed to initialize skill store: %v", err)
	}

	// Initialize skill ingestor + facade (use case)
	skillIngestor := ucskill.NewIngestor(skillStore, embeddingProvider)
	skillFacade := ucskill.NewSkillFacade(skillIngestor, skillStore)

	// Auth setup
	ctx := context.Background()
	var authMgr authdomain.AuthManager

	if os.Getenv("SECURE_MODE") == "true" {
		authRepo, err := postgres.NewPostgresAuth(rawDB)
		if err != nil {
			log.Fatalf("Failed to initialize auth repository: %v", err)
		}
		authMgr = ucauth.NewManager(authRepo, true)

		bootstrapToken, err := authMgr.GetBootstrapToken(ctx)
		if err != nil {
			log.Printf("Warning: failed to generate bootstrap token: %v", err)
		} else if bootstrapToken != "" {
			fmt.Printf("\nBOOTSTRAP TOKEN (shown once): %s\n", bootstrapToken)
			fmt.Println("   Share this with clients so they can run: coder login")
			fmt.Println()
		}
	} else {
		authMgr = ucauth.NewManager(nil, false)
	}

	// 1. gRPC server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", grpcPort))
	if err != nil {
		log.Fatalf("failed to listen on gRPC port: %v", err)
	}

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(grpcinterceptor.UnaryAuth(authMgr)),
		grpc.ChainStreamInterceptor(grpcinterceptor.StreamAuth(authMgr)),
	)
	authpb.RegisterAuthServiceServer(grpcServer, grpcserver.NewAuthServer(authMgr))
	memorypb.RegisterMemoryServiceServer(grpcServer, grpcserver.NewMemoryServer(mgr))
	skillpb.RegisterSkillServiceServer(grpcServer, grpcserver.NewSkillServer(skillFacade))
	reflection.Register(grpcServer)

	go func() {
		log.Printf("coder-node gRPC server listening at %v", lis.Addr())
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("gRPC server failed: %v", err)
		}
	}()

	// 2. HTTP server
	httpMux := http.NewServeMux()

	httpMux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","secure_mode":%v}`, authMgr.IsSecureMode())
	})

	httpserver.NewMemoryServer(mgr).RegisterHandlers(httpMux)
	httpserver.NewSkillServer(skillFacade).RegisterHandlers(httpMux)
	httpserver.NewAuthServer(authMgr).RegisterHandlers(httpMux)
	dashboard.NewDashboardServer(authMgr, version.Version).RegisterHandlers(httpMux)

	var handler http.Handler = httpMux
	handler = httpmiddleware.Auth(authMgr)(handler)

	log.Printf("coder-node HTTP server listening at :%s (secure_mode=%v)", httpPort, authMgr.IsSecureMode())
	if err := http.ListenAndServe(fmt.Sprintf(":%s", httpPort), handler); err != nil {
		log.Fatalf("HTTP server failed: %v", err)
	}
}
