package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	httpclient "github.com/trungtran/coder/internal/transport/http/client"
)

const planUsage = `Usage: coder plan [feature-description] [flags]

Interactive planning workflow. Asks clarifying questions, researches patterns,
then generates a structured PLAN.md with tasks, time estimates, and risks.

After the plan is generated you can:
  Y / Enter   accept and save the plan
  n           discard the plan
  refine      enter Q&A refinement loop (ask questions, request changes)
              type 'done' or empty line to exit refinement and re-confirm

EXAMPLES:
  coder plan "implement JWT authentication"
  coder plan --auto "add Redis caching for search"   # skip Q&A
  coder plan --prd path/to/feature.md               # from PRD document
  coder plan --list                                  # list existing plans
  coder plan --file src/auth.go "refactor this"      # plan for specific file

FLAGS:
`

func runPlan(args []string) {
	fs := flag.NewFlagSet("plan", flag.ExitOnError)
	auto := fs.Bool("auto", false, "Skip Q&A, auto-generate plan with defaults")
	prdFile := fs.String("prd", "", "Read feature description from a PRD markdown file")
	contextFile := fs.String("file", "", "Source file to include as context")
	output := fs.String("o", "", "Output path for PLAN.md (default: .coder/plans/PLAN-<slug>.md)")
	list := fs.Bool("list", false, "List existing plans")

	fs.Usage = func() {
		fmt.Fprint(os.Stderr, planUsage)
		fs.PrintDefaults()
	}
	fs.Parse(args)

	logActivity("plan")

	cfg, _ := loadConfig()
	if cfg == nil {
		cfg = &Config{}
	}

	// List plans
	if *list {
		listPlans()
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
		fmt.Fprint(os.Stderr, "Usage: coder plan <feature-description> [flags]\nRun `coder plan --help` for details.\n")
		os.Exit(1)
	}

	// Optional file context
	var fileCtx string
	if *contextFile != "" {
		data, _ := os.ReadFile(*contextFile)
		fileCtx = string(data)
	}

	baseURL := cfg.Memory.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	chatClient := httpclient.NewChatClient(baseURL, cfg.Auth.AccessToken)
	ctx := context.Background()

	// Banner
	title := truncate(feature, 48)
	fmt.Printf("\n╔══════════════════════════════════════════════════════╗\n")
	fmt.Printf("║  coder plan  ·  %-38s║\n", title)
	fmt.Printf("╚══════════════════════════════════════════════════════╝\n\n")

	var decisions []string
	sessionID := ""

	// Single scanner for all interactive stdin reads in this command
	scanner := bufio.NewScanner(os.Stdin)

	if !*auto {
		// Stage 1: Q&A — identify gray areas
		fmt.Printf("Analysing feature scope...\n\n")

		qaPrompt := buildQAPrompt(feature, fileCtx)
		result, err := chatClient.ChatStream(ctx, qaPrompt, sessionID, true, true, func(delta string) {
			fmt.Print(delta)
		})
		fmt.Println()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if result != nil {
			sessionID = result.SessionID
		}

		// Collect answers interactively
		fmt.Println("\n" + strings.Repeat("─", 54))
		fmt.Println("Answer each question (or press Enter to use default):")
		fmt.Println(strings.Repeat("─", 54))

		qNum := 1
		for {
			fmt.Printf("\n[%d] Your answer (or 'done' to proceed): ", qNum)
			if !scanner.Scan() {
				break
			}
			ans := strings.TrimSpace(scanner.Text())
			if ans == "done" || ans == "" {
				break
			}
			decisions = append(decisions, fmt.Sprintf("Decision %d: %s", qNum, ans))
			qNum++
		}

		fmt.Println("\nDecisions captured. Researching implementation patterns...")
	} else {
		fmt.Println("Auto mode — skipping Q&A, using recommended defaults...")
	}

	// Stage 2+3: Research + Plan generation
	planPrompt := buildPlanPrompt(feature, fileCtx, decisions, *auto)

	fmt.Printf("\nGenerating plan...\n\n")
	fmt.Println(strings.Repeat("═", 58))
	fmt.Printf("  PLAN: %s\n", truncate(feature, 48))
	fmt.Printf("  Generated: %s\n", time.Now().Format("2006-01-02"))
	fmt.Println(strings.Repeat("═", 58) + "\n")

	var planContent strings.Builder
	planContent.WriteString(fmt.Sprintf("# Plan: %s\n\n", feature))
	planContent.WriteString(fmt.Sprintf("Generated: %s\n\n", time.Now().Format("2006-01-02")))

	if len(decisions) > 0 {
		planContent.WriteString("## Context (decisions from Q&A)\n")
		for _, d := range decisions {
			planContent.WriteString("- " + d + "\n")
		}
		planContent.WriteString("\n")
	}

	result, err := chatClient.ChatStream(ctx, planPrompt, sessionID, true, true, func(delta string) {
		fmt.Print(delta)
		planContent.WriteString(delta)
	})
	fmt.Printf("\n\n%s\n", strings.Repeat("═", 58))

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating plan: %v\n", err)
		os.Exit(1)
	}
	if result != nil {
		sessionID = result.SessionID
	}

	// --- Refinement Q&A loop ---
	// Allows user to ask questions or request changes before accepting the plan.
	for {
		fmt.Print("\nAccept plan? [Y/n/refine] › ")
		scanner.Scan()
		answer := strings.TrimSpace(strings.ToLower(scanner.Text()))

		switch answer {
		case "n":
			fmt.Println("Plan discarded.")
			return

		case "refine", "edit", "r", "e":
			// Enter refinement loop using same session
			fmt.Printf("\n%s\n", strings.Repeat("─", 58))
			fmt.Println("  Refinement mode — ask questions or request changes.")
			fmt.Println("  Type your question, or 'done' / empty line to finish.")
			fmt.Printf("%s\n\n", strings.Repeat("─", 58))

			for {
				fmt.Print(bold("You") + " › ")
				if !scanner.Scan() {
					break
				}
				userInput := strings.TrimSpace(scanner.Text())
				if userInput == "" || userInput == "done" || userInput == "ok" {
					fmt.Println("\nExiting refinement. Showing updated plan above — please review and accept.")
					break
				}

				fmt.Print("\n" + bold("Assistant") + " › ")
				var updatedSection strings.Builder
				refResult, refErr := chatClient.ChatStream(ctx, userInput, sessionID, true, true, func(delta string) {
					fmt.Print(delta)
					updatedSection.WriteString(delta)
				})
				fmt.Print("\n\n")

				if refErr != nil {
					fmt.Fprintf(os.Stderr, "  Error: %v\n\n", refErr)
					continue
				}
				if refResult != nil {
					sessionID = refResult.SessionID
				}
				// Append refinement exchange to plan content so saved plan reflects it
				if updatedSection.Len() > 0 {
					planContent.WriteString(fmt.Sprintf("\n\n---\n_Refinement note:_ %s\n\n%s\n", userInput, updatedSection.String()))
				}
			}

		default:
			// Y / enter → accept
			// Save plan
			outPath := *output
			if outPath == "" {
				slug := toSlug(feature)
				outPath = filepath.Join(".coder", "plans", fmt.Sprintf("PLAN-%s.md", slug))
			}

			os.MkdirAll(filepath.Dir(outPath), 0755)
			if err := os.WriteFile(outPath, []byte(planContent.String()), 0644); err != nil {
				fmt.Fprintf(os.Stderr, "Error saving plan: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("\nPlan saved: %s\n", outPath)
			fmt.Printf("\nNext steps:\n")
			fmt.Printf("  Execute:  coder execute-plan --plan %s\n", outPath)
			fmt.Printf("  Verify:   coder qa --plan %s\n", outPath)
			fmt.Printf("  Discuss:  coder chat --session %s\n", sessionID)
			return
		}
	}
}

func buildQAPrompt(feature, fileCtx string) string {
	extra := ""
	if fileCtx != "" {
		extra = fmt.Sprintf("\n\n## File context:\n```\n%s\n```", truncate(fileCtx, 3000))
	}
	return fmt.Sprintf(`I want to implement the following feature:

"%s"%s

Please identify 3-5 key ambiguous decisions or gray areas that need to be clarified before we can create a solid implementation plan.

For each area:
1. State the decision topic clearly
2. Provide 2-4 concrete numbered options (with a recommended option marked)
3. Keep each question focused and answerable

Format as a numbered list. Do not generate the plan yet — only the clarifying questions.`, feature, extra)
}

func buildPlanPrompt(feature, fileCtx string, decisions []string, autoMode bool) string {
	decisionSection := ""
	if len(decisions) > 0 {
		decisionSection = "\n## Decisions made:\n"
		for _, d := range decisions {
			decisionSection += "- " + d + "\n"
		}
	}

	fileSection := ""
	if fileCtx != "" {
		fileSection = fmt.Sprintf("\n## File context:\n```\n%s\n```\n", truncate(fileCtx, 3000))
	}

	autoNote := ""
	if autoMode {
		autoNote = "\nUse the recommended/default option for any unclear decisions."
	}

	return fmt.Sprintf(`Generate a detailed, actionable implementation plan for:

"%s"
%s%s%s
The plan should include:

## Overview
Brief description of what will be built and key decisions.

## Tasks
Each task with:
- Clear title and time estimate (e.g. 30 min, 1h, 2h)
- Specific steps to complete
- Files to create or modify

## Files
List of files to create/modify with brief descriptions.

## Risks
Known risks with severity (LOW/MEDIUM/HIGH) and mitigation.

## Estimated total
Total hours.

Format as clean markdown. Be specific and actionable.`, feature, decisionSection, fileSection, autoNote)
}

func listPlans() {
	plansDir := filepath.Join(".coder", "plans")
	entries, err := os.ReadDir(plansDir)
	if err != nil {
		fmt.Println("No plans found. Create one with: coder plan <feature>")
		return
	}
	if len(entries) == 0 {
		fmt.Println("No plans found.")
		return
	}
	fmt.Printf("\n  %-50s  %s\n", "PLAN FILE", "MODIFIED")
	fmt.Printf("  %s\n", strings.Repeat("─", 70))
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		info, _ := e.Info()
		fmt.Printf("  %-50s  %s\n", e.Name(), info.ModTime().Format("2006-01-02 15:04"))
	}
	fmt.Println()
}

func toSlug(s string) string {
	s = strings.ToLower(s)
	var out strings.Builder
	for _, r := range s {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			out.WriteRune(r)
		} else if out.Len() > 0 {
			last := []rune(out.String())
			if last[len(last)-1] != '-' {
				out.WriteRune('-')
			}
		}
	}
	result := strings.Trim(out.String(), "-")
	if len(result) > 40 {
		result = result[:40]
	}
	return result
}

