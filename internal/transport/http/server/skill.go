package httpserver

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	skilldomain "github.com/trungtran/coder/internal/domain/skill"
)

type SkillServer struct {
	svc skilldomain.SkillUseCase
}

func NewSkillServer(svc skilldomain.SkillUseCase) *SkillServer {
	return &SkillServer{
		svc: svc,
	}
}

func (s *SkillServer) RegisterHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/v1/skill/ingest", s.handleIngest)
	mux.HandleFunc("/v1/skill/search", s.handleSearch)
	mux.HandleFunc("/v1/skill/list", s.handleList)
	mux.HandleFunc("/v1/skill/info", s.handleGetSkill)
	mux.HandleFunc("/v1/skill/delete", s.handleDelete)
	mux.HandleFunc("/v1/skill/files", s.handleFiles)
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
		Category   string `json:"category"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var rules []skilldomain.RuleFile
	for _, ru := range req.Rules {
		rules = append(rules, skilldomain.RuleFile{Path: ru.Path, Content: ru.Content})
	}

	result, err := s.svc.IngestSkill(r.Context(), req.Name, req.SkillMDContent, rules, req.Source, req.SourceRepo, req.Category)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
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

	results, err := s.svc.SearchSkills(r.Context(), req.Query, req.Limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"results": results})
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

	skills, err := s.svc.ListSkills(r.Context(), source, category, limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"skills": skills})
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

	sk, chunks, err := s.svc.GetSkill(r.Context(), name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"skill":  sk,
		"chunks": chunks,
	})
}

// handleFiles handles GET /v1/skill/files?name=<skill> and POST /v1/skill/files.
func (s *SkillServer) handleFiles(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		name := r.URL.Query().Get("name")
		if name == "" {
			http.Error(w, "name is required", http.StatusBadRequest)
			return
		}
		sk, _, err := s.svc.GetSkill(r.Context(), name)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		files, err := s.svc.GetFiles(r.Context(), sk.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		type fileResp struct {
			RelPath     string `json:"rel_path"`
			ContentType string `json:"content_type"`
			Content     string `json:"content"` // base64
			SizeBytes   int    `json:"size_bytes"`
		}
		resp := make([]fileResp, len(files))
		for i, f := range files {
			resp[i] = fileResp{
				RelPath:     f.RelPath,
				ContentType: f.ContentType,
				Content:     base64.StdEncoding.EncodeToString(f.Content),
				SizeBytes:   f.SizeBytes,
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"files": resp})

	case http.MethodPost:
		var req struct {
			SkillName string `json:"skill_name"`
			Files     []struct {
				RelPath     string `json:"rel_path"`
				ContentType string `json:"content_type"`
				Content     string `json:"content"` // base64
				SizeBytes   int    `json:"size_bytes"`
			} `json:"files"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		sk, _, err := s.svc.GetSkill(r.Context(), req.SkillName)
		if err != nil {
			http.Error(w, "skill not found: "+err.Error(), http.StatusNotFound)
			return
		}

		now := time.Now()
		var files []skilldomain.SkillFile
		for _, f := range req.Files {
			decoded, err := base64.StdEncoding.DecodeString(f.Content)
			if err != nil {
				http.Error(w, "invalid base64 for "+f.RelPath, http.StatusBadRequest)
				return
			}
			files = append(files, skilldomain.SkillFile{
				ID:          uuid.New().String(),
				SkillID:     sk.ID,
				RelPath:     f.RelPath,
				ContentType: f.ContentType,
				Content:     decoded,
				ContentHash: sha256Hex(decoded),
				SizeBytes:   len(decoded),
				CreatedAt:   now,
			})
		}

		stored, err := s.svc.StoreFiles(r.Context(), sk.ID, files, now)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"stored": stored})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func sha256Hex(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
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

	if err := s.svc.DeleteSkill(r.Context(), name); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}
