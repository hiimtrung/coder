package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	memdomain "github.com/trungtran/coder/internal/domain/memory"
	skilldomain "github.com/trungtran/coder/internal/domain/skill"
	"github.com/trungtran/coder/internal/infra/embedding"
	"github.com/trungtran/coder/internal/infra/postgres"
	grpcclient "github.com/trungtran/coder/internal/transport/grpc/client"
	httpclient "github.com/trungtran/coder/internal/transport/http/client"
	ucmemory "github.com/trungtran/coder/internal/usecase/memory"
)

// Config represents the coder configuration stored at ~/.coder/config.json.
type Config struct {
	Memory struct {
		Provider     string `json:"provider"`
		Protocol     string `json:"protocol"` // grpc or http
		DatabaseType string `json:"database_type"`
		APIKey       string `json:"api_key"`
		BaseURL      string `json:"base_url"`
		Model        string `json:"model"`
		PostgresDSN  string `json:"postgres_dsn"`
	} `json:"memory"`
	Auth struct {
		AccessToken string `json:"access_token"` // raw token saved after registration
	} `json:"auth"`
}

func loadConfig() (*Config, error) {
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".coder", "config.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func getMemoryManager() memdomain.MemoryManager {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load config: %v\n", err)
		cfg = &Config{}
	}

	providerType := cfg.Memory.Provider
	if providerType == "" {
		providerType = "openai"
	}

	if providerType == "grpc" || providerType == "remote" {
		baseURL := cfg.Memory.BaseURL
		if baseURL == "" {
			baseURL = os.Getenv("CODER_NODE_URL")
		}
		if baseURL == "" {
			fmt.Fprintf(os.Stderr, "Error: base_url is required for %s provider (or set CODER_NODE_URL)\n", providerType)
			os.Exit(1)
		}

		protocol := cfg.Memory.Protocol
		if protocol == "" {
			if providerType == "grpc" {
				protocol = "grpc"
			} else {
				protocol = "http"
			}
		}

		if protocol == "grpc" {
			client, err := grpcclient.NewClient(baseURL)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: failed to connect to coder-node (gRPC): %v\n", err)
				os.Exit(1)
			}
			return client
		} else {
			client, err := httpclient.NewClient(baseURL, cfg.Auth.AccessToken)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: failed to connect to coder-node (HTTP): %v\n", err)
				os.Exit(1)
			}
			return client
		}
	}

	var db memdomain.MemoryRepository

	dbType := cfg.Memory.DatabaseType
	if dbType == "" {
		dbType = "postgres"
	}

	if dbType == "postgres" {
		dsn := cfg.Memory.PostgresDSN
		if dsn == "" {
			dsn = os.Getenv("POSTGRES_DSN")
		}
		if dsn == "" {
			fmt.Fprintf(os.Stderr, "Error: postgres_dsn is not configured\n")
			os.Exit(1)
		}
		pgdb, err := postgres.NewPostgresMemory(dsn)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to open postgres database: %v\n", err)
			os.Exit(1)
		}
		db = pgdb
	} else {
		fmt.Fprintf(os.Stderr, "Error: unsupported database type. Only postgres is supported.\n")
		os.Exit(1)
	}

	var provider memdomain.EmbeddingProvider

	if providerType == "ollama" {
		baseURL := cfg.Memory.BaseURL
		if baseURL == "" {
			baseURL = os.Getenv("OLLAMA_BASE_URL")
		}
		if baseURL == "" {
			fmt.Fprintf(os.Stderr, "Error: ollama base_url is not configured. Local ollama is not supported.\n")
			os.Exit(1)
		}
		model := cfg.Memory.Model
		if model == "" {
			model = os.Getenv("OLLAMA_EMBEDDING_MODEL")
		}
		if model == "" {
			model = "mxbai-embed-large" // dimension 1024
		}
		provider = &embedding.OllamaEmbeddingProvider{
			BaseURL: baseURL,
			Model:   model,
		}
	} else {
		apiKey := cfg.Memory.APIKey
		if apiKey == "" {
			apiKey = os.Getenv("OPENAI_API_KEY")
		}

		baseURL := cfg.Memory.BaseURL
		if baseURL == "" {
			baseURL = os.Getenv("OPENAI_BASE_URL")
		}

		model := cfg.Memory.Model
		if model == "" {
			model = os.Getenv("OPENAI_EMBEDDING_MODEL")
		}
		if model == "" {
			model = "text-embedding-3-small"
		}

		provider = &embedding.OpenAIEmbeddingProvider{
			APIKey:  apiKey,
			BaseURL: baseURL,
			Model:   model,
		}
	}

	return ucmemory.NewManager(db, provider)
}

func getSkillClient() skilldomain.SkillClient {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load config: %v\n", err)
		cfg = &Config{}
	}

	baseURL := cfg.Memory.BaseURL
	if baseURL == "" {
		baseURL = os.Getenv("CODER_NODE_URL")
	}
	if baseURL == "" {
		fmt.Fprintf(os.Stderr, "Error: base_url is required to use skills. Run 'coder login' or set CODER_NODE_URL.\n")
		os.Exit(1)
	}

	protocol := cfg.Memory.Protocol
	if protocol == "" {
		protocol = "grpc"
	}

	if protocol == "grpc" {
		client, err := grpcclient.NewSkillClient(baseURL)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to connect to coder-node skill service (gRPC): %v\n", err)
			os.Exit(1)
		}
		return client
	} else {
		client, err := httpclient.NewSkillClient(baseURL, cfg.Auth.AccessToken)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to connect to coder-node skill service (HTTP): %v\n", err)
			os.Exit(1)
		}
		return client
	}
}

// resolveTargetDir returns the given flag value, or the current working directory.
func resolveTargetDir(flag string) string {
	if flag != "" {
		return flag
	}
	dir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to get current directory: %v\n", err)
		os.Exit(1)
	}
	return dir
}

// truncate shortens a string to max characters, adding "..." if truncated.
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
