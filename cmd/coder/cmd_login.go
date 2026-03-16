package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func runLogin(args []string) {
	fmt.Println("=== coder-node Configuration ===")
	fmt.Println("Choose protocol:")
	fmt.Println("  1) gRPC (recommended)")
	fmt.Println("  2) HTTP")
	fmt.Print("Selection [1]: ")

	var choice string
	fmt.Scanln(&choice)

	protocol := "grpc"
	defaultURL := "localhost:50051"
	if choice == "2" {
		protocol = "http"
		defaultURL = "localhost:8080"
	}

	fmt.Printf("Enter coder-node %s URL [%s]: ", protocol, defaultURL)
	var baseURL string
	fmt.Scanln(&baseURL)
	if baseURL == "" {
		baseURL = defaultURL
	}

	home, _ := os.UserHomeDir()
	configDir := filepath.Join(home, ".coder")
	configPath := filepath.Join(configDir, "config.json")

	if err := os.MkdirAll(configDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating config directory: %v\n", err)
		os.Exit(1)
	}

	cfg, err := loadConfig()
	if err != nil {
		cfg = &Config{}
	}

	cfg.Memory.Provider = "remote"
	cfg.Memory.Protocol = protocol
	cfg.Memory.BaseURL = baseURL

	data, _ := json.MarshalIndent(cfg, "", "  ")
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing config file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n✓ Configuration saved to %s\n", configPath)
	fmt.Printf("  Protocol: %s\n", protocol)
	fmt.Printf("  URL     : %s\n\n", baseURL)

	fmt.Println("Verifying connection...")
	mgr := getMemoryManager()
	defer mgr.Close()

	if _, err := mgr.List(context.Background(), 1, 0); err != nil {
		fmt.Fprintf(os.Stderr, "⚠ Warning: Verification failed: %v\n", err)
		fmt.Println("  Check if your coder-node is running and accessible.")
	} else {
		fmt.Println("✓ Verification successful.")
	}
}
