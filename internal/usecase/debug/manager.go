package debug

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	chatdomain "github.com/trungtran/coder/internal/domain/chat"
	debugdomain "github.com/trungtran/coder/internal/domain/debug"
	memdomain "github.com/trungtran/coder/internal/domain/memory"
	skilldomain "github.com/trungtran/coder/internal/domain/skill"
)

const contextTimeout = 300 * time.Millisecond

const debugSystemPrompt = `You are a senior software engineer debugging a reported error.
%s%s
## Error to debug:
%s
%s%s
Analyze and respond ONLY with a valid JSON object (no markdown fences, no extra text):
{
  "root_cause": "clear explanation of what is wrong and why",
  "location": "file:line if determinable, else empty string",
  "confidence": "HIGH|MEDIUM|LOW",
  "suggested_fix": "concrete code fix or step-by-step instructions",
  "similar_issues": ["past similar bugs if found in context"],
  "follow_up": "what to check if this fix does not work"
}`

// Manager runs the debug use case.
type Manager struct {
	llm    chatdomain.LLMProvider
	memory memdomain.MemoryManager
	skills skilldomain.SkillUseCase
	model  string
}

func NewManager(llm chatdomain.LLMProvider, memory memdomain.MemoryManager, skills skilldomain.SkillUseCase, model string) *Manager {
	if model == "" {
		model = "qwen3.5:0.8b"
	}
	return &Manager{llm: llm, memory: memory, skills: skills, model: model}
}

// Debug analyses an error and returns a structured diagnosis.
func (m *Manager) Debug(ctx context.Context, req debugdomain.DebugRequest) (*debugdomain.DebugResult, error) {
	// Parallel context search
	type memR struct{ results []memdomain.SearchResult }
	type skillR struct{ results []skilldomain.SkillSearchResult }
	memCh := make(chan memR, 1)
	skillCh := make(chan skillR, 1)

	ctxT, cancel := context.WithTimeout(ctx, contextTimeout)
	defer cancel()

	query := req.ErrorMessage
	if len(query) > 300 {
		query = query[:300]
	}

	if req.Context.InjectMemory && m.memory != nil {
		go func() {
			results, _ := m.memory.Search(ctxT, query, "", nil, "", nil, 5)
			memCh <- memR{results}
		}()
	} else {
		memCh <- memR{}
	}
	if req.Context.InjectSkills && m.skills != nil {
		go func() {
			results, _ := m.skills.SearchSkills(ctxT, query, 3)
			skillCh <- skillR{results}
		}()
	} else {
		skillCh <- skillR{}
	}

	memRes := <-memCh
	skillRes := <-skillCh

	var memSection, skillSection, fileSection, diffSection string
	var contextUsed debugdomain.ContextUsed

	if len(memRes.results) > 0 {
		var sb strings.Builder
		sb.WriteString("## Project context (from memory):\n")
		for _, r := range memRes.results {
			sb.WriteString(r.Title + ": " + r.Content + "\n")
			contextUsed.MemoryHits = append(contextUsed.MemoryHits, r.Title)
		}
		memSection = sb.String() + "\n"
	}
	if len(skillRes.results) > 0 {
		var sb strings.Builder
		sb.WriteString("## Relevant patterns (from skills):\n")
		for _, r := range skillRes.results {
			for _, chunk := range r.Chunks {
				sb.WriteString(chunk.Content + "\n")
			}
			contextUsed.SkillHits = append(contextUsed.SkillHits, r.Skill.Name)
		}
		skillSection = sb.String() + "\n"
	}
	if req.FileContext != "" {
		fileSection = fmt.Sprintf("## File context:\n```\n%s\n```\n\n", req.FileContext)
	}
	if req.DiffContext != "" {
		diffSection = fmt.Sprintf("## Git diff context:\n```diff\n%s\n```\n\n", req.DiffContext)
	}

	prompt := fmt.Sprintf(debugSystemPrompt,
		memSection, skillSection,
		req.ErrorMessage,
		fileSection, diffSection,
	)

	llmResp, err := m.llm.Chat(ctx, m.model, []chatdomain.LLMMessage{
		{Role: "user", Content: prompt},
	})
	if err != nil {
		return nil, err
	}

	result := &debugdomain.DebugResult{
		ContextUsed: contextUsed,
		Model:       m.model,
	}

	raw := extractJSON(llmResp.Content)
	if err := json.Unmarshal([]byte(raw), result); err != nil {
		result.RootCause = llmResp.Content
		result.Confidence = "LOW"
	}

	return result, nil
}

func extractJSON(s string) string {
	s = strings.TrimSpace(s)
	if idx := strings.Index(s, "```json"); idx >= 0 {
		s = s[idx+7:]
	} else if idx := strings.Index(s, "```"); idx >= 0 {
		s = s[idx+3:]
	}
	if idx := strings.LastIndex(s, "```"); idx >= 0 {
		s = s[:idx]
	}
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start >= 0 && end > start {
		return s[start : end+1]
	}
	return s
}
