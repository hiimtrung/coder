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

	grpcclient "github.com/trungtran/coder/internal/transport/grpc/client"
)

func runLogin(_ []string) {
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

	// ── Main retry loop ───────────────────────────────────────────────────────
loop:
	for {
		cfg = configureConnection(cfg)

		// Persist config before verification so even a failed verify
		// leaves the user with a working config they can tweak.
		data, _ := json.MarshalIndent(cfg, "", "  ")
		if writeErr := os.WriteFile(configPath, data, 0644); writeErr != nil {
			fmt.Fprintf(os.Stderr, "Error writing config file: %v\n", writeErr)
			os.Exit(1)
		}

		fmt.Printf("\nConfiguration saved to %s\n", configPath)
		fmt.Printf("  Protocol : %s\n", cfg.Memory.Protocol)
		fmt.Printf("  URL      : %s\n", cfg.Memory.BaseURL)
		if cfg.Auth.AccessToken != "" {
			fmt.Println("  Auth     : token configured ✓")
		}
		fmt.Println()

		// ── Verify connection ─────────────────────────────────────────────────
		fmt.Println("Verifying connection to coder-node...")
		mgr := getMemoryManager()
		_, verifyErr := mgr.List(context.Background(), 1, 0)
		mgr.Close()

		if verifyErr == nil {
			fmt.Println("✓ Connection successful.")
			break loop
		}

		// Classify the error for a helpful message
		errMsg := verifyErr.Error()
		switch {
		case strings.Contains(errMsg, "401") || strings.Contains(errMsg, "Unauthorized") || strings.Contains(errMsg, "unauthorized"):
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "✗ Authentication failed (HTTP 401).")
			fmt.Fprintln(os.Stderr, "  The server is running in secure mode but the request was rejected.")
			fmt.Fprintln(os.Stderr, "  Possible causes:")
			fmt.Fprintln(os.Stderr, "    • The bootstrap token was incorrect — registration did not complete.")
			fmt.Fprintln(os.Stderr, "    • The access token in your config is missing, revoked, or stale.")
			fmt.Fprintln(os.Stderr, "    • Your access token has been revoked.")

		case strings.Contains(errMsg, "connection refused") || strings.Contains(errMsg, "no such host") ||
			strings.Contains(errMsg, "Unavailable") || strings.Contains(errMsg, "dial"):
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "✗ Could not reach coder-node.")
			fmt.Fprintln(os.Stderr, "  Check that it is running:")
			fmt.Fprintln(os.Stderr, "    docker ps | grep coder_node")
			fmt.Fprintln(os.Stderr, "  If not started yet:")
			fmt.Fprintln(os.Stderr, "    curl -fsSL https://raw.githubusercontent.com/hiimtrung/coder/main/install-node.sh | sh")
			fmt.Fprintln(os.Stderr, "  Or with secure mode:")
			fmt.Fprintln(os.Stderr, "    curl -fsSL https://raw.githubusercontent.com/hiimtrung/coder/main/install-node.sh | sh -s -- --secure")

		default:
			fmt.Fprintf(os.Stderr, "\n✗ Verification failed: %v\n", verifyErr)
		}

		fmt.Println()
		fmt.Print("What would you like to do?\n")
		fmt.Println("  1) Retry with a different URL / protocol")
		fmt.Println("  2) Re-enter authentication token")
		fmt.Println("  3) Skip verification and continue anyway")
		fmt.Println("  4) Exit")
		fmt.Print("Selection [1]: ")

		var pick string
		fmt.Scanln(&pick)
		switch strings.TrimSpace(pick) {
		case "2":
			cfg = registerAuth(cfg)
			// persist updated token and re-verify
		case "3":
			fmt.Println("Skipping verification. Run 'coder login' anytime to reconfigure.")
			break loop
		case "4":
			fmt.Println("Exiting. Your partial config was saved — run 'coder login' to finish setup.")
			return
		default:
			// "1" or anything else — re-run the full wizard
		}
	}

	fmt.Println()
	fmt.Println("Setup complete. Try:")
	fmt.Println("  coder skill ingest --source local")
	fmt.Println("  coder skill search \"topic\"")
}

// configureConnection prompts for protocol, URL, and optionally auth.
func configureConnection(cfg *Config) *Config {
	fmt.Println("=== coder-node Configuration ===")
	fmt.Println()
	fmt.Println("Choose protocol:")
	fmt.Println("  1) gRPC  — recommended (faster, lower overhead)")
	fmt.Println("  2) HTTP  — use this if your deployment exposes HTTP only")
	fmt.Print("Selection [1]: ")

	var choice string
	fmt.Scanln(&choice)

	protocol := "grpc"
	defaultURL := "localhost:50051"
	if strings.TrimSpace(choice) == "2" {
		protocol = "http"
		defaultURL = "localhost:8080"
	}

	fmt.Printf("Enter coder-node %s URL [%s]: ", protocol, defaultURL)
	var baseURL string
	fmt.Scanln(&baseURL)
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultURL
	}

	cfg.Memory.Provider = "remote"
	cfg.Memory.Protocol = protocol
	cfg.Memory.BaseURL = baseURL

	return registerAuth(cfg)
}

// registerAuth asks whether the server needs auth and handles the bootstrap flow.
func registerAuth(cfg *Config) *Config {
	fmt.Println()
	fmt.Println("Does this server run in secure mode (--secure)?")
	fmt.Print("Requires authentication? (y/N): ")

	var authChoice string
	fmt.Scanln(&authChoice)

	if strings.ToLower(strings.TrimSpace(authChoice)) != "y" {
		cfg.Auth.AccessToken = ""
		return cfg
	}

	fmt.Print("Enter bootstrap token (from server logs): ")
	var bootstrapToken string
	fmt.Scanln(&bootstrapToken)
	bootstrapToken = strings.TrimSpace(bootstrapToken)

	if bootstrapToken == "" {
		fmt.Fprintln(os.Stderr, "Warning: bootstrap token is empty — skipping registration.")
		return cfg
	}

	// Collect git identity
	gitName := gitOutput("config", "--get", "user.name")
	gitEmail := gitOutput("config", "--get", "user.email")

	if gitEmail == "" {
		fmt.Print("Enter your git email (identifies this machine): ")
		fmt.Scanln(&gitEmail)
		gitEmail = strings.TrimSpace(gitEmail)
	}
	if gitName == "" {
		fmt.Print("Enter your name [anonymous]: ")
		fmt.Scanln(&gitName)
		gitName = strings.TrimSpace(gitName)
		if gitName == "" {
			gitName = "anonymous"
		}
	}

	fmt.Printf("Registering client as %s <%s>...\n", gitName, gitEmail)

	rawToken, regErr := registerClient(cfg, bootstrapToken, gitName, gitEmail)
	if regErr != nil {
		fmt.Fprintf(os.Stderr, "✗ Registration failed: %v\n", regErr)
		fmt.Fprintln(os.Stderr, "  Check the bootstrap token and that the selected coder-node endpoint is correct and reachable.")
		fmt.Fprintln(os.Stderr, "  You can retry registration by selecting option 2 after verification fails.")
	} else {
		cfg.Auth.AccessToken = rawToken
		fmt.Printf("✓ Registered as %s — access token saved.\n", gitEmail)
	}

	return cfg
}

func registerClient(cfg *Config, bootstrapToken, gitName, gitEmail string) (string, error) {
	if cfg != nil && cfg.Memory.Protocol == "grpc" {
		client, err := grpcclient.NewAuthClient(cfg.Memory.BaseURL, "")
		if err != nil {
			return "", fmt.Errorf("could not reach gRPC server: %w", err)
		}
		defer client.Close()

		rawToken, _, err := client.RegisterClient(context.Background(), bootstrapToken, gitName, gitEmail)
		return rawToken, err
	}

	httpBase := ""
	if cfg != nil {
		httpBase = toHTTPBase(cfg.Memory.BaseURL)
	}
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
		if msg, ok := errBody["error"].(string); ok && msg != "" {
			return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, msg)
		}
		if resp.StatusCode == http.StatusNotFound {
			return "", fmt.Errorf("HTTP 404: register-client endpoint not found at %s", httpBase)
		}
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
