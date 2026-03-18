package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func runLogin(_ []string) {
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

	// Ask if server is in secure mode and attempt client registration
	fmt.Print("\nDoes this server require authentication? (y/N): ")
	var authChoice string
	fmt.Scanln(&authChoice)

	if strings.ToLower(strings.TrimSpace(authChoice)) == "y" {
		httpBase := baseURL
		if !strings.HasPrefix(httpBase, "http://") && !strings.HasPrefix(httpBase, "https://") {
			httpBase = "http://" + httpBase
		}

		fmt.Print("Enter bootstrap token (provided by the server admin at startup): ")
		var bootstrapToken string
		fmt.Scanln(&bootstrapToken)
		bootstrapToken = strings.TrimSpace(bootstrapToken)

		// Collect git identity
		gitName := gitOutput("config", "--get", "user.name")
		gitEmail := gitOutput("config", "--get", "user.email")

		if gitEmail == "" {
			fmt.Print("Enter your git email (used to identify this client): ")
			fmt.Scanln(&gitEmail)
			gitEmail = strings.TrimSpace(gitEmail)
		}
		if gitName == "" {
			fmt.Print("Enter your git name [anonymous]: ")
			fmt.Scanln(&gitName)
			gitName = strings.TrimSpace(gitName)
			if gitName == "" {
				gitName = "anonymous"
			}
		}

		rawToken, regErr := registerClient(httpBase, bootstrapToken, gitName, gitEmail)
		if regErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: client registration failed: %v\n", regErr)
			fmt.Println("  Continuing without authentication — server may reject requests.")
		} else {
			cfg.Auth.AccessToken = rawToken
			fmt.Printf("\nClient registered as %s.\n", gitEmail)
			fmt.Println("Access token saved to config. Keep it safe — it will not be shown again.")
		}
	}

	data, _ := json.MarshalIndent(cfg, "", "  ")
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing config file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nConfiguration saved to %s\n", configPath)
	fmt.Printf("  Protocol: %s\n", protocol)
	fmt.Printf("  URL     : %s\n\n", baseURL)

	fmt.Println("Verifying connection...")
	mgr := getMemoryManager()
	defer mgr.Close()

	if _, err := mgr.List(context.Background(), 1, 0); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Verification failed: %v\n", err)
		fmt.Println("  Check if your coder-node is running and accessible.")
	} else {
		fmt.Println("Verification successful.")
	}
}

// registerClient calls POST /v1/auth/register-client and returns the raw access token.
func registerClient(httpBase, bootstrapToken, gitName, gitEmail string) (string, error) {
	payload, _ := json.Marshal(map[string]string{
		"bootstrap_token": bootstrapToken,
		"git_name":        gitName,
		"git_email":       gitEmail,
	})

	resp, err := http.Post(httpBase+"/v1/auth/register-client", "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return "", fmt.Errorf("could not reach server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody map[string]any
		json.NewDecoder(resp.Body).Decode(&errBody)
		return "", fmt.Errorf("server returned HTTP %d", resp.StatusCode)
	}

	var result struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}
	return result.AccessToken, nil
}

