package review

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	chatdomain "github.com/trungtran/coder/internal/domain/chat"
	reviewdomain "github.com/trungtran/coder/internal/domain/review"
	memdomain "github.com/trungtran/coder/internal/domain/memory"
	skilldomain "github.com/trungtran/coder/internal/domain/skill"
)

const contextTimeout = 300 * time.Millisecond

const reviewSystemPrompt = `You are a senior code reviewer. Analyze the following code changes and provide structured feedback.

%s%s%s
## Instructions
Return ONLY a valid JSON object — no markdown fences, no extra text — with this exact structure:
{
  "summary": "one paragraph overall assessment",
  "strengths": ["strength 1", "strength 2"],
  "concerns": [
    {
      "severity": "HIGH|MEDIUM|LOW",
      "description": "clear description",
      "location": "file:line if known, else empty string",
      "suggestion": "concrete fix"
    }
  ],
  "suggestions": ["improvement 1", "improvement 2"]
}

Severity guide:
  HIGH   — security issue, data loss risk, production bug
  MEDIUM — correctness issue, missing test, bad pattern
  LOW    — style, naming, minor clarity

%s
## Code to review:
%s`

// Manager runs the review use case.
type Manager struct {
	llm    chatdomain.LLMProvider
	memory memdomain.MemoryManager
	skills skilldomain.SkillUseCase
	model  string
}

func NewManager(llm chatdomain.LLMProvider, memory memdomain.MemoryManager, skills skilldomain.SkillUseCase, model string) *Manager {
	if model == "" {
		model = "llama3.2:latest"
	}
	return &Manager{llm: llm, memory: memory, skills: skills, model: model}
}

// Review performs a structured code review.
func (m *Manager) Review(ctx context.Context, req reviewdomain.ReviewRequest) (*reviewdomain.ReviewResult, error) {
	// Parallel context search
	type memResult struct {
		results []memdomain.SearchResult
	}
	type skillResult struct {
		results []skilldomain.SkillSearchResult
	}
	memCh := make(chan memResult, 1)
	skillCh := make(chan skillResult, 1)

	ctxT, cancel := context.WithTimeout(ctx, contextTimeout)
	defer cancel()

	query := req.Content
	if len(query) > 500 {
		query = query[:500]
	}

	if req.Context.InjectMemory && m.memory != nil {
		go func() {
			results, _ := m.memory.Search(ctxT, query, "", nil, "", nil, 5)
			memCh <- memResult{results}
		}()
	} else {
		memCh <- memResult{}
	}
	if req.Context.InjectSkills && m.skills != nil {
		go func() {
			results, _ := m.skills.SearchSkills(ctxT, query, 3)
			skillCh <- skillResult{results}
		}()
	} else {
		skillCh <- skillResult{}
	}

	memRes := <-memCh
	skillRes := <-skillCh

	// Build context sections
	var skillSection, memSection, focusSection string
	var contextUsed reviewdomain.ContextUsed

	if len(skillRes.results) > 0 {
		var sb strings.Builder
		sb.WriteString("## Relevant patterns and standards (from skills):\n")
		for _, r := range skillRes.results {
			for _, chunk := range r.Chunks {
				sb.WriteString(chunk.Content + "\n")
			}
			contextUsed.SkillHits = append(contextUsed.SkillHits, r.Skill.Name)
		}
		skillSection = sb.String() + "\n"
	}
	if len(memRes.results) > 0 {
		var sb strings.Builder
		sb.WriteString("## Project context (from memory):\n")
		for _, r := range memRes.results {
			sb.WriteString(r.Title + ": " + r.Content + "\n")
			contextUsed.MemoryHits = append(contextUsed.MemoryHits, r.Title)
		}
		memSection = sb.String() + "\n"
	}
	if req.Focus != "" {
		focusSection = fmt.Sprintf("## Focus area: %s\n\n", req.Focus)
	}

	prompt := fmt.Sprintf(reviewSystemPrompt,
		memSection, skillSection, focusSection, focusSection, req.Content,
	)

	llmResp, err := m.llm.Chat(ctx, m.model, []chatdomain.LLMMessage{
		{Role: "user", Content: prompt},
	})
	if err != nil {
		return nil, err
	}

	// Parse JSON response
	result := &reviewdomain.ReviewResult{
		ContextUsed: contextUsed,
		Model:       m.model,
	}

	// Extract JSON block — LLM may wrap in markdown fences
	raw := extractJSON(llmResp.Content)
	if err := json.Unmarshal([]byte(raw), result); err != nil {
		// Fallback: return raw as summary
		result.Summary = llmResp.Content
		return result, nil
	}

	// Compute stats
	for _, c := range result.Concerns {
		switch strings.ToUpper(c.Severity) {
		case "HIGH":
			result.Stats.ConcernsHigh++
		case "MEDIUM":
			result.Stats.ConcernsMedium++
		case "LOW":
			result.Stats.ConcernsLow++
		}
	}
	// Count files from diff
	result.Stats.FilesReviewed = countFiles(req.Content, req.Type)

	return result, nil
}

// extractJSON strips optional markdown code fences from LLM output.
func extractJSON(s string) string {
	s = strings.TrimSpace(s)
	// Strip ```json ... ``` or ``` ... ```
	if idx := strings.Index(s, "```json"); idx >= 0 {
		s = s[idx+7:]
	} else if idx := strings.Index(s, "```"); idx >= 0 {
		s = s[idx+3:]
	}
	if idx := strings.LastIndex(s, "```"); idx >= 0 {
		s = s[:idx]
	}
	// Find first { and last }
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start >= 0 && end > start {
		return s[start : end+1]
	}
	return s
}

// countFiles estimates the number of files reviewed from a diff or type.
func countFiles(content, reviewType string) int {
	if reviewType == "file" {
		return 1
	}
	count := 0
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "diff --git") || strings.HasPrefix(line, "--- a/") {
			count++
		}
	}
	if count == 0 {
		return 1
	}
	return count
}
