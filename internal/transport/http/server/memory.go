package httpserver

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	memdomain "github.com/trungtran/coder/internal/domain/memory"
)

type MemoryServer struct {
	mgr memdomain.MemoryManager
}

func NewMemoryServer(mgr memdomain.MemoryManager) *MemoryServer {
	return &MemoryServer{mgr: mgr}
}

func (s *MemoryServer) RegisterHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/v1/memory/store", s.handleStore)
	mux.HandleFunc("/v1/memory/search", s.handleSearch)
	mux.HandleFunc("/v1/memory/list", s.handleList)
	mux.HandleFunc("/v1/memory/delete", s.handleDelete)
	mux.HandleFunc("/v1/memory/verify", s.handleVerify)
	mux.HandleFunc("/v1/memory/supersede", s.handleSupersede)
	mux.HandleFunc("/v1/memory/audit", s.handleAudit)
	mux.HandleFunc("/v1/memory/compact", s.handleCompact)
}

func (s *MemoryServer) handleStore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Title    string         `json:"title"`
		Content  string         `json:"content"`
		Type     string         `json:"type"`
		Metadata map[string]any `json:"metadata"`
		Scope    string         `json:"scope"`
		Tags     []string       `json:"tags"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id, err := s.mgr.Store(r.Context(), req.Title, req.Content, memdomain.MemoryType(req.Type), req.Metadata, req.Scope, req.Tags)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"id": id})
}

func (s *MemoryServer) handleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Query       string         `json:"query"`
		Scope       string         `json:"scope"`
		Tags        []string       `json:"tags"`
		Type        string         `json:"type"`
		MetaFilters map[string]any `json:"meta_filters"`
		Limit       int            `json:"limit"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Limit <= 0 {
		req.Limit = 5
	}

	results, err := s.mgr.Search(r.Context(), req.Query, req.Scope, req.Tags, memdomain.MemoryType(req.Type), req.MetaFilters, req.Limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"results": results})
}

func (s *MemoryServer) handleList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	if limit <= 0 {
		limit = 10
	}

	items, err := s.mgr.List(r.Context(), limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"items": items})
}

func (s *MemoryServer) handleDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		// Try to parse from body if it's a POST
		var req struct {
			ID string `json:"id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			id = req.ID
		}
	}

	if id == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}

	if err := s.mgr.Delete(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (s *MemoryServer) handleVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ID         string     `json:"id"`
		VerifiedAt *time.Time `json:"verified_at"`
		VerifiedBy string     `json:"verified_by"`
		Confidence *float64   `json:"confidence"`
		SourceRef  string     `json:"source_ref"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var opts memdomain.VerifyOptions
	if req.VerifiedAt != nil {
		opts.VerifiedAt = req.VerifiedAt.UTC()
	}
	opts.VerifiedBy = req.VerifiedBy
	opts.Confidence = req.Confidence
	opts.SourceRef = req.SourceRef

	updated, err := s.mgr.Verify(r.Context(), req.ID, opts)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"updated_count": updated})
}

func (s *MemoryServer) handleSupersede(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ID            string `json:"id"`
		ReplacementID string `json:"replacement_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	updated, err := s.mgr.Supersede(r.Context(), req.ID, req.ReplacementID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"updated_count": updated})
}

func (s *MemoryServer) handleAudit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req memdomain.AuditOptions
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	report, err := s.mgr.Audit(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"report": report})
}

func (s *MemoryServer) handleCompact(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Threshold float32 `json:"threshold"`
		Revector  bool    `json:"revector"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// threshold might be optional
	}

	if req.Revector {
		if err := s.mgr.Revector(r.Context()); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	count, err := s.mgr.Compact(r.Context(), req.Threshold)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"removed_count": count})
}
