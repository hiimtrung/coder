package httpserver

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/trungtran/coder/internal/skill"
)

type SkillServer struct {
	ingestor *skill.Ingestor
	store    skill.SkillService
}

func NewSkillServer(ingestor *skill.Ingestor, store skill.SkillService) *SkillServer {
	return &SkillServer{
		ingestor: ingestor,
		store:    store,
	}
}

func (s *SkillServer) RegisterHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/v1/skill/ingest", s.handleIngest)
	mux.HandleFunc("/v1/skill/search", s.handleSearch)
	mux.HandleFunc("/v1/skill/list", s.handleList)
	mux.HandleFunc("/v1/skill/info", s.handleGetSkill)
	mux.HandleFunc("/v1/skill/delete", s.handleDelete)
}

func (s *SkillServer) handleIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name           string `json:"name"`
		SkillMDContent string `json:"skill_md_content"`
		Rules          []struct {
			Path    string `json:"path"`
			Content string `json:"content"`
		} `json:"rules"`
		Source     string `json:"source"`
		SourceRepo string `json:"source_repo"`
		Category  string `json:"category"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var rules []skill.RuleFile
	for _, ru := range req.Rules {
		rules = append(rules, skill.RuleFile{Path: ru.Path, Content: ru.Content})
	}

	result, err := s.ingestor.IngestSkill(r.Context(), req.Name, req.SkillMDContent, rules, req.Source, req.SourceRepo, req.Category)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"skill_id":     result.SkillName,
		"chunks_total": result.ChunksTotal,
		"chunks_new":   result.ChunksNew,
	})
}

func (s *SkillServer) handleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Limit <= 0 {
		req.Limit = 5
	}

	results, err := s.ingestor.SearchSkills(r.Context(), req.Query, req.Limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"results": results})
}

func (s *SkillServer) handleList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	source := r.URL.Query().Get("source")
	category := r.URL.Query().Get("category")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	if limit <= 0 {
		limit = 100
	}

	skills, err := s.store.ListSkills(r.Context(), source, category, limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"skills": skills})
}

func (s *SkillServer) handleGetSkill(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	sk, chunks, err := s.store.GetSkill(r.Context(), name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"skill":  sk,
		"chunks": chunks,
	})
}

func (s *SkillServer) handleDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	name := r.URL.Query().Get("name")
	if name == "" {
		var req struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			name = req.Name
		}
	}

	if name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	if err := s.store.DeleteSkill(r.Context(), name); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}
