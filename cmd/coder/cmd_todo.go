package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ── coder todo ────────────────────────────────────────────────────────────────

const todoUsage = `Usage: coder todo [action] [text] [flags]

Manage the project backlog in .coder/STATE.md.

ACTIONS:
  list           List all backlog items (default)
  add <text>     Add a new backlog item
  done <text>    Remove a backlog item (substring match)
  clear          Clear all backlog items

EXAMPLES:
  coder todo
  coder todo add "investigate rate limiting"
  coder todo done "rate limiting"

FLAGS:
`

func runTodo(args []string) {
	fs := flag.NewFlagSet("todo", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, todoUsage)
		fs.PrintDefaults()
	}
	fs.Parse(args)

	logActivity("todo")

	rest := fs.Args()
	action := "list"
	if len(rest) > 0 {
		action = rest[0]
		rest = rest[1:]
	}

	state, err := loadState()
	if err != nil {
		state = &ProjectState{PRs: make(map[int]string)}
	}

	switch action {
	case "list", "":
		if len(state.Backlog) == 0 {
			fmt.Println("No backlog items.")
			return
		}
		for i, item := range state.Backlog {
			fmt.Printf("  %d. %s\n", i+1, item)
		}

	case "add":
		if len(rest) == 0 {
			fmt.Fprintln(os.Stderr, "Error: text required. Usage: coder todo add <text>")
			os.Exit(1)
		}
		text := strings.Join(rest, " ")
		state.Backlog = append(state.Backlog, text)
		saveState(state)
		fmt.Printf("Added: %s\n", text)

	case "done", "remove", "rm":
		if len(rest) == 0 {
			fmt.Fprintln(os.Stderr, "Error: text required. Usage: coder todo done <text>")
			os.Exit(1)
		}
		needle := strings.ToLower(strings.Join(rest, " "))
		var kept []string
		removed := 0
		for _, item := range state.Backlog {
			if strings.Contains(strings.ToLower(item), needle) {
				removed++
			} else {
				kept = append(kept, item)
			}
		}
		state.Backlog = kept
		saveState(state)
		fmt.Printf("Removed %d item(s).\n", removed)

	case "clear":
		state.Backlog = nil
		saveState(state)
		fmt.Println("Backlog cleared.")

	default:
		fmt.Fprintf(os.Stderr, "Unknown action: %q\n\n", action)
		fmt.Fprint(os.Stderr, todoUsage)
		os.Exit(1)
	}
}

// ── coder stats ───────────────────────────────────────────────────────────────

const statsUsage = `Usage: coder stats [flags]

Show project statistics: phases, commits, plans, and files.

FLAGS:
`

func runStats(args []string) {
	fs := flag.NewFlagSet("stats", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, statsUsage)
		fs.PrintDefaults()
	}
	fs.Parse(args)

	logActivity("stats")

	fmt.Printf("\n── Project Stats ──────────────────────────────────────\n\n")

	// Roadmap phases
	roadmap, _ := loadRoadmap()
	donePhases := 0
	for _, rp := range roadmap {
		if rp.Status == "done" || rp.Status == "shipped" {
			donePhases++
		}
	}
	fmt.Printf("  Phases     : %d total, %d done\n", len(roadmap), donePhases)

	// .coder/phases/ files
	phasesDir := coderPath("phases")
	plans := 0
	summaries := 0
	if entries, err := os.ReadDir(phasesDir); err == nil {
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), "-PLAN.md") {
				plans++
			}
			if strings.HasSuffix(e.Name(), "-SUMMARY.md") {
				summaries++
			}
		}
	}
	fmt.Printf("  Plans      : %d\n", plans)
	fmt.Printf("  Summaries  : %d / %d\n", summaries, plans)

	// Git stats
	commitCount := ""
	if out, err := exec.Command("git", "rev-list", "--count", "HEAD").Output(); err == nil {
		commitCount = strings.TrimSpace(string(out))
	}
	if commitCount != "" {
		fmt.Printf("  Git commits: %s\n", commitCount)
	}

	// File count
	fileCount := 0
	filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			name := info.Name()
			if name == ".git" || name == "node_modules" || name == "vendor" || name == ".coder" {
				return filepath.SkipDir
			}
		} else {
			fileCount++
		}
		return nil
	})
	fmt.Printf("  Source files: %d\n", fileCount)

	// STATE.md info
	if state, err := loadState(); err == nil {
		fmt.Printf("\n  Current phase: %d  step: %s\n", state.CurrentPhase, state.Step)
		fmt.Printf("  Last action  : %s\n", state.LastAction)
	}

	fmt.Printf("\n──────────────────────────────────────────────────────\n\n")
}

// ── coder health ──────────────────────────────────────────────────────────────

const healthUsage = `Usage: coder health [flags]

Check project health: missing artifacts, blockers, stale state.

FLAGS:
`

func runHealth(args []string) {
	fs := flag.NewFlagSet("health", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, healthUsage)
		fs.PrintDefaults()
	}
	fs.Parse(args)

	logActivity("health")

	issues := 0

	fmt.Printf("\n── Project Health ─────────────────────────────────────\n\n")

	// Check PROJECT.md
	if !projectExists() {
		fmt.Println("  ✗ .coder/PROJECT.md missing — run: coder new-project")
		issues++
	} else {
		fmt.Println("  ✓ Project initialized")
	}

	// Check ROADMAP.md
	roadmap, roadmapErr := loadRoadmap()
	if roadmapErr != nil {
		fmt.Println("  ✗ .coder/ROADMAP.md missing — run: coder new-project")
		issues++
	} else {
		fmt.Printf("  ✓ Roadmap: %d phases\n", len(roadmap))
	}

	// Check STATE.md
	state, stateErr := loadState()
	if stateErr != nil {
		fmt.Println("  ⚠ .coder/STATE.md missing or unreadable")
		issues++
	} else {
		// Stale state check
		if !state.Updated.IsZero() && time.Since(state.Updated) > 7*24*time.Hour {
			fmt.Printf("  ⚠ State is stale (%s) — check progress\n", state.Updated.Format("2006-01-02"))
			issues++
		} else {
			fmt.Printf("  ✓ State: phase=%d step=%s\n", state.CurrentPhase, state.Step)
		}

		// Blockers
		if len(state.Blockers) > 0 {
			fmt.Printf("  ⚠ %d blocker(s) recorded:\n", len(state.Blockers))
			for _, b := range state.Blockers {
				fmt.Printf("    - %s\n", b)
			}
			issues++
		}
	}

	// Check current phase artifacts
	if state != nil && state.CurrentPhase > 0 {
		phase := state.CurrentPhase
		phasePfx := fmt.Sprintf("%02d", phase)

		contextPath := coderPath("phases", phasePfx+"-CONTEXT.md")
		if _, err := os.Stat(contextPath); os.IsNotExist(err) {
			fmt.Printf("  ⚠ CONTEXT.md for phase %d not found — run: coder discuss-phase %d\n", phase, phase)
			issues++
		}
	}

	fmt.Println()
	if issues == 0 {
		fmt.Println("  ✓ All health checks passed")
	} else {
		fmt.Printf("  %d issue(s) found\n", issues)
	}
	fmt.Printf("\n──────────────────────────────────────────────────────\n\n")
}

// ── coder note ────────────────────────────────────────────────────────────────

const noteUsage = `Usage: coder note <text> [flags]

Record a project note/decision to .coder/STATE.md.

EXAMPLES:
  coder note "decided to use JWT with refresh tokens"
  coder note --blocker "waiting for API credentials from client"

FLAGS:
`

func runNote(args []string) {
	fs := flag.NewFlagSet("note", flag.ExitOnError)
	blocker := fs.Bool("blocker", false, "Record as a blocker instead of a decision")
	backlog := fs.Bool("backlog", false, "Record as a backlog item")

	fs.Usage = func() {
		fmt.Fprint(os.Stderr, noteUsage)
		fs.PrintDefaults()
	}
	fs.Parse(args)

	logActivity("note")

	rest := fs.Args()
	if len(rest) == 0 {
		fmt.Fprintln(os.Stderr, "Error: note text required. Usage: coder note <text>")
		os.Exit(1)
	}

	text := strings.Join(rest, " ")
	ts := time.Now().Format("2006-01-02")

	state, err := loadState()
	if err != nil {
		state = &ProjectState{PRs: make(map[int]string)}
	}

	if *blocker {
		entry := fmt.Sprintf("[%s] %s", ts, text)
		state.Blockers = append(state.Blockers, entry)
		saveState(state)
		fmt.Printf("Blocker recorded: %s\n", text)
	} else if *backlog {
		state.Backlog = append(state.Backlog, text)
		saveState(state)
		fmt.Printf("Backlog item added: %s\n", text)
	} else {
		entry := fmt.Sprintf("[%s] %s", ts, text)
		state.Decisions = append(state.Decisions, entry)
		saveState(state)
		fmt.Printf("Decision recorded: %s\n", text)
	}
}

// ── coder do ──────────────────────────────────────────────────────────────────

const doUsage = `Usage: coder do <task description> [flags]

Run a one-off AI task in the context of your project.
Injects project context (STATE.md, REQUIREMENTS.md) before executing.

EXAMPLES:
  coder do "write unit tests for the auth service"
  coder do "refactor the payment module to use dependency injection"

FLAGS:
`

func runDo(args []string) {
	fs := flag.NewFlagSet("do", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, doUsage)
		fs.PrintDefaults()
	}
	fs.Parse(args)

	logActivity("do")

	rest := fs.Args()
	if len(rest) == 0 {
		fmt.Fprintln(os.Stderr, "Error: task description required.\nUsage: coder do \"<task>\"")
		os.Exit(1)
	}

	task := strings.Join(rest, " ")

	// Build context
	var ctx strings.Builder
	if projectContent := readFileOrEmpty(coderPath("PROJECT.md")); projectContent != "" {
		ctx.WriteString("Project context:\n")
		ctx.WriteString(truncate(projectContent, 1000))
		ctx.WriteString("\n\n")
	}
	if reqContent := readFileOrEmpty(coderPath("REQUIREMENTS.md")); reqContent != "" {
		ctx.WriteString("Requirements:\n")
		ctx.WriteString(truncate(reqContent, 1000))
		ctx.WriteString("\n\n")
	}
	if state, err := loadState(); err == nil {
		ctx.WriteString(fmt.Sprintf("Current phase: %d, step: %s\n\n", state.CurrentPhase, state.Step))
	}

	prompt := fmt.Sprintf("%sTask:\n%s\n\nExecute this task now. Show what you're doing step by step.", ctx.String(), task)

	cfg, _ := loadConfig()
	if cfg == nil {
		cfg = &Config{}
	}
	chatClient := getChatClient(cfg)
	bgCtx := context.Background()

	fmt.Printf("\nExecuting: %s\n\n", task)
	fmt.Println(strings.Repeat("─", 54))

	_, err := chatClient.ChatStream(bgCtx, prompt, "", true, true, func(delta string) {
		fmt.Print(delta)
	})
	fmt.Printf("\n%s\n", strings.Repeat("─", 54))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
