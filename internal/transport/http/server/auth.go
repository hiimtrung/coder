package httpserver

import (
	"encoding/json"
	"fmt"
	"net/http"

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
