package httpserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	authdomain "github.com/trungtran/coder/internal/domain/auth"
	chatdomain "github.com/trungtran/coder/internal/domain/chat"
)

// ChatServer handles /v1/chat and /v1/sessions endpoints.
type ChatServer struct {
	mgr chatdomain.Manager
}

func NewChatServer(mgr chatdomain.Manager) *ChatServer {
	return &ChatServer{mgr: mgr}
}

func (s *ChatServer) RegisterHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/v1/chat", s.handleChat)
	mux.HandleFunc("/v1/chat/stream", s.handleChatStream)
	mux.HandleFunc("/v1/sessions", s.handleSessions)
	mux.HandleFunc("/v1/sessions/", s.handleSession)
}

// POST /v1/chat — blocking completion
func (s *ChatServer) handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	clientID := clientIDFromRequest(r)

	var req struct {
		Message   string `json:"message"`
		SessionID string `json:"session_id"`
		Context   struct {
			InjectMemory bool   `json:"inject_memory"`
			InjectSkills bool   `json:"inject_skills"`
			MemoryLimit  int    `json:"memory_limit"`
			SkillLimit   int    `json:"skill_limit"`
			ExtraSystem  string `json:"extra_system"`
		} `json:"context"`
	}
	// Set inject defaults to true
	req.Context.InjectMemory = true
	req.Context.InjectSkills = true

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":{"code":"VAL_INVALID_BODY","message":"invalid request body"}}`, http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Message) == "" {
		http.Error(w, `{"error":{"code":"VAL_MISSING_MESSAGE","message":"message is required"}}`, http.StatusBadRequest)
		return
	}

	chatReq := chatdomain.ChatRequest{
		Message:   req.Message,
		SessionID: req.SessionID,
		Context: chatdomain.ChatContext{
			InjectMemory: req.Context.InjectMemory,
			InjectSkills: req.Context.InjectSkills,
			MemoryLimit:  req.Context.MemoryLimit,
			SkillLimit:   req.Context.SkillLimit,
			ExtraSystem:  req.Context.ExtraSystem,
		},
	}

	resp, err := s.mgr.Chat(r.Context(), clientID, chatReq)
	if err != nil {
		code := "INF_CHAT_ERROR"
		if strings.Contains(err.Error(), "INF_LLM_UNREACHABLE") {
			code = "INF_LLM_UNREACHABLE"
		}
		http.Error(w, fmt.Sprintf(`{"error":{"code":"%s","message":"%s","action":"Ensure Ollama is running: ollama serve"}}`, code, sanitize(err.Error())), http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"reply":        resp.Reply,
		"session_id":   resp.SessionID,
		"context_used": resp.ContextUsed,
		"model":        resp.Model,
		"tokens":       resp.Tokens,
	})
}

// POST /v1/chat/stream — SSE streaming completion
func (s *ChatServer) handleChatStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	clientID := clientIDFromRequest(r)

	var req struct {
		Message   string `json:"message"`
		SessionID string `json:"session_id"`
		Context   struct {
			InjectMemory bool   `json:"inject_memory"`
			InjectSkills bool   `json:"inject_skills"`
			MemoryLimit  int    `json:"memory_limit"`
			SkillLimit   int    `json:"skill_limit"`
			ExtraSystem  string `json:"extra_system"`
		} `json:"context"`
	}
	req.Context.InjectMemory = true
	req.Context.InjectSkills = true

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Message) == "" {
		http.Error(w, "message is required", http.StatusBadRequest)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	chatReq := chatdomain.ChatRequest{
		Message:   req.Message,
		SessionID: req.SessionID,
		Context: chatdomain.ChatContext{
			InjectMemory: req.Context.InjectMemory,
			InjectSkills: req.Context.InjectSkills,
			MemoryLimit:  req.Context.MemoryLimit,
			SkillLimit:   req.Context.SkillLimit,
			ExtraSystem:  req.Context.ExtraSystem,
		},
	}

	// onDelta streams each token chunk as SSE
	onDelta := func(delta string) {
		data, _ := json.Marshal(map[string]string{"delta": delta})
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}

	resp, err := s.mgr.ChatStream(r.Context(), clientID, chatReq, onDelta)
	if err != nil {
		data, _ := json.Marshal(map[string]string{"error": err.Error()})
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
		return
	}

	// Send done event with metadata
	done, _ := json.Marshal(map[string]any{
		"done":         true,
		"session_id":   resp.SessionID,
		"context_used": resp.ContextUsed,
		"tokens":       resp.Tokens,
	})
	fmt.Fprintf(w, "data: %s\n\n", done)
	flusher.Flush()
}

// GET /v1/sessions — list sessions for the authenticated client
func (s *ChatServer) handleSessions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		clientID := clientIDFromRequest(r)
		sessions, err := s.mgr.ListSessions(r.Context(), clientID, 50)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if sessions == nil {
			sessions = []chatdomain.Session{}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"sessions": sessions})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// GET /v1/sessions/:id  — get session history
// DELETE /v1/sessions/:id — delete session
func (s *ChatServer) handleSession(w http.ResponseWriter, r *http.Request) {
	// Extract ID from path: /v1/sessions/{id}
	id := strings.TrimPrefix(r.URL.Path, "/v1/sessions/")
	if id == "" {
		http.Error(w, "session id required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		session, msgs, err := s.mgr.GetSession(r.Context(), id)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":{"code":"BIZ_SESSION_NOT_FOUND","message":"%s"}}`, sanitize(err.Error())), http.StatusNotFound)
			return
		}
		if msgs == nil {
			msgs = []chatdomain.Message{}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"session":  session,
			"messages": msgs,
		})

	case http.MethodDelete:
		if err := s.mgr.DeleteSession(r.Context(), id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"success": true})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// clientIDFromRequest extracts the client ID from the request context (set by auth middleware).
// Falls back to "anonymous" in open mode.
func clientIDFromRequest(r *http.Request) string {
	client := authdomain.ClientFromContext(r.Context())
	if client != nil && client.ID != "" {
		return client.ID
	}
	return "anonymous"
}

func sanitize(s string) string {
	s = strings.ReplaceAll(s, `"`, `'`)
	s = strings.ReplaceAll(s, "\n", " ")
	return s
}
