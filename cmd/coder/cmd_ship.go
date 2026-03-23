package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

const shipUsage = `Usage: coder ship [N] [flags]

Create a pull request for phase N (or current phase) using gh CLI.
Generates a structured PR body from phase summaries and VERIFICATION.md.

EXAMPLES:
  coder ship            # ship current phase
  coder ship 3          # ship phase 3 explicitly
  coder ship --draft    # open as draft PR
  coder ship --base main --title "feat: phase 3 auth module"

FLAGS:
`

func runShip(args []string) {
	fs := flag.NewFlagSet("ship", flag.ExitOnError)
	draft := fs.Bool("draft", false, "Open PR as draft")
	base := fs.String("base", "main", "Base branch for the PR")
	title := fs.String("title", "", "Custom PR title (auto-generated if empty)")
	skipPush := fs.Bool("skip-push", false, "Skip git push before creating PR")

	fs.Usage = func() {
		fmt.Fprint(os.Stderr, shipUsage)
		fs.PrintDefaults()
	}
	fs.Parse(args)

	logActivity("ship")

	// Resolve phase number
	phaseNum := 0
	if len(fs.Args()) > 0 {
		fmt.Sscanf(fs.Args()[0], "%d", &phaseNum)
	}
	if phaseNum == 0 {
		state, err := loadState()
		if err == nil && state.CurrentPhase > 0 {
			phaseNum = state.CurrentPhase
		}
	}
	if phaseNum == 0 {
		fmt.Fprintln(os.Stderr, "Error: could not determine phase number.\nUsage: coder ship <N>")
		os.Exit(1)
	}

	phasePfx := fmt.Sprintf("%02d", phaseNum)

	// Resolve phase name
	phaseName := fmt.Sprintf("Phase %d", phaseNum)
	if roadmap, err := loadRoadmap(); err == nil {
		for _, rp := range roadmap {
			if rp.Number == phaseNum {
				phaseName = rp.Name
				break
			}
		}
	}

	fmt.Printf("\n╔══════════════════════════════════════════════════════╗\n")
	fmt.Printf("║  ship  ·  Phase %-2d  ·  %-34s║\n", phaseNum, truncate(phaseName, 34))
	fmt.Printf("╚══════════════════════════════════════════════════════╝\n\n")

	// Push current branch
	if !*skipPush {
		fmt.Printf("Pushing branch...\n")
		pushCmd := exec.Command("git", "push", "--set-upstream", "origin", "HEAD")
		pushCmd.Stdout = os.Stdout
		pushCmd.Stderr = os.Stderr
		if err := pushCmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: git push failed: %v\n", err)
			fmt.Println("Continuing anyway...")
		}
	}

	// Collect phase summaries
	phasesDir := coderPath("phases")
	entries, _ := os.ReadDir(phasesDir)
	var summaries []string
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, phasePfx+"-") && strings.HasSuffix(name, "-SUMMARY.md") {
			data, err := os.ReadFile(coderPath("phases", name))
			if err == nil {
				summaries = append(summaries, string(data))
			}
		}
	}

	// Collect VERIFICATION.md
	verifyContent := readFileOrEmpty(coderPath("phases", phasePfx+"-VERIFICATION.md"))

	// Generate PR body with AI
	cfg, _ := loadConfig()
	if cfg == nil {
		cfg = &Config{}
	}
	chatClient := getChatClient(cfg)
	ctx := context.Background()

	prPrompt := fmt.Sprintf(`Generate a GitHub PR body for Phase %d: %s

Summaries of completed plans:
%s

Verification:
%s

Write a PR description with these sections:
## Summary
(2-4 bullet points of what was built)

## Changes
(files and components touched, brief descriptions)

## Test plan
(checklist for reviewer to verify the PR)

## Notes
(any caveats, follow-ups, known issues)

Keep it concise and professional. Use present tense for summary bullets.`,
		phaseNum, phaseName,
		truncate(strings.Join(summaries, "\n\n---\n\n"), 3000),
		truncate(verifyContent, 1000),
	)

	fmt.Println("Generating PR description...")
	var prBody strings.Builder
	_, err := chatClient.ChatStream(ctx, prPrompt, "", false, false, func(delta string) {
		fmt.Print(delta)
		prBody.WriteString(delta)
	})
	fmt.Println()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: AI PR generation failed: %v\n", err)
		prBody.WriteString(fmt.Sprintf("## Summary\n\nPhase %d: %s\n\n## Test plan\n\n- [ ] Verify all acceptance criteria\n", phaseNum, phaseName))
	}

	// Build PR title
	prTitle := *title
	if prTitle == "" {
		prTitle = fmt.Sprintf("feat: phase %d — %s", phaseNum, strings.ToLower(phaseName))
		if len(prTitle) > 72 {
			prTitle = fmt.Sprintf("feat: phase %d — %s", phaseNum, truncate(strings.ToLower(phaseName), 60))
		}
	}

	// Confirm
	fmt.Printf("\nPR title: %s\n", prTitle)
	fmt.Printf("Base branch: %s\n", *base)
	if *draft {
		fmt.Println("Mode: draft")
	}
	fmt.Print("\nCreate PR? [Y/n] ")
	var ans string
	fmt.Scanln(&ans)
	ans = strings.TrimSpace(strings.ToLower(ans))
	if ans == "n" || ans == "no" {
		fmt.Println("Cancelled.")
		return
	}

	// Build gh pr create args
	ghArgs := []string{"pr", "create",
		"--title", prTitle,
		"--body", prBody.String(),
		"--base", *base,
	}
	if *draft {
		ghArgs = append(ghArgs, "--draft")
	}

	fmt.Println("\nCreating PR...")
	ghCmd := exec.Command("gh", ghArgs...)
	ghOut, err := ghCmd.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: gh pr create failed: %v\n", err)
		if exitErr, ok := err.(*exec.ExitError); ok {
			fmt.Fprintln(os.Stderr, string(exitErr.Stderr))
		}
		os.Exit(1)
	}

	prURL := strings.TrimSpace(string(ghOut))
	fmt.Printf("\nPR created: %s\n", prURL)

	// Save PR URL to STATE.md
	state, err := loadState()
	if err != nil {
		state = &ProjectState{PRs: make(map[int]string)}
	}
	if state.PRs == nil {
		state.PRs = make(map[int]string)
	}
	state.PRs[phaseNum] = prURL
	state.Step = "ship"
	state.LastAction = fmt.Sprintf("shipped phase %d — %s", phaseNum, time.Now().Format("2006-01-02"))
	if saveErr := saveState(state); saveErr != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not save state: %v\n", saveErr)
	}

	fmt.Printf("\nNext: coder milestone complete %d\n", phaseNum)
}
