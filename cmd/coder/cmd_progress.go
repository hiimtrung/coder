package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

const progressUsage = `Usage: coder progress [flags]

Show current project progress — active phase, step, decisions, blockers.
Reads .coder/STATE.md and .coder/ROADMAP.md.

FLAGS:
`

const nextUsage = `Usage: coder next [flags]

Print the next recommended command based on current project state.
Reads .coder/STATE.md to determine phase and step.

FLAGS:
`

func runProgress(args []string) {
	fs := flag.NewFlagSet("progress", flag.ExitOnError)
	short := fs.Bool("short", false, "Print one-line summary only")

	fs.Usage = func() {
		fmt.Fprint(os.Stderr, progressUsage)
		fs.PrintDefaults()
	}
	fs.Parse(args)

	logActivity("progress")

	if !projectExists() {
		fmt.Fprintln(os.Stderr, "No project found. Run: coder new-project")
		os.Exit(1)
	}

	state, err := loadState()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not read STATE.md: %v\n", err)
		os.Exit(1)
	}

	if *short {
		fmt.Printf("phase=%d step=%s last=%s\n", state.CurrentPhase, state.Step, state.LastAction)
		return
	}

	// Load roadmap for phase list
	roadmap, _ := loadRoadmap()

	fmt.Printf("\n╔══════════════════════════════════════════════════════╗\n")
	fmt.Printf("║  PROJECT PROGRESS                                    ║\n")
	fmt.Printf("╚══════════════════════════════════════════════════════╝\n\n")

	fmt.Printf("  Project    : %s\n", state.Project)
	fmt.Printf("  Phase      : %d\n", state.CurrentPhase)
	fmt.Printf("  Step       : %s\n", state.Step)
	fmt.Printf("  Last action: %s\n", state.LastAction)
	if !state.Updated.IsZero() {
		fmt.Printf("  Updated    : %s\n", state.Updated.Format("2006-01-02 15:04"))
	}

	// Roadmap overview
	if len(roadmap) > 0 {
		fmt.Printf("\n  ── Roadmap ──────────────────────────────────────────\n")
		for _, rp := range roadmap {
			marker := "  "
			if rp.Number == state.CurrentPhase {
				marker = "→ "
			} else if rp.Status == "done" || rp.Status == "shipped" {
				marker = "✓ "
			}
			statusTag := ""
			if rp.Status != "planned" && rp.Status != "" {
				statusTag = fmt.Sprintf("  [%s]", rp.Status)
			}
			fmt.Printf("  %sPhase %d: %s%s\n", marker, rp.Number, rp.Name, statusTag)
		}
	}

	// PRs
	if len(state.PRs) > 0 {
		fmt.Printf("\n  ── Pull Requests ────────────────────────────────────\n")
		for phase, url := range state.PRs {
			fmt.Printf("  Phase %d: %s\n", phase, url)
		}
	}

	// Decisions
	if len(state.Decisions) > 0 {
		fmt.Printf("\n  ── Decisions ────────────────────────────────────────\n")
		for _, d := range state.Decisions {
			fmt.Printf("  • %s\n", d)
		}
	}

	// Blockers
	if len(state.Blockers) > 0 {
		fmt.Printf("\n  ── Blockers ─────────────────────────────────────────\n")
		for _, b := range state.Blockers {
			fmt.Printf("  ⚠ %s\n", b)
		}
	}

	// Backlog
	if len(state.Backlog) > 0 {
		fmt.Printf("\n  ── Backlog ──────────────────────────────────────────\n")
		for _, b := range state.Backlog {
			fmt.Printf("  · %s\n", b)
		}
	}

	fmt.Printf("\n  Next: coder next\n")
	fmt.Printf("╚══════════════════════════════════════════════════════╝\n\n")
}

func runNext(args []string) {
	fs := flag.NewFlagSet("next", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, nextUsage)
		fs.PrintDefaults()
	}
	fs.Parse(args)

	logActivity("next")

	if !projectExists() {
		fmt.Println("coder new-project")
		return
	}

	state, err := loadState()
	if err != nil {
		// No state yet — suggest map-codebase
		fmt.Println("coder map-codebase")
		return
	}

	phase := state.CurrentPhase
	if phase == 0 {
		phase = 1
	}

	cmd := resolveNextCommand(state.Step, phase)
	fmt.Println(cmd)
}

// resolveNextCommand returns the suggested next command based on current step.
func resolveNextCommand(step string, phase int) string {
	switch strings.ToLower(step) {
	case "":
		return "coder map-codebase"
	case "map", "mapped":
		return fmt.Sprintf("coder discuss-phase %d", phase)
	case "discuss", "discussed":
		return fmt.Sprintf("coder plan-phase %d", phase)
	case "plan", "planned":
		return fmt.Sprintf("coder execute-phase %d", phase)
	case "execute", "executing":
		return fmt.Sprintf("coder execute-phase %d --gaps-only", phase)
	case "qa":
		return fmt.Sprintf("coder qa --phase %d", phase)
	case "ship":
		return fmt.Sprintf("coder milestone complete %d", phase)
	case "milestone", "done":
		return fmt.Sprintf("coder discuss-phase %d", phase+1)
	default:
		return fmt.Sprintf("coder discuss-phase %d", phase)
	}
}
