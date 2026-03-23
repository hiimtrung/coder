package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const coderDir = ".coder"

// ProjectState represents the content of .coder/STATE.md
type ProjectState struct {
	Project      string
	CurrentPhase int
	Step         string // discuss | plan | execute | qa | ship | done
	LastAction   string
	Updated      time.Time
	Decisions    []string
	Blockers     []string
	Backlog      []string
	PRs          map[int]string // phase number → PR URL
}

// RoadmapPhase is one phase entry parsed from ROADMAP.md
type RoadmapPhase struct {
	Number int
	Name   string
	Status string // planned | in_progress | done | shipped
}

func coderPath(parts ...string) string {
	all := append([]string{coderDir}, parts...)
	return filepath.Join(all...)
}

func loadState() (*ProjectState, error) {
	data, err := os.ReadFile(coderPath("STATE.md"))
	if err != nil {
		return nil, err
	}
	s := &ProjectState{PRs: make(map[int]string)}
	lines := strings.Split(string(data), "\n")
	section := ""
	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		switch {
		case strings.HasPrefix(line, "project:"):
			s.Project = strings.TrimSpace(strings.TrimPrefix(line, "project:"))
		case strings.HasPrefix(line, "current_phase:"):
			s.CurrentPhase, _ = strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "current_phase:")))
		case strings.HasPrefix(line, "step:"):
			s.Step = strings.TrimSpace(strings.TrimPrefix(line, "step:"))
		case strings.HasPrefix(line, "last_action:"):
			s.LastAction = strings.TrimSpace(strings.TrimPrefix(line, "last_action:"))
		case strings.HasPrefix(line, "updated:"):
			s.Updated, _ = time.Parse(time.RFC3339, strings.TrimSpace(strings.TrimPrefix(line, "updated:")))
		case line == "## Decisions":
			section = "decisions"
		case line == "## Blockers":
			section = "blockers"
		case line == "## Backlog":
			section = "backlog"
		case line == "## PRs":
			section = "prs"
		case strings.HasPrefix(line, "## "):
			section = ""
		case strings.HasPrefix(line, "- ") && section != "":
			val := strings.TrimPrefix(line, "- ")
			switch section {
			case "decisions":
				s.Decisions = append(s.Decisions, val)
			case "blockers":
				s.Blockers = append(s.Blockers, val)
			case "backlog":
				s.Backlog = append(s.Backlog, val)
			case "prs":
				// format: "phase N: url"
				re := regexp.MustCompile(`phase (\d+): (.+)`)
				if m := re.FindStringSubmatch(val); len(m) == 3 {
					n, _ := strconv.Atoi(m[1])
					s.PRs[n] = m[2]
				}
			}
		}
	}
	return s, nil
}

func saveState(s *ProjectState) error {
	os.MkdirAll(coderDir, 0755)
	s.Updated = time.Now()
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("project: %s\n", s.Project))
	sb.WriteString(fmt.Sprintf("current_phase: %d\n", s.CurrentPhase))
	sb.WriteString(fmt.Sprintf("step: %s\n", s.Step))
	sb.WriteString(fmt.Sprintf("last_action: %s\n", s.LastAction))
	sb.WriteString(fmt.Sprintf("updated: %s\n", s.Updated.Format(time.RFC3339)))
	sb.WriteString("\n## Decisions\n")
	for _, d := range s.Decisions {
		sb.WriteString(fmt.Sprintf("- %s\n", d))
	}
	sb.WriteString("\n## Blockers\n")
	for _, b := range s.Blockers {
		sb.WriteString(fmt.Sprintf("- %s\n", b))
	}
	sb.WriteString("\n## Backlog\n")
	for _, b := range s.Backlog {
		sb.WriteString(fmt.Sprintf("- %s\n", b))
	}
	sb.WriteString("\n## PRs\n")
	for phase, url := range s.PRs {
		sb.WriteString(fmt.Sprintf("- phase %d: %s\n", phase, url))
	}
	return os.WriteFile(coderPath("STATE.md"), []byte(sb.String()), 0644)
}

func loadRoadmap() ([]RoadmapPhase, error) {
	data, err := os.ReadFile(coderPath("ROADMAP.md"))
	if err != nil {
		return nil, err
	}
	var phases []RoadmapPhase
	re := regexp.MustCompile(`(?m)^##?\s+Phase\s+(\d+)[^\n]*—\s*([^\n]+)`)
	statusRe := regexp.MustCompile(`\[(x|done|shipped| )\]`)
	for _, m := range re.FindAllStringSubmatch(string(data), -1) {
		n, _ := strconv.Atoi(m[1])
		name := strings.TrimSpace(m[2])
		status := "planned"
		if sm := statusRe.FindString(name); sm != "" {
			if strings.Contains(sm, "x") || strings.Contains(sm, "done") || strings.Contains(sm, "shipped") {
				status = "done"
			}
			name = strings.TrimSpace(statusRe.ReplaceAllString(name, ""))
		}
		phases = append(phases, RoadmapPhase{Number: n, Name: name, Status: status})
	}
	return phases, nil
}

func phaseDir(n int) string {
	return coderPath("phases", fmt.Sprintf("%02d", n))
}

func ensurePhaseDir(n int) error {
	return os.MkdirAll(phaseDir(n), 0755)
}

func projectExists() bool {
	_, err := os.Stat(coderPath("PROJECT.md"))
	return err == nil
}
