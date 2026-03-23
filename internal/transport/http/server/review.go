package httpserver

import (
	"encoding/json"
	"net/http"

	reviewdomain "github.com/trungtran/coder/internal/domain/review"
	ucreview "github.com/trungtran/coder/internal/usecase/review"
)

// ReviewServer handles POST /v1/review.
type ReviewServer struct {
	mgr *ucreview.Manager
}

func NewReviewServer(mgr *ucreview.Manager) *ReviewServer {
	return &ReviewServer{mgr: mgr}
}

func (s *ReviewServer) RegisterHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/v1/review", s.handleReview)
}

// POST /v1/review
func (s *ReviewServer) handleReview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Type    string `json:"type"`    // diff | file | pr
		Content string `json:"content"` // raw diff or file content
		Focus   string `json:"focus"`   // optional
		Context struct {
			InjectMemory bool `json:"inject_memory"`
			InjectSkills bool `json:"inject_skills"`
		} `json:"context"`
	}
	// Default context injection to true
	req.Context.InjectMemory = true
	req.Context.InjectSkills = true

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":{"code":"VAL_INVALID_BODY","message":"invalid request body"}}`, http.StatusBadRequest)
		return
	}
	if req.Content == "" {
		http.Error(w, `{"error":{"code":"VAL_MISSING_CONTENT","message":"content is required"}}`, http.StatusBadRequest)
		return
	}
	if req.Type == "" {
		req.Type = "diff"
	}

	result, err := s.mgr.Review(r.Context(), reviewdomain.ReviewRequest{
		Type:    req.Type,
		Content: req.Content,
		Focus:   req.Focus,
		Context: reviewdomain.ReviewContext{
			InjectMemory: req.Context.InjectMemory,
			InjectSkills: req.Context.InjectSkills,
		},
	})
	if err != nil {
		http.Error(w, `{"error":{"code":"INF_REVIEW_ERROR","message":"`+sanitize(err.Error())+`"}}`, http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
