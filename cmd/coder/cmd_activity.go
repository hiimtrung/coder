package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os/exec"
	"strings"
)

// logActivity sends a fire-and-forget activity log to coder-node.
// It does not block and silently ignores all errors.
func logActivity(command string) {
	cfg, err := loadConfig()
	if err != nil || cfg.Auth.AccessToken == "" || cfg.Memory.BaseURL == "" {
		return
	}
	go func() {
		repo := gitOutput("config", "--get", "remote.origin.url")
		branch := gitOutput("rev-parse", "--abbrev-ref", "HEAD")

		baseURL := cfg.Memory.BaseURL
		if !strings.HasPrefix(baseURL, "http") {
			baseURL = "http://" + baseURL
		}

		payload, _ := json.Marshal(map[string]string{
			"command": command,
			"repo":    sanitiseRepoURL(repo),
			"branch":  branch,
		})
		req, err := http.NewRequest("POST", baseURL+"/v1/auth/log-activity", bytes.NewBuffer(payload))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+cfg.Auth.AccessToken)
		http.DefaultClient.Do(req) //nolint:errcheck
	}()
}

// gitOutput runs a git command and returns trimmed stdout, empty on error.
func gitOutput(args ...string) string {
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// sanitiseRepoURL strips credentials from git URLs.
func sanitiseRepoURL(url string) string {
	// Remove git@ prefix: git@github.com:org/repo.git -> github.com/org/repo
	if idx := strings.Index(url, "@"); idx != -1 {
		after := url[idx+1:]
		after = strings.ReplaceAll(after, ":", "/")
		after = strings.TrimSuffix(after, ".git")
		return after
	}
	// Remove https://user:pass@ or https:// prefix
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	return strings.TrimSuffix(url, ".git")
}
