package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

const discussPhaseUsage = `Usage: coder discuss-phase <N> [flags]

Generate a CONTEXT.md for phase N by identifying gray areas and capturing
decisions via interactive Q&A or auto mode.

EXAMPLES:
  coder discuss-phase 1
  coder discuss-phase 2 --auto    # AI picks defaults, no Q&A
  coder discuss-phase 3 --batch   # ask all questions at once

FLAGS:
`

func runDiscussPhase(args []string) {
	fs := flag.NewFlagSet("discuss-phase", flag.ExitOnError)
	auto := fs.Bool("auto", false, "AI picks recommended defaults, no interactive Q&A")
	batch := fs.Bool("batch", false, "Ask all questions at once instead of one-by-one")

	fs.Usage = func() {
		fmt.Fprint(os.Stderr, discussPhaseUsage)
		fs.PrintDefaults()
	}
	fs.Parse(args)

	logActivity("discuss-phase")

	// Parse phase number
	if len(fs.Args()) == 0 {
		fmt.Fprintln(os.Stderr, "Error: phase number required.\nUsage: coder discuss-phase <N>")
		os.Exit(1)
	}
	phaseNum := 0
	_, err := fmt.Sscanf(fs.Args()[0], "%d", &phaseNum)
	if err != nil || phaseNum < 1 {
		fmt.Fprintf(os.Stderr, "Error: invalid phase number %q\n", fs.Args()[0])
		os.Exit(1)
	}
	phaseTag := fmt.Sprintf("%02d", phaseNum)

	// Check PROJECT.md exists
	if !projectExists() {
		fmt.Fprintln(os.Stderr, "Error: .coder/PROJECT.md not found.\nRun coder new-project first.")
		os.Exit(1)
	}

	// Load ROADMAP.md and find phase name
	phases, err := loadRoadmap()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: could not load .coder/ROADMAP.md: %v\n", err)
		os.Exit(1)
	}
	phaseName := ""
	phaseDesc := ""
	for _, p := range phases {
		if p.Number == phaseNum {
			phaseName = p.Name
			phaseDesc = p.Name
			break
		}
	}
	if phaseName == "" {
		fmt.Fprintf(os.Stderr, "Error: phase %d not found in ROADMAP.md\n", phaseNum)
		os.Exit(1)
	}

	// Check if CONTEXT.md already exists
	contextPath := coderPath("phases", phaseTag+"-CONTEXT.md")
	if _, statErr := os.Stat(contextPath); statErr == nil {
		fmt.Printf("Context exists: %s\n", contextPath)
		fmt.Print("Update? [y/N] ")
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		ans := strings.TrimSpace(strings.ToLower(scanner.Text()))
		if ans != "y" && ans != "yes" {
			fmt.Println("Keeping existing context.")
			return
		}
	}

	// Create phase directory
	if err := ensurePhaseDir(phaseNum); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating phase directory: %v\n", err)
		os.Exit(1)
	}

	// Load prior context
	projectContent := readFileOrEmpty(coderPath("PROJECT.md"))
	requirementsContent := readFileOrEmpty(coderPath("REQUIREMENTS.md"))
	priorContextSummaries := collectPriorContextSummaries(phaseNum)

	cfg, _ := loadConfig()
	if cfg == nil {
		cfg = &Config{}
	}
	chatClient := getChatClient(cfg)
	ctx := context.Background()

	// Banner
	fmt.Printf("\n╔══════════════════════════════════════════════════════╗\n")
	fmt.Printf("║  discuss-phase %d  ·  %-34s║\n", phaseNum, truncate(phaseName, 34))
	fmt.Printf("╚══════════════════════════════════════════════════════╝\n\n")

	var qaLog strings.Builder

	if !*auto {
		// Build Q&A prompt and stream it
		fmt.Printf("Identifying gray areas for this phase...\n\n")
		qaPrompt := buildDiscussQAPrompt(phaseNum, phaseName, phaseDesc, projectContent, priorContextSummaries, requirementsContent)

		var questionsText strings.Builder
		_, err = chatClient.ChatStream(ctx, qaPrompt, "", true, true, func(delta string) {
			fmt.Print(delta)
			questionsText.WriteString(delta)
		})
		fmt.Println()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating questions: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("\n" + strings.Repeat("─", 54))

		if *batch {
			// Batch mode: collect all answers in one prompt
			fmt.Println("Answer all questions above (type 'done' on a new line when finished):")
			fmt.Println(strings.Repeat("─", 54))
			var batchAnswers strings.Builder
			scanner := bufio.NewScanner(os.Stdin)
			for scanner.Scan() {
				line := scanner.Text()
				if strings.TrimSpace(line) == "done" {
					break
				}
				batchAnswers.WriteString(line + "\n")
			}
			if batchAnswers.Len() > 0 {
				qaLog.WriteString("Questions:\n")
				qaLog.WriteString(questionsText.String())
				qaLog.WriteString("\nAnswers:\n")
				qaLog.WriteString(batchAnswers.String())
			}
		} else {
			// One-by-one interactive loop
			fmt.Println("Answer each question (press Enter or type 'done' to finish):")
			fmt.Println(strings.Repeat("─", 54))
			qaLog.WriteString("Questions:\n")
			qaLog.WriteString(questionsText.String())
			qaLog.WriteString("\n\nAnswers:\n")
			scanner := bufio.NewScanner(os.Stdin)
			qNum := 1
			for {
				fmt.Printf("\n[Q%d] Your answer (or 'done' to proceed): ", qNum)
				if !scanner.Scan() {
					break
				}
				ans := strings.TrimSpace(scanner.Text())
				if ans == "done" || ans == "" {
					break
				}
				qaLog.WriteString(fmt.Sprintf("Q%d: %s\n", qNum, ans))
				qNum++
			}
		}
	} else {
		fmt.Println("Auto mode — AI will use recommended defaults for all decisions...")
		qaLog.WriteString("Mode: auto (AI-selected recommended defaults)\n")
		qaLog.WriteString(fmt.Sprintf("Phase: %d — %s\n", phaseNum, phaseName))
	}

	// Generate CONTEXT.md from Q&A results
	fmt.Printf("\nGenerating CONTEXT.md...\n\n")
	fmt.Println(strings.Repeat("═", 58))

	contextPrompt := buildContextGenerationPrompt(phaseNum, phaseName, qaLog.String(), *auto)

	var contextContent strings.Builder
	contextContent.WriteString(fmt.Sprintf("# Phase %d Context — %s\n\n", phaseNum, phaseName))
	contextContent.WriteString(fmt.Sprintf("Generated: %s\n\n", time.Now().Format("2006-01-02")))

	_, err = chatClient.ChatStream(ctx, contextPrompt, "", true, true, func(delta string) {
		fmt.Print(delta)
		contextContent.WriteString(delta)
	})
	fmt.Printf("\n%s\n", strings.Repeat("═", 58))

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating CONTEXT.md: %v\n", err)
		os.Exit(1)
	}

	// Ask to accept
	fmt.Print("\nAccept this context? [Y/n] ")
	scanner2 := bufio.NewScanner(os.Stdin)
	scanner2.Scan()
	acceptAns := strings.TrimSpace(strings.ToLower(scanner2.Text()))
	if acceptAns == "n" || acceptAns == "no" {
		fmt.Println("Context discarded.")
		return
	}

	// Write CONTEXT.md
	if err := os.WriteFile(contextPath, []byte(contextContent.String()), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", contextPath, err)
		os.Exit(1)
	}
	fmt.Printf("\nContext saved: %s\n", contextPath)

	// Update STATE.md
	state, err := loadState()
	if err != nil {
		// STATE.md may not exist yet; create a minimal one
		state = &ProjectState{PRs: make(map[int]string)}
	}
	state.Step = "plan"
	state.LastAction = fmt.Sprintf("discuss-phase %d completed", phaseNum)
	if err := saveState(state); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not update STATE.md: %v\n", err)
	}

	fmt.Printf("\nNext: coder plan-phase %d\n", phaseNum)
}

// collectPriorContextSummaries reads CONTEXT.md files from phases before phaseNum.
func collectPriorContextSummaries(currentPhase int) string {
	var sb strings.Builder
	for i := 1; i < currentPhase; i++ {
		tag := fmt.Sprintf("%02d", i)
		path := coderPath("phases", tag+"-CONTEXT.md")
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		sb.WriteString(fmt.Sprintf("--- Phase %d context (summary) ---\n", i))
		sb.WriteString(truncate(string(data), 500))
		sb.WriteString("\n\n")
	}
	return sb.String()
}

// readFileOrEmpty reads a file and returns its contents, or empty string on error.
func readFileOrEmpty(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

func buildDiscussQAPrompt(phaseNum int, phaseName, phaseDesc, projectContent, priorContext, requirementsContent string) string {
	priorSection := ""
	if priorContext != "" {
		priorSection = fmt.Sprintf("\nPrior decisions (from previous phases):\n%s\n", truncate(priorContext, 1500))
	}
	reqSection := ""
	if requirementsContent != "" {
		reqSection = fmt.Sprintf("\nRequirements context:\n%s\n", truncate(requirementsContent, 1000))
	}
	return fmt.Sprintf(`I am planning phase %d (%s) of this project.

Project context:
%s

Phase goal (from ROADMAP.md):
%s
%s%s
Identify 3-5 gray areas that need decisions before I can plan this phase.
For each gray area:
1. State the decision topic clearly
2. Provide 2-4 concrete numbered options (mark the recommended one with *)
3. Keep it specific to THIS phase — not generic advice

Focus on what's being built:
- If it's an API/CLI → response format, auth, error handling, flags
- If it's UI → layout, interactions, empty states, loading states
- If it's data → schema design, access patterns, consistency

Do NOT ask about: technical implementation details, architecture choices, scope expansion.
Output ONLY the questions, not the CONTEXT.md yet.`,
		phaseNum,
		phaseName,
		truncate(projectContent, 2000),
		phaseDesc,
		priorSection,
		reqSection,
	)
}

func buildContextGenerationPrompt(phaseNum int, phaseName, qaContent string, autoMode bool) string {
	autoNote := ""
	if autoMode {
		autoNote = "\n(Auto mode was used — select the recommended option for each decision area.)\n"
	}
	return fmt.Sprintf(`Based on this Q&A session for phase %d (%s):%s

Questions and answers:
%s

Generate a CONTEXT.md with these sections:

# Phase %d Context — %s

For each decision discussed, create a section:
## {Decision Topic}
Decision: {what was decided}
Rationale: {why}
Impact: {how this affects implementation}

## Deferred Ideas
- {any scope creep items that were deferred}

Keep decisions specific and actionable. A reader of this file should know exactly what to build without asking further questions.`,
		phaseNum, phaseName, autoNote,
		qaContent,
		phaseNum, phaseName,
	)
}
