package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const tokenUsage = `coder token — Manage your coder-node access token

USAGE:
  coder token <subcommand>

SUBCOMMANDS:
  show     Display current token info (masked) and client identity
  rotate   Generate a new access token, invalidating the current one

EXAMPLES:
  coder token show      # Show current token and client identity
  coder token rotate    # Rotate your access token (saves new token to config)
`

func runToken(args []string) {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		fmt.Print(tokenUsage)
		return
	}

	switch args[0] {
	case "show":
		runTokenShow()
	case "rotate":
		runTokenRotate()
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown token subcommand %q\n\n", args[0])
		fmt.Print(tokenUsage)
		os.Exit(1)
	}
}

// runTokenShow prints the currently configured token (masked) and calls /v1/auth/me.
func runTokenShow() {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to load config: %v\n", err)
		os.Exit(1)
	}

	if cfg.Auth.AccessToken == "" {
		fmt.Println("No access token configured.")
		fmt.Println("Run 'coder login' to connect to a coder-node and register.")
		return
	}

	masked := maskToken(cfg.Auth.AccessToken)
	fmt.Printf("Access token : %s\n", masked)

	// Fetch identity from server if a base URL is configured
	baseURL := cfg.Memory.BaseURL
	if baseURL == "" {
		fmt.Println("(No server URL configured — cannot fetch client identity)")
		return
	}

	httpBase := toHTTPBase(baseURL)
	req, err := http.NewRequest(http.MethodGet, httpBase+"/v1/auth/me", nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}
	req.Header.Set("Authorization", "Bearer "+cfg.Auth.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error contacting server: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "Server returned %d — check your token or server URL.\n", resp.StatusCode)
		return
	}

	var info struct {
		ID          string `json:"id"`
		GitName     string `json:"git_name"`
		GitEmail    string `json:"git_email"`
		CreatedAt   string `json:"created_at"`
		LastSeenAt  string `json:"last_seen_at"`
		SecureMode  bool   `json:"secure_mode"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing response: %v\n", err)
		return
	}

	fmt.Printf("Client ID    : %s\n", info.ID)
	fmt.Printf("Name         : %s\n", info.GitName)
	fmt.Printf("Email        : %s\n", info.GitEmail)
	fmt.Printf("Registered   : %s\n", info.CreatedAt)
	fmt.Printf("Last seen    : %s\n", info.LastSeenAt)
	fmt.Printf("Secure mode  : %v\n", info.SecureMode)
}

// runTokenRotate calls POST /v1/auth/token/rotate and saves the new token.
func runTokenRotate() {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to load config: %v\n", err)
		os.Exit(1)
	}

	if cfg.Auth.AccessToken == "" {
		fmt.Fprintln(os.Stderr, "Error: no access token configured. Run 'coder login' first.")
		os.Exit(1)
	}

	baseURL := cfg.Memory.BaseURL
	if baseURL == "" {
		fmt.Fprintln(os.Stderr, "Error: no server URL configured. Run 'coder login' first.")
		os.Exit(1)
	}

	httpBase := toHTTPBase(baseURL)

	fmt.Println("Rotating access token...")
	req, err := http.NewRequest(http.MethodPost, httpBase+"/v1/auth/token/rotate", bytes.NewReader([]byte("{}")))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	req.Header.Set("Authorization", "Bearer "+cfg.Auth.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error contacting server: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody map[string]any
		json.NewDecoder(resp.Body).Decode(&errBody)
		fmt.Fprintf(os.Stderr, "Error: server returned %d\n", resp.StatusCode)
		if e, ok := errBody["error"].(map[string]any); ok {
			fmt.Fprintf(os.Stderr, "  %s: %s\n", e["code"], e["message"])
		}
		os.Exit(1)
	}

	var result struct {
		AccessToken string `json:"access_token"`
		Message     string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing response: %v\n", err)
		os.Exit(1)
	}

	// Save new token to config
	cfg.Auth.AccessToken = result.AccessToken
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".coder", "config.json")
	data, _ := json.MarshalIndent(cfg, "", "  ")
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to save new token: %v\n", err)
		fmt.Printf("\nNew token (save this manually): %s\n", result.AccessToken)
		os.Exit(1)
	}

	fmt.Printf("\n✓ %s\n", result.Message)
	fmt.Printf("  New token : %s\n", maskToken(result.AccessToken))
	fmt.Printf("  Saved to  : %s\n", configPath)
}

// maskToken shows the first 8 and last 4 characters of a token.
func maskToken(token string) string {
	if len(token) <= 12 {
		return strings.Repeat("*", len(token))
	}
	return token[:8] + strings.Repeat("*", len(token)-12) + token[len(token)-4:]
}

// toHTTPBase converts a gRPC host:port URL to an HTTP base URL.
// e.g. "localhost:50051" → "http://localhost:8080"
// e.g. "http://myhost:8080" → "http://myhost:8080"
func toHTTPBase(baseURL string) string {
	if strings.HasPrefix(baseURL, "http://") || strings.HasPrefix(baseURL, "https://") {
		return strings.TrimRight(baseURL, "/")
	}
	// Assume it's a raw host:port (gRPC style).
	// Try to derive the HTTP port from env or use a conventional offset.
	httpPort := os.Getenv("CODER_NODE_HTTP_URL")
	if httpPort != "" {
		return strings.TrimRight(httpPort, "/")
	}
	// Default: replace port 50051→8080, otherwise prepend http://
	host := baseURL
	if idx := strings.LastIndex(host, ":"); idx >= 0 {
		port := host[idx+1:]
		if port == "50051" {
			return "http://" + host[:idx] + ":8080"
		}
	}
	return "http://" + host
}
