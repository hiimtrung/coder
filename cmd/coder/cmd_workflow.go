package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	httpclient "github.com/trungtran/coder/internal/transport/http/client"
)

const workflowUsage = `Usage: coder workflow <feature-description> [flags]

Auto-chain orchestration: plan → review → implement hints → qa → fix → done.
Describe a feature once; coder runs the full development workflow end-to-end.

EXAMPLES:
  coder workflow "implement Redis caching for skill search"
  coder workflow --steps plan,review "refactor auth service"
  coder workflow --prd path/to/feature.md
  coder workflow --resume
  coder workflow --list
  coder workflow --dry-run "add rate limiting"

FLAGS:
`

// workflowStepStatus represents the lifecycle of a single workflow step.
type workflowStepStatus struct {
	Status   string `json:"status"`   // "pending" | "in_progress" | "done" | "skipped"
	Artifact string `json:"artifact"` // path to output file (if any)
	Notes    string `json:"notes"`
}

// workflowState is the persistent state file for a workflow run.
type workflowState struct {
	ID      string    `json:"id"`
	Feature string    `json:"feature"`
	PrdFile string    `json:"prd_file,omitempty"`
	Status  string    `json:"status"` // "plan" | "review" | "implement" | "qa" | "fix" | "done"
	Created time.Time `json:"created"`
	Updated time.Time `json:"updated"`
	Steps   struct {
		Plan      workflowStepStatus `json:"plan"`
		Review    workflowStepStatus `json:"review"`
		Implement workflowStepStatus `json:"implement"`
		QA        workflowStepStatus `json:"qa"`
		Fix       workflowStepStatus `json:"fix"`
	} `json:"steps"`
}

func runWorkflow(args []string) {
	fs := flag.NewFlagSet("workflow", flag.ExitOnError)
	steps := fs.String("steps", "", "Comma-separated steps to run: plan,review,implement,qa,fix")
	prdFile := fs.String("prd", "", "Read feature description from a PRD markdown file")
	resume := fs.Bool("resume", false, "Resume last in-progress workflow")
	list := fs.Bool("list", false, "List existing workflows")
	dryRun := fs.Bool("dry-run", false, "Show plan only, do not execute")

	fs.Usage = func() {
		fmt.Fprint(os.Stderr, workflowUsage)
		fs.PrintDefaults()
	}
	fs.Parse(args)

	logActivity("workflow")

	cfg, _ := loadConfig()
	if cfg == nil {
		cfg = &Config{}
	}

	if *list {
		listWorkflows()
		return
	}

	if *resume {
		wf := findLastWorkflow()
		if wf == nil {
			fmt.Println("No in-progress workflow found. Start one with: coder workflow <feature>")
			return
		}
		runWorkflowSteps(wf, *steps, *dryRun, cfg)
		return
	}

	// Gather feature description
	var feature string
	if *prdFile != "" {
		data, err := os.ReadFile(*prdFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading PRD: %v\n", err)
			os.Exit(1)
		}
		feature = string(data)
	} else if len(fs.Args()) > 0 {
		feature = strings.Join(fs.Args(), " ")
	} else {
		fmt.Fprint(os.Stderr, "Usage: coder workflow <feature-description> [flags]\nRun `coder workflow --help` for details.\n")
		os.Exit(1)
	}

	// Create new workflow state
	now := time.Now()
	slug := toSlug(feature)
	id := fmt.Sprintf("wf-%d", now.Unix())
	wfFile := filepath.Join(".coder", "workflows", fmt.Sprintf("WF-%s-%s.json", slug, now.Format("2006-01-02")))

	wf := &workflowState{
		ID:      id,
		Feature: feature,
		PrdFile: *prdFile,
		Status:  "plan",
		Created: now,
		Updated: now,
	}
	wf.Steps.Plan.Status = "pending"
	wf.Steps.Review.Status = "pending"
	wf.Steps.Implement.Status = "pending"
	wf.Steps.QA.Status = "pending"
	wf.Steps.Fix.Status = "pending"

	os.MkdirAll(filepath.Dir(wfFile), 0755)
	wf.Steps.Plan.Artifact = wfFile
	saveWorkflowState(wf, wfFile)

	// Store wfFile path in state for resume
	wf.Steps.Plan.Artifact = wfFile

	runWorkflowSteps(wf, *steps, *dryRun, cfg)
}

// runWorkflowSteps executes all (or selected) workflow steps.
func runWorkflowSteps(wf *workflowState, stepsFlag string, dryRun bool, cfg *Config) {
	enabledSteps := parseWorkflowSteps(stepsFlag)

	title := truncate(wf.Feature, 48)
	fmt.Printf("\n╔══════════════════════════════════════════════════════╗\n")
	fmt.Printf("║  coder workflow  ·  %-34s║\n", title)
	fmt.Printf("╚══════════════════════════════════════════════════════╝\n\n")

	if dryRun {
		fmt.Printf("DRY RUN — showing plan only, no steps will execute.\n\n")
	}

	wfFile := findWorkflowFile(wf.ID)

	baseURL := cfg.Memory.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	chatClient := httpclient.NewChatClient(baseURL, cfg.Auth.AccessToken)

	scanner := bufio.NewScanner(os.Stdin)

	// ─────────────────────────────────────────────
	// Step 1: PLAN
	// ─────────────────────────────────────────────
	if shouldRun("plan", enabledSteps) && wf.Steps.Plan.Status != "done" {
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println("  STEP 1 — PLAN")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

		planPath := wf.Steps.Plan.Artifact
		if !strings.HasSuffix(planPath, ".md") {
			slug := toSlug(wf.Feature)
			planPath = filepath.Join(".coder", "plans", fmt.Sprintf("PLAN-%s.md", slug))
		}

		if dryRun {
			fmt.Printf("  [DRY RUN] Would generate plan → %s\n\n", planPath)
			wf.Steps.Plan.Status = "skipped"
		} else {
			wf.Steps.Plan.Status = "in_progress"
			wf.Status = "plan"
			saveWorkflowState(wf, wfFile)

			fmt.Printf("\nGenerating plan (auto mode)...\n\n")
			planContent := generatePlan(wf.Feature, chatClient)
			if planContent == "" {
				fmt.Fprintln(os.Stderr, "Plan generation failed.")
				os.Exit(1)
			}

			os.MkdirAll(filepath.Dir(planPath), 0755)
			if err := os.WriteFile(planPath, []byte(planContent), 0644); err != nil {
				fmt.Fprintf(os.Stderr, "Error saving plan: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("\nPlan saved → %s\n", planPath)
			wf.Steps.Plan.Status = "done"
			wf.Steps.Plan.Artifact = planPath
			wf.Updated = time.Now()
			saveWorkflowState(wf, wfFile)
		}
	}

	// ─────────────────────────────────────────────
	// Step 2: REVIEW PLAN
	// ─────────────────────────────────────────────
	if shouldRun("review", enabledSteps) && wf.Steps.Review.Status != "done" {
		fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println("  STEP 2 — REVIEW PLAN")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

		planPath := wf.Steps.Plan.Artifact
		if dryRun || planPath == "" || !strings.HasSuffix(planPath, ".md") {
			fmt.Println("  [DRY RUN] Would review plan for completeness and risks.")
			wf.Steps.Review.Status = "skipped"
		} else {
			wf.Steps.Review.Status = "in_progress"
			wf.Status = "review"
			saveWorkflowState(wf, wfFile)

			planData, err := os.ReadFile(planPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Cannot read plan file: %v\n", err)
			} else {
				fmt.Printf("\nReviewing plan for risks and completeness...\n\n")
				reviewPlan(string(planData), chatClient)
			}

			// ── Checkpoint ──────────────────────────────────────
			fmt.Println("\n" + strings.Repeat("─", 58))
			fmt.Print("Proceed with this plan? [Y/n/edit] › ")
			scanner.Scan()
			ans := strings.TrimSpace(strings.ToLower(scanner.Text()))
			if ans == "n" || ans == "no" {
				fmt.Println("Workflow stopped at plan review. Edit the plan and run `coder workflow --resume`.")
				wf.Steps.Review.Status = "pending"
				saveWorkflowState(wf, wfFile)
				return
			}

			wf.Steps.Review.Status = "done"
			wf.Updated = time.Now()
			saveWorkflowState(wf, wfFile)
		}
	}

	// ─────────────────────────────────────────────
	// Step 3: IMPLEMENT (hints)
	// ─────────────────────────────────────────────
	if shouldRun("implement", enabledSteps) && wf.Steps.Implement.Status != "done" {
		fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println("  STEP 3 — IMPLEMENTATION CHECKLIST")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

		planPath := wf.Steps.Plan.Artifact

		if dryRun || planPath == "" {
			fmt.Println("  [DRY RUN] Would generate implementation checklist.")
			wf.Steps.Implement.Status = "skipped"
		} else {
			wf.Steps.Implement.Status = "in_progress"
			wf.Status = "implement"
			saveWorkflowState(wf, wfFile)

			planData, _ := os.ReadFile(planPath)
			fmt.Printf("\nGenerating implementation checklist...\n\n")
			generateImplementChecklist(wf.Feature, string(planData), chatClient)

			fmt.Println("\n" + strings.Repeat("─", 58))
			fmt.Print("Done implementing? Press Enter to continue to QA › ")
			scanner.Scan()

			wf.Steps.Implement.Status = "done"
			wf.Updated = time.Now()
			saveWorkflowState(wf, wfFile)
		}
	}

	// ─────────────────────────────────────────────
	// Step 4: QA
	// ─────────────────────────────────────────────
	if shouldRun("qa", enabledSteps) && wf.Steps.QA.Status != "done" {
		fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println("  STEP 4 — QA / UAT VERIFICATION")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

		planPath := wf.Steps.Plan.Artifact

		if dryRun || planPath == "" {
			fmt.Println("  [DRY RUN] Would launch QA session from plan acceptance criteria.")
			wf.Steps.QA.Status = "skipped"
		} else {
			wf.Steps.QA.Status = "in_progress"
			wf.Status = "qa"
			saveWorkflowState(wf, wfFile)

			fmt.Printf("\nLaunching QA session from plan: %s\n", planPath)
			fmt.Println("Running: coder qa --plan " + planPath)
			fmt.Println(strings.Repeat("─", 58))

			// Invoke the QA flow inline
			runQA([]string{"--plan", planPath})

			wf.Steps.QA.Status = "done"
			wf.Updated = time.Now()
			saveWorkflowState(wf, wfFile)
		}
	}

	// ─────────────────────────────────────────────
	// Step 5: FIX (if issues found)
	// ─────────────────────────────────────────────
	if shouldRun("fix", enabledSteps) && wf.Steps.Fix.Status != "done" {
		fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println("  STEP 5 — FIX & RETRY")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println()

		if dryRun {
			fmt.Println("  [DRY RUN] Would run debug + fix loop on any QA issues.")
			wf.Steps.Fix.Status = "skipped"
		} else {
			fmt.Print("Were there any issues in QA? [y/N] › ")
			scanner.Scan()
			ans := strings.TrimSpace(strings.ToLower(scanner.Text()))
			if ans == "y" || ans == "yes" {
				wf.Steps.Fix.Status = "in_progress"
				wf.Status = "fix"
				saveWorkflowState(wf, wfFile)

				fmt.Println("\nDescribe the issue (or paste error output):")
				fmt.Print("> ")
				scanner.Scan()
				issue := strings.TrimSpace(scanner.Text())

				if issue != "" {
					fmt.Printf("\nRunning debug analysis...\n\n")
					runDebug([]string{issue})
				}

				fmt.Print("\nIssues resolved? [Y/n] › ")
				scanner.Scan()
				resolved := strings.TrimSpace(strings.ToLower(scanner.Text()))
				if resolved != "n" {
					wf.Steps.Fix.Status = "done"
				} else {
					wf.Steps.Fix.Status = "pending"
					fmt.Println("\nFix still pending. Run `coder workflow --resume` when ready.")
					saveWorkflowState(wf, wfFile)
					return
				}
			} else {
				wf.Steps.Fix.Status = "skipped"
			}
			wf.Updated = time.Now()
			saveWorkflowState(wf, wfFile)
		}
	}

	// ─────────────────────────────────────────────
	// DONE
	// ─────────────────────────────────────────────
	wf.Status = "done"
	wf.Updated = time.Now()
	saveWorkflowState(wf, wfFile)

	fmt.Println("\n" + strings.Repeat("═", 58))
	fmt.Println("  WORKFLOW COMPLETE")
	fmt.Println(strings.Repeat("═", 58))
	fmt.Printf("\n  Feature : %s\n", truncate(wf.Feature, 48))
	fmt.Printf("  Plan    : %s\n", wf.Steps.Plan.Artifact)
	fmt.Printf("  QA      : %s\n", wf.Steps.QA.Status)
	fmt.Printf("  Fix     : %s\n", wf.Steps.Fix.Status)
	fmt.Printf("  Duration: %s\n", time.Since(wf.Created).Round(time.Second))
	fmt.Println("\n" + strings.Repeat("═", 58))
}

// ─── LLM helpers ────────────────────────────────────────────────────────────

// generatePlan calls the chat endpoint in auto mode and returns the raw plan text.
func generatePlan(feature string, client *httpclient.ChatClient) string {
	prompt := buildPlanPrompt(feature, "", nil, true)
	ctx := context.Background()

	var planContent strings.Builder
	planContent.WriteString(fmt.Sprintf("# Plan: %s\n\nGenerated: %s\n\n", feature, time.Now().Format("2006-01-02")))

	_, err := client.ChatStream(ctx, prompt, "", true, true, func(delta string) {
		fmt.Print(delta)
		planContent.WriteString(delta)
	})
	fmt.Println()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating plan: %v\n", err)
		return ""
	}
	return planContent.String()
}

// reviewPlan sends the plan text to the LLM for self-review.
func reviewPlan(planText string, client *httpclient.ChatClient) {
	prompt := fmt.Sprintf(`Review the following implementation plan. Identify:
1. Completeness gaps — anything missing that would block implementation?
2. Hidden risks — security, performance, or data integrity concerns?
3. Ambiguous tasks — any step that is unclear or under-specified?
4. Suggestions — improvements or alternative approaches worth considering?

Keep the review concise. Format as:

## Completeness
<findings or "None">

## Risks
<findings or "None">

## Ambiguities
<findings or "None">

## Suggestions
<findings or "None">

---

%s`, truncate(planText, 4000))

	ctx := context.Background()
	_, err := client.ChatStream(ctx, prompt, "", true, true, func(delta string) {
		fmt.Print(delta)
	})
	fmt.Println()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Plan review error: %v\n", err)
	}
}

// generateImplementChecklist produces an ordered implementation checklist.
func generateImplementChecklist(feature, planText string, client *httpclient.ChatClient) {
	prompt := fmt.Sprintf(`Based on the following plan for "%s", generate a concrete, ordered implementation checklist.

For each item:
- One clear, actionable task
- File(s) to create or modify
- Estimated time

Format as a numbered list. Be specific.

---

%s`, feature, truncate(planText, 4000))

	ctx := context.Background()
	_, err := client.ChatStream(ctx, prompt, "", true, true, func(delta string) {
		fmt.Print(delta)
	})
	fmt.Println()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Checklist generation error: %v\n", err)
	}
}

// ─── State helpers ────────────────────────────────────────────────────────────

func saveWorkflowState(wf *workflowState, path string) {
	if path == "" {
		return
	}
	os.MkdirAll(filepath.Dir(path), 0755)
	data, err := json.MarshalIndent(wf, "", "  ")
	if err != nil {
		return
	}
	os.WriteFile(path, data, 0644)
}

func findWorkflowFile(id string) string {
	wfDir := filepath.Join(".coder", "workflows")
	entries, err := os.ReadDir(wfDir)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		path := filepath.Join(wfDir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var wf workflowState
		if err := json.Unmarshal(data, &wf); err != nil {
			continue
		}
		if wf.ID == id {
			return path
		}
	}
	return ""
}

func findLastWorkflow() *workflowState {
	wfDir := filepath.Join(".coder", "workflows")
	entries, err := os.ReadDir(wfDir)
	if err != nil {
		return nil
	}
	var latest *workflowState
	var latestTime time.Time
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		info, _ := e.Info()
		if info.ModTime().Before(latestTime) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(wfDir, e.Name()))
		if err != nil {
			continue
		}
		var wf workflowState
		if err := json.Unmarshal(data, &wf); err != nil {
			continue
		}
		if wf.Status != "done" {
			latest = &wf
			latestTime = info.ModTime()
		}
	}
	return latest
}

func listWorkflows() {
	wfDir := filepath.Join(".coder", "workflows")
	entries, err := os.ReadDir(wfDir)
	if err != nil || len(entries) == 0 {
		fmt.Println("No workflows found. Start one with: coder workflow <feature>")
		return
	}
	fmt.Printf("\n  %-12s  %-8s  %-44s  %s\n", "WORKFLOW ID", "STATUS", "FEATURE", "UPDATED")
	fmt.Printf("  %s\n", strings.Repeat("─", 80))
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(wfDir, e.Name()))
		if err != nil {
			continue
		}
		var wf workflowState
		if err := json.Unmarshal(data, &wf); err != nil {
			continue
		}
		fmt.Printf("  %-12s  %-8s  %-44s  %s\n",
			truncate(wf.ID, 12),
			wf.Status,
			truncate(wf.Feature, 44),
			wf.Updated.Format("2006-01-02 15:04"),
		)
	}
	fmt.Println()
}

// ─── Step selection ────────────────────────────────────────────────────────────

// parseWorkflowSteps parses a comma-separated step list.
// Empty string means "all steps".
func parseWorkflowSteps(s string) map[string]bool {
	if s == "" {
		return nil // nil = all
	}
	m := make(map[string]bool)
	for _, part := range strings.Split(s, ",") {
		m[strings.TrimSpace(strings.ToLower(part))] = true
	}
	return m
}

// shouldRun returns true if step is enabled (nil map means all enabled).
func shouldRun(step string, enabled map[string]bool) bool {
	if enabled == nil {
		return true
	}
	return enabled[step]
}
