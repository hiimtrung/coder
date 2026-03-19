package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os/exec"
	"strings"
)

// logActivity sends a fire-and-forget activity log to coder-node.
// It collects rich git context from the working directory and posts
// asynchronously — all errors are silently ignored so the CLI never blocks.
func logActivity(command string) {
	cfg, err := loadConfig()
	if err != nil || cfg.Auth.AccessToken == "" || cfg.Memory.BaseURL == "" {
		return
	}
	go func() {
		repo    := sanitiseRepoURL(gitOutput("config", "--get", "remote.origin.url"))
		branch  := gitOutput("rev-parse", "--abbrev-ref", "HEAD")
		commit  := gitOutput("rev-parse", "--short", "HEAD")
		project := gitProjectName()

		httpBase := toHTTPBase(cfg.Memory.BaseURL)

		payload, _ := json.Marshal(map[string]string{
			"command": command,
			"repo":    repo,
			"branch":  branch,
			"commit":  commit,
			"project": project,
		})
		req, err := http.NewRequest("POST", httpBase+"/v1/auth/log-activity", bytes.NewBuffer(payload))
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

// gitProjectName returns the basename of the git repository root directory.
// e.g. /Users/dev/my-app → "my-app". Returns "" if not in a git repo.
func gitProjectName() string {
	root := gitOutput("rev-parse", "--show-toplevel")
	if root == "" {
		return ""
	}
	// Use path/filepath.Base-equivalent without importing path
	for i := len(root) - 1; i >= 0; i-- {
		if root[i] == '/' || root[i] == '\\' {
			return root[i+1:]
		}
	}
	return root
}

// sanitiseRepoURL strips credentials and normalises git remote URLs.
//
//	git@github.com:org/repo.git  → github.com/org/repo
//	https://user:pass@host/repo  → host/repo
func sanitiseRepoURL(rawURL string) string {
	// git@ SSH format
	if idx := strings.Index(rawURL, "@"); idx != -1 {
		after := rawURL[idx+1:]
		after = strings.ReplaceAll(after, ":", "/")
		return strings.TrimSuffix(after, ".git")
	}
	// HTTPS — strip scheme and optional user:pass@
	rawURL = strings.TrimPrefix(rawURL, "https://")
	rawURL = strings.TrimPrefix(rawURL, "http://")
	if idx := strings.Index(rawURL, "@"); idx != -1 {
		rawURL = rawURL[idx+1:]
	}
	return strings.TrimSuffix(rawURL, ".git")
}
