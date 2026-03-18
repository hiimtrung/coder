package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/trungtran/coder/api/grpc/memorypb"
	"github.com/trungtran/coder/api/grpc/skillpb"
	"github.com/trungtran/coder/internal/infra/embedding"
	"github.com/trungtran/coder/internal/infra/postgres"
	grpcserver "github.com/trungtran/coder/internal/transport/grpc/server"
	httpserver "github.com/trungtran/coder/internal/transport/http/server"
	ucmemory "github.com/trungtran/coder/internal/usecase/memory"
	ucskill "github.com/trungtran/coder/internal/usecase/skill"
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

	// Initialize Postgres with shared DB handle
	memDB, rawDB, err := postgres.NewPostgresMemoryWithDB(dsn)
	if err != nil {
		log.Fatalf("Failed to initialize postgres: %v", err)
	}

	// Initialize Ollama provider
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

	// 1. Setup gRPC server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", grpcPort))
	if err != nil {
		log.Fatalf("failed to listen on gRPC port: %v", err)
	}

	grpcServer := grpc.NewServer()
	memoryServer := grpcserver.NewMemoryServer(mgr)
	memorypb.RegisterMemoryServiceServer(grpcServer, memoryServer)

	skillServer := grpcserver.NewSkillServer(skillFacade)
	skillpb.RegisterSkillServiceServer(grpcServer, skillServer)

	reflection.Register(grpcServer)

	go func() {
		log.Printf("coder-node gRPC server listening at %v", lis.Addr())
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("gRPC server failed: %v", err)
		}
	}()

	// 2. Setup HTTP server
	httpMux := http.NewServeMux()
	httpMemoryServer := httpserver.NewMemoryServer(mgr)
	httpMemoryServer.RegisterHandlers(httpMux)

	httpSkillServer := httpserver.NewSkillServer(skillFacade)
	httpSkillServer.RegisterHandlers(httpMux)

	log.Printf("coder-node HTTP server listening at :%s", httpPort)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", httpPort), httpMux); err != nil {
		log.Fatalf("HTTP server failed: %v", err)
	}
}
