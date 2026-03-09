package main

import (
	"fmt"
	"log"
	"net"
	"os"

	"github.com/trungtran/coder/api/grpc/memorypb"
	"github.com/trungtran/coder/internal/grpcserver"
	"github.com/trungtran/coder/internal/memory"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	port := os.Getenv("GRPC_PORT")
	if port == "" {
		port = "50051"
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

	// Initialize Postgres memory db
	db, err := memory.NewPostgresMemory(dsn)
	if err != nil {
		log.Fatalf("Failed to initialize postgres: %v", err)
	}

	// Initialize Ollama provider
	provider := &memory.OllamaEmbeddingProvider{
		BaseURL: ollamaBase,
		Model:   ollamaModel,
	}

	// Initialize memory manager
	mgr := memory.NewManager(db, provider)
	defer mgr.Close()

	// Setup gRPC server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	memoryServer := grpcserver.NewMemoryServer(mgr)
	memorypb.RegisterMemoryServiceServer(grpcServer, memoryServer)
	reflection.Register(grpcServer)

	log.Printf("coder-node grpc server listening at %v", lis.Addr())
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
