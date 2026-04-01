package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	skilldomain "github.com/trungtran/coder/internal/domain/skill"
)

const activeSkillsStateFile = "active-skills.json"

type activeSkillState struct {
	Task       string             `json:"task"`
	Trigger    string             `json:"trigger"`
	Budget     int                `json:"budget"`
	ResolvedAt time.Time          `json:"resolved_at"`
	Keep       []string           `json:"keep"`
	Add        []string           `json:"add"`
	Drop       []string           `json:"drop"`
	Skills     []activeSkillEntry `json:"skills"`
}

type activeSkillEntry struct {
	Name       string  `json:"name"`
	Category   string  `json:"category"`
	Score      float32 `json:"score"`
	ChunkCount int     `json:"chunk_count"`
	Reason     string  `json:"reason"`
}

type skillResolveOutput struct {
	Task       string                `json:"task"`
	Trigger    string                `json:"trigger"`
	Budget     int                   `json:"budget"`
	ResolvedAt time.Time             `json:"resolved_at"`
	Keep       []string              `json:"keep"`
	Add        []string              `json:"add"`
	Drop       []string              `json:"drop"`
	Skills     []resolvedSkillResult `json:"skills"`
}

type resolvedSkillResult struct {
	Name        string                         `json:"name"`
	Category    string                         `json:"category"`
	Score       float32                        `json:"score"`
	Reason      string                         `json:"reason"`
	Description string                         `json:"description,omitempty"`
	Chunks      []skilldomain.SkillChunkResult `json:"chunks"`
}

func activeSkillsPath() string {
	return coderPath(activeSkillsStateFile)
}

func loadActiveSkillState() (*activeSkillState, error) {
	data, err := os.ReadFile(activeSkillsPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var state activeSkillState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func saveActiveSkillState(state *activeSkillState) error {
	if err := os.MkdirAll(coderDir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(activeSkillsPath(), data, 0644); err != nil {
		return err
	}
	return syncContextState(state, nil)
}

func normalizeSkillNames(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	var out []string
	seen := make(map[string]bool)
	for _, item := range strings.Split(raw, ",") {
		name := strings.TrimSpace(item)
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, name)
	}
	return out
}

func currentSkillsFromFlagOrState(raw string) ([]string, error) {
	if names := normalizeSkillNames(raw); len(names) > 0 {
		return names, nil
	}

	state, err := loadActiveSkillState()
	if err != nil || state == nil {
		return nil, err
	}

	out := make([]string, 0, len(state.Skills))
	for _, skill := range state.Skills {
		out = append(out, skill.Name)
	}
	return out, nil
}

func buildResolveOutput(task, trigger string, budget int, current []string, results []skilldomain.SkillSearchResult) (*skillResolveOutput, *activeSkillState) {
	if budget <= 0 {
		budget = 3
	}

	selected, keep, add, drop := selectResolvedSkills(results, current, budget)
	resolvedAt := time.Now()

	output := &skillResolveOutput{
		Task:       task,
		Trigger:    trigger,
		Budget:     budget,
		ResolvedAt: resolvedAt,
		Keep:       keep,
		Add:        add,
		Drop:       drop,
		Skills:     make([]resolvedSkillResult, 0, len(selected)),
	}
	state := &activeSkillState{
		Task:       task,
		Trigger:    trigger,
		Budget:     budget,
		ResolvedAt: resolvedAt,
		Keep:       keep,
		Add:        add,
		Drop:       drop,
		Skills:     make([]activeSkillEntry, 0, len(selected)),
	}

	currentSet := toSkillNameSet(current)
	for _, sr := range selected {
		reason := resolveReason(sr.Skill.Name, sr.Score, currentSet[strings.ToLower(sr.Skill.Name)])
		output.Skills = append(output.Skills, resolvedSkillResult{
			Name:        sr.Skill.Name,
			Category:    sr.Skill.Category,
			Score:       sr.Score,
			Reason:      reason,
			Description: sr.Skill.Description,
			Chunks:      sr.Chunks,
		})
		state.Skills = append(state.Skills, activeSkillEntry{
			Name:       sr.Skill.Name,
			Category:   sr.Skill.Category,
			Score:      sr.Score,
			ChunkCount: len(sr.Chunks),
			Reason:     reason,
		})
	}

	return output, state
}

func selectResolvedSkills(results []skilldomain.SkillSearchResult, current []string, budget int) ([]skilldomain.SkillSearchResult, []string, []string, []string) {
	if budget <= 0 {
		budget = 3
	}

	ranked := append([]skilldomain.SkillSearchResult(nil), results...)
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].Score == ranked[j].Score {
			return ranked[i].Skill.Name < ranked[j].Skill.Name
		}
		return ranked[i].Score > ranked[j].Score
	})
	if len(ranked) > budget {
		ranked = ranked[:budget]
	}

	currentSet := toSkillNameSet(current)
	selectedSet := make(map[string]bool, len(ranked))

	keep := make([]string, 0, len(ranked))
	add := make([]string, 0, len(ranked))
	for _, sr := range ranked {
		key := strings.ToLower(sr.Skill.Name)
		selectedSet[key] = true
		if currentSet[key] {
			keep = append(keep, sr.Skill.Name)
			continue
		}
		add = append(add, sr.Skill.Name)
	}

	var drop []string
	for _, name := range current {
		if !selectedSet[strings.ToLower(name)] {
			drop = append(drop, name)
		}
	}

	return ranked, keep, add, drop
}

func toSkillNameSet(names []string) map[string]bool {
	set := make(map[string]bool, len(names))
	for _, name := range names {
		set[strings.ToLower(strings.TrimSpace(name))] = true
	}
	return set
}

func resolveReason(name string, score float32, alreadyActive bool) string {
	switch {
	case alreadyActive && score >= 0.80:
		return "kept active: still a high-confidence match"
	case alreadyActive:
		return "kept active: still relevant after re-resolve"
	case score >= 0.80:
		return "added: high-confidence match for the current task"
	case score >= 0.55:
		return "added: medium-confidence support skill"
	default:
		return "added: fallback skill for low-confidence coverage"
	}
}

func renderRawSkillContext(results []skilldomain.SkillSearchResult) string {
	var sb strings.Builder
	for i, sr := range results {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(fmt.Sprintf("<!-- coder-skill name=\"%s\" category=\"%s\" score=\"%.4f\" -->\n", sr.Skill.Name, sr.Skill.Category, sr.Score))
		sb.WriteString(fmt.Sprintf("# Skill: %s\n\n", sr.Skill.Name))
		if sr.Skill.Description != "" {
			sb.WriteString(sr.Skill.Description)
			sb.WriteString("\n\n")
		}
		for _, chunk := range sr.Chunks {
			sb.WriteString(fmt.Sprintf("<!-- coder-skill-chunk skill=\"%s\" chunk_index=\"%d\" chunk_type=\"%s\" section_id=\"%s\" score=\"%.4f\" -->\n", sr.Skill.Name, chunk.ChunkIndex, chunk.ChunkType, chunk.SectionID, chunk.Score))
			sb.WriteString(chunk.Content)
			if !strings.HasSuffix(chunk.Content, "\n") {
				sb.WriteString("\n")
			}
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

func renderRawSkillInfo(sk *skilldomain.Skill, chunks []skilldomain.SkillChunk) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<!-- coder-skill name=\"%s\" category=\"%s\" source=\"%s\" -->\n", sk.Name, sk.Category, sk.Source))
	sb.WriteString(fmt.Sprintf("# Skill: %s\n\n", sk.Name))
	if sk.Description != "" {
		sb.WriteString(sk.Description)
		sb.WriteString("\n\n")
	}
	for _, chunk := range chunks {
		sb.WriteString(fmt.Sprintf("<!-- coder-skill-chunk skill=\"%s\" chunk_index=\"%d\" chunk_type=\"%s\" section_id=\"%s\" -->\n", sk.Name, chunk.ChunkIndex, chunk.ChunkType, chunk.SectionID))
		sb.WriteString(chunk.Content)
		if !strings.HasSuffix(chunk.Content, "\n") {
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}
	return sb.String()
}
