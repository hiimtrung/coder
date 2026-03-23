package httpserver

import (
	"encoding/json"
	"net/http"

	debugdomain "github.com/trungtran/coder/internal/domain/debug"
	ucdebug "github.com/trungtran/coder/internal/usecase/debug"
)

// DebugServer handles POST /v1/debug.
type DebugServer struct {
	mgr *ucdebug.Manager
}

func NewDebugServer(mgr *ucdebug.Manager) *DebugServer {
	return &DebugServer{mgr: mgr}
}

func (s *DebugServer) RegisterHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/v1/debug", s.handleDebug)
}

// POST /v1/debug
func (s *DebugServer) handleDebug(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ErrorMessage string `json:"error_message"`
		FileContext  string `json:"file_context"`
		DiffContext  string `json:"diff_context"`
		Context      struct {
			InjectMemory bool `json:"inject_memory"`
			InjectSkills bool `json:"inject_skills"`
		} `json:"context"`
	}
	req.Context.InjectMemory = true
	req.Context.InjectSkills = true

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":{"code":"VAL_INVALID_BODY","message":"invalid request body"}}`, http.StatusBadRequest)
		return
	}
	if req.ErrorMessage == "" {
		http.Error(w, `{"error":{"code":"VAL_MISSING_ERROR","message":"error_message is required"}}`, http.StatusBadRequest)
		return
	}

	result, err := s.mgr.Debug(r.Context(), debugdomain.DebugRequest{
		ErrorMessage: req.ErrorMessage,
		FileContext:  req.FileContext,
		DiffContext:  req.DiffContext,
		Context: debugdomain.DebugContext{
			InjectMemory: req.Context.InjectMemory,
			InjectSkills: req.Context.InjectSkills,
		},
	})
	if err != nil {
		http.Error(w, `{"error":{"code":"INF_DEBUG_ERROR","message":"`+sanitize(err.Error())+`"}}`, http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
