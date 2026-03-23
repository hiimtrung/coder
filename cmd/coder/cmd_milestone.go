package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

const milestoneUsage = `Usage: coder milestone <action> [N] [flags]

Manage phase lifecycle: audit, complete, archive, and start next.

ACTIONS:
  audit [N]     Show completion status for phase N (or current phase)
  complete [N]  Mark phase N as done, update STATE.md
  archive [N]   Move phase N files to .coder/archive/
  next          Advance to the next phase

EXAMPLES:
  coder milestone audit 3
  coder milestone complete 3
  coder milestone archive 2
  coder milestone next

FLAGS:
`

func runMilestone(args []string) {
	fs := flag.NewFlagSet("milestone", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, milestoneUsage)
		fs.PrintDefaults()
	}
	fs.Parse(args)

	logActivity("milestone")

	rest := fs.Args()
	if len(rest) == 0 {
		fmt.Fprint(os.Stderr, milestoneUsage)
		os.Exit(1)
	}

	action := rest[0]
	phaseArg := ""
	if len(rest) > 1 {
		phaseArg = rest[1]
	}

	switch action {
	case "audit":
		runMilestoneAudit(phaseArg)
	case "complete":
		runMilestoneComplete(phaseArg)
	case "archive":
		runMilestoneArchive(phaseArg)
	case "next":
		runMilestoneNext()
	default:
		fmt.Fprintf(os.Stderr, "Unknown milestone action: %q\n\n", action)
		fmt.Fprint(os.Stderr, milestoneUsage)
		os.Exit(1)
	}
}

func resolvePhaseNum(arg string) int {
	phaseNum := 0
	if arg != "" {
		fmt.Sscanf(arg, "%d", &phaseNum)
	}
	if phaseNum == 0 {
		if state, err := loadState(); err == nil {
			phaseNum = state.CurrentPhase
		}
	}
	if phaseNum == 0 {
		fmt.Fprintln(os.Stderr, "Error: phase number required.")
		os.Exit(1)
	}
	return phaseNum
}

func runMilestoneAudit(phaseArg string) {
	phaseNum := resolvePhaseNum(phaseArg)
	phasePfx := fmt.Sprintf("%02d", phaseNum)

	phaseName := fmt.Sprintf("Phase %d", phaseNum)
	if roadmap, err := loadRoadmap(); err == nil {
		for _, rp := range roadmap {
			if rp.Number == phaseNum {
				phaseName = rp.Name
				break
			}
		}
	}

	fmt.Printf("\n── Milestone Audit: Phase %d — %s ──\n\n", phaseNum, phaseName)

	// Check required artifacts
	artifacts := []struct {
		path  string
		label string
	}{
		{coderPath("phases", phasePfx+"-CONTEXT.md"), "CONTEXT.md"},
		{coderPath("phases", phasePfx+"-VERIFICATION.md"), "VERIFICATION.md"},
	}

	// Count PLAN.md and SUMMARY.md files
	planCount := 0
	summaryCount := 0
	if entries, err := os.ReadDir(coderPath("phases")); err == nil {
		for _, e := range entries {
			n := e.Name()
			if strings.HasPrefix(n, phasePfx+"-") {
				if strings.HasSuffix(n, "-PLAN.md") {
					planCount++
				}
				if strings.HasSuffix(n, "-SUMMARY.md") {
					summaryCount++
				}
			}
		}
	}

	for _, a := range artifacts {
		if _, err := os.Stat(a.path); err == nil {
			fmt.Printf("  ✓ %s\n", a.label)
		} else {
			fmt.Printf("  ✗ %s (missing)\n", a.label)
		}
	}
	fmt.Printf("  ✓ Plans   : %d found\n", planCount)
	fmt.Printf("  %s Summaries: %d / %d\n", checkMark(summaryCount == planCount), summaryCount, planCount)

	// Check PR
	if state, err := loadState(); err == nil {
		if url, ok := state.PRs[phaseNum]; ok && url != "" {
			fmt.Printf("  ✓ PR       : %s\n", url)
		} else {
			fmt.Printf("  ✗ PR       : not created yet\n")
		}
	}

	fmt.Println()
}

func checkMark(ok bool) string {
	if ok {
		return "✓"
	}
	return "✗"
}

func runMilestoneComplete(phaseArg string) {
	phaseNum := resolvePhaseNum(phaseArg)

	phaseName := fmt.Sprintf("Phase %d", phaseNum)
	if roadmap, err := loadRoadmap(); err == nil {
		for _, rp := range roadmap {
			if rp.Number == phaseNum {
				phaseName = rp.Name
				break
			}
		}
	}

	state, err := loadState()
	if err != nil {
		state = &ProjectState{PRs: make(map[int]string)}
	}

	state.Step = "done"
	state.LastAction = fmt.Sprintf("completed phase %d — %s", phaseNum, time.Now().Format("2006-01-02"))

	// Add decision record
	decision := fmt.Sprintf("Phase %d (%s) completed on %s", phaseNum, phaseName, time.Now().Format("2006-01-02"))
	state.Decisions = append(state.Decisions, decision)

	if err := saveState(state); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving state: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Phase %d marked as complete.\n", phaseNum)
	fmt.Printf("Next: coder milestone next\n")
}

func runMilestoneArchive(phaseArg string) {
	phaseNum := resolvePhaseNum(phaseArg)
	phasePfx := fmt.Sprintf("%02d", phaseNum)

	archiveDir := coderPath("archive", phasePfx)
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating archive dir: %v\n", err)
		os.Exit(1)
	}

	phasesDir := coderPath("phases")
	entries, err := os.ReadDir(phasesDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading phases dir: %v\n", err)
		os.Exit(1)
	}

	moved := 0
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), phasePfx+"-") {
			src := coderPath("phases", e.Name())
			dst := coderPath("archive", phasePfx, e.Name())
			if err := os.Rename(src, dst); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not move %s: %v\n", e.Name(), err)
			} else {
				moved++
			}
		}
	}

	fmt.Printf("Archived %d files for phase %d → .coder/archive/%s/\n", moved, phaseNum, phasePfx)
}

func runMilestoneNext() {
	state, err := loadState()
	if err != nil {
		fmt.Println("No project state. Run: coder new-project")
		return
	}

	nextPhase := state.CurrentPhase + 1
	roadmap, _ := loadRoadmap()

	// Validate next phase exists
	found := false
	nextName := ""
	for _, rp := range roadmap {
		if rp.Number == nextPhase {
			found = true
			nextName = rp.Name
			break
		}
	}

	if !found {
		fmt.Println("All phases complete! Your project is done.")
		// Run final git tag
		fmt.Print("Create release tag? [y/N] ")
		var ans string
		fmt.Scanln(&ans)
		if strings.TrimSpace(strings.ToLower(ans)) == "y" {
			tag := fmt.Sprintf("v1.0.0-%s", time.Now().Format("20060102"))
			cmd := exec.Command("git", "tag", "-a", tag, "-m", fmt.Sprintf("Project complete %s", time.Now().Format("2006-01-02")))
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Run()
			fmt.Printf("Tagged: %s\n", tag)
		}
		return
	}

	// Advance state
	state.CurrentPhase = nextPhase
	state.Step = "discuss"
	state.LastAction = fmt.Sprintf("advanced to phase %d", nextPhase)
	if err := saveState(state); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving state: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Advanced to Phase %d: %s\n", nextPhase, nextName)
	fmt.Printf("Next: coder discuss-phase %d\n", nextPhase)
}
