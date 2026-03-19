package httpserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	authdomain "github.com/trungtran/coder/internal/domain/auth"
)

// AuthServer handles authentication and client management endpoints.
type AuthServer struct {
	mgr authdomain.AuthManager
}

func NewAuthServer(mgr authdomain.AuthManager) *AuthServer {
	return &AuthServer{mgr: mgr}
}

func (s *AuthServer) RegisterHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/v1/auth/register-client", s.handleRegisterClient)
	mux.HandleFunc("/v1/auth/clients", s.handleListClients)
	mux.HandleFunc("/v1/auth/log-activity", s.handleLogActivity)
	mux.HandleFunc("/v1/auth/bootstrap/regenerate", s.handleRegenerateBootstrap)
	mux.HandleFunc("/v1/auth/bootstrap/status", s.handleBootstrapStatus)
	mux.HandleFunc("/v1/auth/token/rotate", s.handleTokenRotate)
	mux.HandleFunc("/v1/auth/me", s.handleMe)
}

// handleRegisterClient POST /v1/auth/register-client
func (s *AuthServer) handleRegisterClient(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		BootstrapToken string `json:"bootstrap_token"`
		GitName        string `json:"git_name"`
		GitEmail       string `json:"git_email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.GitEmail == "" {
		http.Error(w, `{"error":{"code":"VAL_MISSING_GIT_EMAIL","message":"git_email is required"}}`, http.StatusBadRequest)
		return
	}

	rawToken, err := s.mgr.RegisterClient(r.Context(), req.BootstrapToken, req.GitName, req.GitEmail)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":{"code":"AUTH_REGISTRATION_FAILED","message":"%s"}}`, err.Error()), http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"access_token": rawToken,
		"message":      fmt.Sprintf("Client registered as %s. Store this token safely — it will not be shown again.", req.GitEmail),
	})
}

// handleListClients GET /v1/auth/clients  (requires valid token)
func (s *AuthServer) handleListClients(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	clients, err := s.mgr.ListClients(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"clients": clients})
}

// handleRegenerateBootstrap POST /v1/auth/bootstrap/regenerate  (requires valid token)
// Invalidates the current bootstrap token hash and generates a brand-new one.
// Returns the raw token — show it to the admin, it will NOT be shown again.
func (s *AuthServer) handleRegenerateBootstrap(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	newToken, err := s.mgr.RegenerateBootstrapToken(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":{"code":"AUTH_REGENERATE_FAILED","message":"%s"}}`, err.Error()), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"bootstrap_token": newToken,
		"message":         "Bootstrap token regenerated. Share this with new developers — it will not be shown again.",
	})
}

// handleBootstrapStatus GET /v1/auth/bootstrap/status  (requires valid token)
// Returns whether a bootstrap token hash is currently configured.
// Never reveals the hash or raw token.
func (s *AuthServer) handleBootstrapStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	has, err := s.mgr.HasBootstrapToken(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"bootstrap_token_configured": has,
		"secure_mode":                s.mgr.IsSecureMode(),
	})
}

// handleTokenRotate POST /v1/auth/token/rotate  (requires valid token)
// Generates a new access token for the calling client, invalidating the old one.
// The new raw token is returned once — the client must save it.
func (s *AuthServer) handleTokenRotate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract current client from context (set by auth middleware).
	client := authdomain.ClientFromContext(r.Context())
	if client == nil || client.ID == "anonymous" {
		http.Error(w, `{"error":{"code":"AUTH_TOKEN_MISSING","message":"Authentication required"}}`, http.StatusUnauthorized)
		return
	}

	newToken, err := s.mgr.RotateToken(r.Context(), client.ID)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":{"code":"AUTH_ROTATE_FAILED","message":"%s"}}`, err.Error()), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"access_token": newToken,
		"message":      "Token rotated successfully. Update your config with the new token — the old token is now invalid.",
	})
}

// handleMe GET /v1/auth/me  (requires valid token)
// Returns the current authenticated client's info.
func (s *AuthServer) handleMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// In open mode there is no real client identity.
	if !s.mgr.IsSecureMode() {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":         "anonymous",
			"git_name":   "anonymous",
			"git_email":  "",
			"secure_mode": false,
		})
		return
	}

	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		http.Error(w, `{"error":{"code":"AUTH_TOKEN_MISSING","message":"Authorization header required"}}`, http.StatusUnauthorized)
		return
	}
	rawToken := strings.TrimPrefix(authHeader, "Bearer ")

	client, err := s.mgr.ValidateToken(r.Context(), rawToken)
	if err != nil {
		http.Error(w, `{"error":{"code":"AUTH_TOKEN_INVALID","message":"Invalid or expired access token"}}`, http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"id":          client.ID,
		"git_name":    client.GitName,
		"git_email":   client.GitEmail,
		"created_at":  client.CreatedAt,
		"last_seen_at": client.LastSeenAt,
		"secure_mode": true,
	})
}

// handleLogActivity POST /v1/auth/log-activity  (requires valid token)
func (s *AuthServer) handleLogActivity(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Command string `json:"command"`
		Repo    string `json:"repo"`
		Branch  string `json:"branch"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	authHeader := r.Header.Get("Authorization")
	rawToken := ""
	if len(authHeader) > 7 {
		rawToken = authHeader[7:] // strip "Bearer "
	}

	if err := s.mgr.LogActivity(r.Context(), rawToken, req.Command, req.Repo, req.Branch); err != nil {
		// Non-fatal — client should not fail on activity log errors
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
