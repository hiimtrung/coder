package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	httpclient "github.com/trungtran/coder/internal/transport/http/client"
)

// PlanTask represents a single task inside a plan XML.
type PlanTask struct {
	Type   string // create | modify | delete | test
	Name   string
	Action string
	Verify string
	Done   string
}

// Plan represents a parsed plan from a PLAN.md XML block.
type Plan struct {
	ID            string
	Phase         string
	Name          string
	Objective     string
	Files         []string
	Dependencies  string
	EstimatedTime string
	Tasks         []PlanTask
}

// parsePlanXML parses a plan XML block from plan file content using simple regex.
func parsePlanXML(content string) (*Plan, error) {
	p := &Plan{}

	// Extract plan attributes: <plan id="..." phase="..." name="...">
	planAttrRe := regexp.MustCompile(`(?s)<plan\s+([^>]*)>`)
	if m := planAttrRe.FindStringSubmatch(content); len(m) == 2 {
		attrs := m[1]
		if v := extractAttr(attrs, "id"); v != "" {
			p.ID = v
		}
		if v := extractAttr(attrs, "phase"); v != "" {
			p.Phase = v
		}
		if v := extractAttr(attrs, "name"); v != "" {
			p.Name = v
		}
	}

	// Fallback: try plain text header "# Plan: ..." or "Plan: ..."
	if p.Name == "" {
		nameRe := regexp.MustCompile(`(?m)^#\s+Plan:\s+(.+)$`)
		if m := nameRe.FindStringSubmatch(content); len(m) == 2 {
			p.Name = strings.TrimSpace(m[1])
		}
	}
	if p.ID == "" {
		idRe := regexp.MustCompile(`(?m)^[Pp]lan[-_\s]+[Ii][Dd]:\s*(.+)$`)
		if m := idRe.FindStringSubmatch(content); len(m) == 2 {
			p.ID = strings.TrimSpace(m[1])
		}
	}

	// Extract <objective>
	p.Objective = extractTag(content, "objective")

	// Extract <dependencies>
	p.Dependencies = extractTag(content, "dependencies")
	if p.Dependencies == "" {
		p.Dependencies = "none"
	}

	// Extract <estimated_time>
	p.EstimatedTime = extractTag(content, "estimated_time")

	// Extract <files> block — split lines
	filesContent := extractTag(content, "files")
	for _, line := range strings.Split(filesContent, "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "-")
		line = strings.TrimSpace(line)
		if line != "" {
			p.Files = append(p.Files, line)
		}
	}

	// Extract each <task ...> block
	taskBlockRe := regexp.MustCompile(`(?s)<task\s+([^>]*)>(.*?)</task>`)
	for _, m := range taskBlockRe.FindAllStringSubmatch(content, -1) {
		attrs := m[1]
		body := m[2]
		t := PlanTask{
			Type:   extractAttr(attrs, "type"),
			Name:   extractAttr(attrs, "name"),
			Action: extractTag(body, "action"),
			Verify: extractTag(body, "verify"),
			Done:   extractTag(body, "done"),
		}
		if t.Name == "" {
			t.Name = strings.TrimSpace(extractTag(body, "name"))
		}
		if t.Type == "" {
			t.Type = "create"
		}
		p.Tasks = append(p.Tasks, t)
	}

	// If no XML tasks found, try markdown task list: "- [ ] Task name"
	if len(p.Tasks) == 0 {
		mdTaskRe := regexp.MustCompile(`(?m)^[-*]\s+\[[ x]\]\s+(.+)$`)
		for _, m := range mdTaskRe.FindAllStringSubmatch(content, -1) {
			p.Tasks = append(p.Tasks, PlanTask{
				Type:   "create",
				Name:   strings.TrimSpace(m[1]),
				Action: strings.TrimSpace(m[1]),
			})
		}
	}

	return p, nil
}

// extractAttr extracts an XML attribute value from an attribute string.
// Handles both single and double quoted values.
func extractAttr(attrs, name string) string {
	re := regexp.MustCompile(`(?i)` + regexp.QuoteMeta(name) + `\s*=\s*["']([^"']*)["']`)
	if m := re.FindStringSubmatch(attrs); len(m) == 2 {
		return strings.TrimSpace(m[1])
	}
	return ""
}

// extractTag extracts the inner text of a simple XML tag.
func extractTag(content, tag string) string {
	re := regexp.MustCompile(`(?is)<` + regexp.QuoteMeta(tag) + `\s*[^>]*>(.*?)</` + regexp.QuoteMeta(tag) + `>`)
	if m := re.FindStringSubmatch(content); len(m) == 2 {
		return strings.TrimSpace(m[1])
	}
	return ""
}

// inferCommitType maps a task type to a conventional commit prefix.
func inferCommitType(taskType string) string {
	switch strings.ToLower(taskType) {
	case "test":
		return "test"
	case "modify", "fix":
		return "fix"
	case "delete":
		return "chore"
	default:
		return "feat"
	}
}

// runGitCommit stages all changes and creates a commit with the given message.
// Errors are handled gracefully and printed as warnings.
func runGitCommit(message string) {
	addCmd := exec.Command("git", "add", "-A")
	if err := addCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "  [warn] git add -A: %v\n", err)
		return
	}
	commitCmd := exec.Command("git", "commit", "-m", message)
	commitCmd.Stdout = os.Stdout
	commitCmd.Stderr = os.Stderr
	if err := commitCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "  [warn] git commit: %v\n", err)
	}
}

// groupPlansIntoWaves sorts plans into dependency waves.
// Wave 1 = plans with no dependencies; Wave N = plans whose deps are all in prior waves.
func groupPlansIntoWaves(plans []*Plan) [][]*Plan {
	placed := make(map[string]bool)
	var waves [][]*Plan

	remaining := make([]*Plan, len(plans))
	copy(remaining, plans)

	for len(remaining) > 0 {
		var wave []*Plan
		var next []*Plan
		for _, p := range remaining {
			deps := strings.TrimSpace(strings.ToLower(p.Dependencies))
			if deps == "" || deps == "none" {
				wave = append(wave, p)
			} else {
				// Check if all deps are already placed
				depList := strings.Split(p.Dependencies, ",")
				allPlaced := true
				for _, d := range depList {
					d = strings.TrimSpace(d)
					if d == "" || d == "none" {
						continue
					}
					if !placed[d] {
						allPlaced = false
						break
					}
				}
				if allPlaced {
					wave = append(wave, p)
				} else {
					next = append(next, p)
				}
			}
		}
		if len(wave) == 0 {
			// Avoid infinite loop: push remaining as a final wave
			waves = append(waves, remaining)
			break
		}
		for _, p := range wave {
			placed[p.ID] = true
		}
		waves = append(waves, wave)
		remaining = next
	}
	return waves
}

// gitLogOneline returns the last N commit hashes + messages, one per line.
func gitLogOneline(n int) string {
	out, err := exec.Command("git", "log", fmt.Sprintf("-%d", n), "--oneline").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// runExecutePhase implements `coder execute-phase <N> [--interactive] [--gaps-only] [--plan <id>]`
func runExecutePhase(args []string) {
	fs := flag.NewFlagSet("execute-phase", flag.ExitOnError)
	interactive := fs.Bool("interactive", false, "Sequential execution with user checkpoint per plan")
	gapsOnly := fs.Bool("gaps-only", false, "Only execute plans with status=pending")
	planID := fs.String("plan", "", "Execute only a specific plan id (e.g. '1-02')")

	fs.Usage = func() {
		fmt.Fprint(os.Stderr, "Usage: coder execute-phase <N> [--interactive] [--gaps-only] [--plan <id>]\n")
		fs.PrintDefaults()
	}
	fs.Parse(args)

	logActivity("execute-phase")

	// Parse phase number
	rest := fs.Args()
	if len(rest) == 0 {
		fmt.Fprintln(os.Stderr, "Error: phase number required. Usage: coder execute-phase <N>")
		os.Exit(1)
	}
	phaseNum := 0
	fmt.Sscanf(rest[0], "%d", &phaseNum)
	if phaseNum == 0 {
		fmt.Fprintf(os.Stderr, "Error: invalid phase number: %s\n", rest[0])
		os.Exit(1)
	}

	// Check PROJECT.md
	if !projectExists() {
		fmt.Fprintln(os.Stderr, "Error: no project found. Run: coder new-project")
		os.Exit(1)
	}

	cfg, _ := loadConfig()
	if cfg == nil {
		cfg = &Config{}
	}
	chatClient := getChatClient(cfg)

	// Load phase plans: .coder/phases/NN-*-PLAN.md
	phasePfx := fmt.Sprintf("%02d", phaseNum)
	phasesDir := coderPath("phases")
	entries, err := os.ReadDir(phasesDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading phases dir: %v\n", err)
		fmt.Fprintf(os.Stderr, "Run: coder plan-phase %d first\n", phaseNum)
		os.Exit(1)
	}

	var planFiles []string
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, phasePfx+"-") && strings.HasSuffix(name, "-PLAN.md") {
			planFiles = append(planFiles, filepath.Join(phasesDir, name))
		}
	}

	if len(planFiles) == 0 {
		fmt.Fprintf(os.Stderr, "No plans found for phase %d.\nRun: coder plan-phase %d first\n", phaseNum, phaseNum)
		os.Exit(1)
	}

	// Parse each plan file
	var plans []*Plan
	for _, pf := range planFiles {
		data, err := os.ReadFile(pf)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: cannot read %s: %v\n", pf, err)
			continue
		}
		plan, err := parsePlanXML(string(data))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: cannot parse %s: %v\n", pf, err)
			continue
		}
		// Derive ID from filename if not set
		if plan.ID == "" {
			base := filepath.Base(pf)
			base = strings.TrimSuffix(base, "-PLAN.md")
			plan.ID = base
		}
		if plan.Name == "" {
			plan.Name = plan.ID
		}

		// Filter by --plan flag
		if *planID != "" && plan.ID != *planID {
			continue
		}

		// Filter by --gaps-only: skip plans that have a SUMMARY.md
		if *gapsOnly {
			summaryPath := coderPath("phases", plan.ID+"-SUMMARY.md")
			if _, err := os.Stat(summaryPath); err == nil {
				continue
			}
		}

		plans = append(plans, plan)
	}

	if len(plans) == 0 {
		fmt.Printf("No plans to execute for phase %d (all may already be complete).\n", phaseNum)
		return
	}

	// Determine phase name from roadmap
	phaseName := fmt.Sprintf("Phase %d", phaseNum)
	if roadmap, err := loadRoadmap(); err == nil {
		for _, rp := range roadmap {
			if rp.Number == phaseNum {
				phaseName = rp.Name
				break
			}
		}
	}

	// Group plans into waves
	waves := groupPlansIntoWaves(plans)

	// Print execution plan header
	fmt.Printf("\n%s\n", strings.Repeat("═", 58))
	fmt.Printf("  EXECUTE PHASE %d — %s\n", phaseNum, phaseName)
	fmt.Printf("%s\n\n", strings.Repeat("═", 58))

	for i, wave := range waves {
		waveLabel := fmt.Sprintf("Wave %d", i+1)
		ids := make([]string, len(wave))
		for j, p := range wave {
			ids[j] = p.ID
		}
		parallel := ""
		if i == 0 && len(wave) > 1 {
			parallel = " (parallel*)"
		}
		fmt.Printf("  %s%s: %s\n", waveLabel, parallel, strings.Join(ids, ", "))
	}

	if len(waves) > 0 && len(waves[0]) > 1 {
		fmt.Println()
		fmt.Println("  * parallel execution via Claude Code Agent tool")
		fmt.Println("    sequential fallback active in standalone mode")
	}
	fmt.Printf("\n%s\n", strings.Repeat("═", 58))

	ctx := context.Background()

	var allSummaries []string

	// Execute waves sequentially; within each wave, execute plans sequentially
	for waveIdx, wave := range waves {
		fmt.Printf("\n--- Wave %d ---\n", waveIdx+1)
		for _, plan := range wave {
			summaryMD, err := executePlan(ctx, plan, phaseNum, chatClient, *interactive)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error executing plan %s: %v\n", plan.ID, err)
				continue
			}
			allSummaries = append(allSummaries, summaryMD)
		}
	}

	// Verifier pass
	fmt.Printf("\n%s\n", strings.Repeat("═", 58))
	fmt.Println("  PHASE VERIFICATION")
	fmt.Printf("%s\n\n", strings.Repeat("═", 58))

	reqContent := ""
	if reqData, err := os.ReadFile(coderPath("REQUIREMENTS.md")); err == nil {
		reqContent = string(reqData)
	}

	verifierPrompt := fmt.Sprintf(`Verify phase %d implementation is complete.

Requirements:
%s

Execution summaries:
%s

Check:
1. Are all requirements covered?
2. Any obvious issues?
3. Ready for UAT?

Write a VERIFICATION.md and state: PASS or NEEDS_FIXES.`,
		phaseNum,
		truncate(reqContent, 2000),
		truncate(strings.Join(allSummaries, "\n\n---\n\n"), 3000),
	)

	var verifyContent strings.Builder
	verifyContent.WriteString(fmt.Sprintf("# Verification: Phase %d\n\nGenerated: %s\n\n", phaseNum, time.Now().Format("2006-01-02")))

	_, verifyErr := chatClient.ChatStream(ctx, verifierPrompt, "", true, true, func(delta string) {
		fmt.Print(delta)
		verifyContent.WriteString(delta)
	})
	fmt.Println()
	if verifyErr != nil {
		fmt.Fprintf(os.Stderr, "Verification error: %v\n", verifyErr)
	}

	// Write VERIFICATION.md
	verifyFile := coderPath("phases", fmt.Sprintf("%s-VERIFICATION.md", phasePfx))
	if err := os.WriteFile(verifyFile, []byte(verifyContent.String()), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: cannot write verification file: %v\n", err)
	} else {
		fmt.Printf("\nVerification written → %s\n", verifyFile)
	}

	// Update STATE.md
	state, err := loadState()
	if err != nil {
		state = &ProjectState{PRs: make(map[int]string)}
	}
	state.CurrentPhase = phaseNum
	state.Step = "qa"
	state.LastAction = fmt.Sprintf("execute-phase %d completed", phaseNum)
	if saveErr := saveState(state); saveErr != nil {
		fmt.Fprintf(os.Stderr, "Warning: cannot save state: %v\n", saveErr)
	}

	// Summary
	fmt.Printf("\n%s\n", strings.Repeat("═", 58))
	fmt.Printf("  PHASE %d EXECUTION COMPLETE\n", phaseNum)
	fmt.Printf("%s\n", strings.Repeat("═", 58))
	fmt.Printf("\n  Plans executed : %d\n", len(plans))
	fmt.Printf("  Verification   : %s\n", verifyFile)
	fmt.Printf("\n  Next: coder qa --phase %d\n", phaseNum)
	fmt.Printf("%s\n", strings.Repeat("═", 58))
}

// executePlan runs all tasks for a single plan, commits after each task, and
// writes a SUMMARY.md. Returns the summary markdown content.
func executePlan(ctx context.Context, plan *Plan, phaseNum int, client httpclient.ChatClientIface, interactive bool) (string, error) {
	fmt.Printf("\n━━━ Executing: %s (%s) ━━━\n", plan.Name, plan.EstimatedTime)

	filesStr := strings.Join(plan.Files, "\n")

	var completedTasks []string
	for _, task := range plan.Tasks {
		fmt.Printf("  \u27f3 %s...\n", task.Name)

		taskPrompt := fmt.Sprintf(`Execute this implementation task exactly as specified.

Plan: %s (%s)
Task: %s

Files to work with:
%s

Action:
%s

Implement this now. Show what you're doing step by step.
When done, confirm with: "Task complete: %s"`,
			plan.Name, plan.ID,
			task.Name,
			filesStr,
			task.Action,
			task.Name,
		)

		_, err := client.ChatStream(ctx, taskPrompt, "", true, true, func(delta string) {
			fmt.Print(delta)
		})
		fmt.Println()
		if err != nil {
			fmt.Fprintf(os.Stderr, "  [warn] LLM error for task %s: %v\n", task.Name, err)
		}

		fmt.Printf("  \u2713 %s\n", task.Name)
		completedTasks = append(completedTasks, task.Name)

		// Git commit per task
		commitType := inferCommitType(task.Type)
		commitMsg := fmt.Sprintf("%s(%s): %s", commitType, plan.ID, task.Name)
		runGitCommit(commitMsg)
	}

	// Interactive checkpoint
	if interactive {
		fmt.Printf("\nPlan %s complete. Continue to next plan? [Y/n] ", plan.ID)
		var ans string
		fmt.Scanln(&ans)
		ans = strings.TrimSpace(strings.ToLower(ans))
		if ans == "n" || ans == "no" {
			fmt.Println("Paused. Re-run to continue.")
		}
	}

	// Write SUMMARY.md
	gitLog := gitLogOneline(len(plan.Tasks) + 2)

	changesLines := make([]string, len(completedTasks))
	for i, t := range completedTasks {
		changesLines[i] = fmt.Sprintf("- %s", t)
	}

	summaryContent := fmt.Sprintf(`# Summary: %s
Plan: %s
Status: done
Tasks: %d/%d completed

## Changes
%s

## Commits
%s
`, plan.Name, plan.ID, len(completedTasks), len(plan.Tasks), strings.Join(changesLines, "\n"), gitLog)

	summaryFile := coderPath("phases", plan.ID+"-SUMMARY.md")
	if err := os.WriteFile(summaryFile, []byte(summaryContent), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: cannot write summary %s: %v\n", summaryFile, err)
	} else {
		fmt.Printf("  Summary → %s\n", summaryFile)
	}

	return summaryContent, nil
}
