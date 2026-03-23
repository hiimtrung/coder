package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

const planPhaseUsage = `Usage: coder plan-phase <N> [flags]

Generate structured XML implementation plans for phase N, with optional
research and verification loop.

EXAMPLES:
  coder plan-phase 1
  coder plan-phase 2 --skip-research
  coder plan-phase 3 --skip-verify
  coder plan-phase 4 --gaps
  coder plan-phase 5 --prd path/to/feature.md

FLAGS:
`

func runPlanPhase(args []string) {
	fs := flag.NewFlagSet("plan-phase", flag.ExitOnError)
	skipResearch := fs.Bool("skip-research", false, "Skip research if RESEARCH.md already exists")
	skipVerify := fs.Bool("skip-verify", false, "Skip verification loop")
	gaps := fs.Bool("gaps", false, "Re-plan only items flagged in VERIFICATION.md")
	prdFile := fs.String("prd", "", "Use PRD file instead of CONTEXT.md")

	fs.Usage = func() {
		fmt.Fprint(os.Stderr, planPhaseUsage)
		fs.PrintDefaults()
	}
	fs.Parse(args)

	logActivity("plan-phase")

	// Parse phase number
	if len(fs.Args()) == 0 {
		fmt.Fprintln(os.Stderr, "Error: phase number required.\nUsage: coder plan-phase <N>")
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
	for _, p := range phases {
		if p.Number == phaseNum {
			phaseName = p.Name
			break
		}
	}
	if phaseName == "" {
		fmt.Fprintf(os.Stderr, "Error: phase %d not found in ROADMAP.md\n", phaseNum)
		os.Exit(1)
	}

	// Determine context source
	contextPath := coderPath("phases", phaseTag+"-CONTEXT.md")
	if *prdFile != "" {
		// Generate CONTEXT.md from PRD
		prdData, err := os.ReadFile(*prdFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading PRD file: %v\n", err)
			os.Exit(1)
		}
		// Write PRD content as CONTEXT.md stub so downstream steps can use it
		prdContextContent := fmt.Sprintf("# Phase %d Context — %s\n\nGenerated: %s\n\n## Source\nGenerated from PRD: %s\n\n## PRD Content\n%s\n",
			phaseNum, phaseName, time.Now().Format("2006-01-02"), *prdFile, string(prdData))
		if err := ensurePhaseDir(phaseNum); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating phase directory: %v\n", err)
			os.Exit(1)
		}
		if err := os.WriteFile(contextPath, []byte(prdContextContent), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing CONTEXT.md from PRD: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Context generated from PRD: %s\n", contextPath)
	} else {
		// Require CONTEXT.md to exist
		if _, statErr := os.Stat(contextPath); statErr != nil {
			fmt.Fprintf(os.Stderr, "Error: %s not found.\nRun coder discuss-phase %d first, or use --prd <file>.\n", contextPath, phaseNum)
			os.Exit(1)
		}
	}

	// Create phase directory
	if err := ensurePhaseDir(phaseNum); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating phase directory: %v\n", err)
		os.Exit(1)
	}

	// Load context files
	projectContent := readFileOrEmpty(coderPath("PROJECT.md"))
	requirementsContent := readFileOrEmpty(coderPath("REQUIREMENTS.md"))
	contextContent := readFileOrEmpty(contextPath)

	cfg, _ := loadConfig()
	if cfg == nil {
		cfg = &Config{}
	}
	chatClient := getChatClient(cfg)
	ctx := context.Background()

	// Banner
	fmt.Printf("\n╔══════════════════════════════════════════════════════╗\n")
	fmt.Printf("║  plan-phase %d  ·  %-36s║\n", phaseNum, truncate(phaseName, 36))
	fmt.Printf("╚══════════════════════════════════════════════════════╝\n\n")

	// --- Step A: Research ---
	researchPath := coderPath("phases", phaseTag+"-RESEARCH.md")
	researchContent := ""

	skipResearchStep := *skipResearch && fileExists(researchPath)
	if skipResearchStep {
		fmt.Printf("Skipping research (RESEARCH.md exists): %s\n", researchPath)
		researchContent = readFileOrEmpty(researchPath)
	} else {
		fmt.Printf("Researching phase %d...\n\n", phaseNum)
		researchPrompt := buildPhaseResearchPrompt(phaseNum, phaseName, projectContent, contextContent)

		var researchBuf strings.Builder
		researchBuf.WriteString(fmt.Sprintf("# Phase %d Research — %s\n\nGenerated: %s\n\n", phaseNum, phaseName, time.Now().Format("2006-01-02")))

		_, err = chatClient.ChatStream(ctx, researchPrompt, "", true, true, func(delta string) {
			fmt.Print(delta)
			researchBuf.WriteString(delta)
		})
		fmt.Println()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error during research: %v\n", err)
			os.Exit(1)
		}

		researchContent = researchBuf.String()
		if err := os.WriteFile(researchPath, []byte(researchContent), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing research file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("\nResearch complete → %s\n", researchPath)
	}

	// --- Step B: Plan generation ---

	// Handle --gaps mode: load existing plans and only re-plan flagged items
	verificationContent := ""
	if *gaps {
		verificationPath := coderPath("phases", phaseTag+"-VERIFICATION.md")
		verificationContent = readFileOrEmpty(verificationPath)
		if verificationContent == "" {
			fmt.Fprintf(os.Stderr, "Warning: --gaps specified but no VERIFICATION.md found at %s\n", verificationPath)
		}
	}

	fmt.Printf("\nGenerating plans...\n\n")

	planningPrompt := buildPhasePlanningPrompt(phaseNum, phaseName, projectContent, requirementsContent, contextContent, researchContent, verificationContent, *gaps)

	var plansBuf strings.Builder
	_, err = chatClient.ChatStream(ctx, planningPrompt, "", true, true, func(delta string) {
		fmt.Print(delta)
		plansBuf.WriteString(delta)
	})
	fmt.Println()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating plans: %v\n", err)
		os.Exit(1)
	}

	// Parse and write individual plan files
	xmlBlocks := extractXMLBlocks(plansBuf.String())
	var planFiles []string

	if len(xmlBlocks) == 0 {
		// Fall back: write the whole response as a single plan
		fallbackPath := coderPath("phases", fmt.Sprintf("%s-01-PLAN.md", phaseTag))
		content := fmt.Sprintf("# Phase %d Plan\n\nGenerated: %s\n\n%s\n", phaseNum, time.Now().Format("2006-01-02"), plansBuf.String())
		if err := os.WriteFile(fallbackPath, []byte(content), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing plan file: %v\n", err)
			os.Exit(1)
		}
		planFiles = append(planFiles, fallbackPath)
	} else {
		for i, block := range xmlBlocks {
			planFileName := fmt.Sprintf("%s-%02d-PLAN.md", phaseTag, i+1)
			planPath := coderPath("phases", planFileName)
			content := fmt.Sprintf("# Phase %d Plan %d\n\nGenerated: %s\n\n```xml\n%s\n```\n", phaseNum, i+1, time.Now().Format("2006-01-02"), block)
			if err := os.WriteFile(planPath, []byte(content), 0644); err != nil {
				fmt.Fprintf(os.Stderr, "Error writing plan file %s: %v\n", planPath, err)
				os.Exit(1)
			}
			planFiles = append(planFiles, planPath)
		}
	}

	fmt.Printf("\n%d plan(s) generated\n", len(planFiles))

	// --- Step C: Verification loop ---
	if !*skipVerify {
		fmt.Println("\nVerifying plans...")

		allPlanXML := collectPlanXML(planFiles)
		passed := false
		maxIter := 3

		for iter := 1; iter <= maxIter; iter++ {
			checkerPrompt := buildPhaseCheckerPrompt(phaseNum, requirementsContent, allPlanXML)

			var checkBuf strings.Builder
			_, err = chatClient.ChatStream(ctx, checkerPrompt, "", true, true, func(delta string) {
				fmt.Print(delta)
				checkBuf.WriteString(delta)
			})
			fmt.Println()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: verification call failed (iter %d): %v\n", iter, err)
				break
			}

			checkResult := checkBuf.String()

			// Write verification result
			verPath := coderPath("phases", phaseTag+"-VERIFICATION.md")
			verContent := fmt.Sprintf("# Phase %d Verification — Iteration %d\n\nGenerated: %s\n\n%s\n", phaseNum, iter, time.Now().Format("2006-01-02"), checkResult)
			os.WriteFile(verPath, []byte(verContent), 0644) //nolint:errcheck

			if strings.Contains(checkResult, "PASS") {
				passed = true
				break
			}

			// FAIL — stream a fix prompt
			if iter < maxIter {
				fmt.Printf("\nIssues found (iteration %d/%d). Generating fixes...\n\n", iter, maxIter)
				fixPrompt := buildPhaseFixPrompt(phaseNum, phaseName, checkResult, allPlanXML)

				var fixBuf strings.Builder
				_, err = chatClient.ChatStream(ctx, fixPrompt, "", true, true, func(delta string) {
					fmt.Print(delta)
					fixBuf.WriteString(delta)
				})
				fmt.Println()
				if err != nil {
					fmt.Fprintf(os.Stderr, "Warning: fix call failed (iter %d): %v\n", iter, err)
					break
				}

				// Re-parse and update plan files with fixed XML
				fixedBlocks := extractXMLBlocks(fixBuf.String())
				if len(fixedBlocks) > 0 {
					for i, block := range fixedBlocks {
						if i >= len(planFiles) {
							break
						}
						content := fmt.Sprintf("# Phase %d Plan %d (revised)\n\nGenerated: %s\n\n```xml\n%s\n```\n", phaseNum, i+1, time.Now().Format("2006-01-02"), block)
						os.WriteFile(planFiles[i], []byte(content), 0644) //nolint:errcheck
					}
					allPlanXML = collectPlanXML(planFiles)
				}
			}
		}

		if passed {
			fmt.Println("\nPlans verified")
		} else {
			fmt.Println("\nVerification incomplete after 3 iterations — review manually")
		}
	}

	// Interactive confirm before finalising
	fmt.Print("\nAccept these plans? [Y/n] ")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	ans := strings.TrimSpace(strings.ToLower(scanner.Text()))
	if ans == "n" || ans == "no" {
		fmt.Println("Plans discarded.")
		return
	}

	// Update STATE.md
	state, err := loadState()
	if err != nil {
		state = &ProjectState{PRs: make(map[int]string)}
	}
	state.Step = "execute"
	state.LastAction = fmt.Sprintf("plan-phase %d completed", phaseNum)
	if err := saveState(state); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not update STATE.md: %v\n", err)
	}

	// Final summary
	fmt.Printf("\nPlans ready:\n")
	for _, f := range planFiles {
		fmt.Printf("  %s\n", f)
	}
	fmt.Printf("\nNext: coder execute-phase %d\n", phaseNum)
}

// fileExists returns true if the path exists and is a regular file.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// collectPlanXML reads plan files and concatenates their contents.
func collectPlanXML(planFiles []string) string {
	var sb strings.Builder
	for _, f := range planFiles {
		content := readFileOrEmpty(f)
		if content != "" {
			sb.WriteString(content)
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

// extractXMLBlocks finds ```xml ... ``` fenced blocks containing <plan> elements.
func extractXMLBlocks(s string) []string {
	var blocks []string
	re := regexp.MustCompile("(?s)```xml\\s*\\n(<plan[^`]+)</plan>\\s*\\n```")
	for _, m := range re.FindAllStringSubmatch(s, -1) {
		blocks = append(blocks, m[1]+"</plan>")
	}
	return blocks
}

func buildPhaseResearchPrompt(phaseNum int, phaseName, projectContent, contextContent string) string {
	return fmt.Sprintf(`I need to implement phase %d (%s) of this project.

Project: %s
Phase goal: %s
Decisions made: %s

Research what I need to know BEFORE planning. Cover:

## Recommended Approach
{implementation strategy}

## Library Choices
| Library | Purpose | Notes |

## Key Patterns (from existing code to follow)
{check existing codebase structure}

## Pitfalls to Avoid
- {known issues}

Be specific. Focus on actionable findings for this exact phase.`,
		phaseNum, phaseName,
		truncate(projectContent, 1500),
		phaseName,
		truncate(contextContent, 2000),
	)
}

func buildPhasePlanningPrompt(phaseNum int, phaseName, projectContent, requirementsContent, contextContent, researchContent, verificationContent string, gapsMode bool) string {
	gapsSection := ""
	if gapsMode && verificationContent != "" {
		gapsSection = fmt.Sprintf("\nRe-plan ONLY items flagged in this verification report:\n%s\n", truncate(verificationContent, 1500))
	}

	return fmt.Sprintf(`Generate implementation plans for phase %d (%s).

Project: %s
Requirements for this phase: %s
Decisions: %s
Research findings: %s
%s
Create 2-4 atomic plans. Each plan = one cohesive concern that one developer can implement in 1-2 hours.

For EACH plan, output EXACTLY this XML format (inside `+"```xml"+` fences):

`+"```xml"+`
<plan id="%d-{01}" phase="%d" name="{concern name}">
  <objective>{what this plan delivers in one sentence}</objective>
  <files>
{file1}
{file2}
  </files>
  <dependencies>none</dependencies>
  <estimated_time>{30m | 1h | 2h}</estimated_time>
  <tasks>
    <task type="create|modify">
      <name>{short task name}</name>
      <action>
{specific instructions — library choices, patterns, what to implement}
      </action>
      <verify>{runnable command or observable outcome}</verify>
      <done>{one-line acceptance criterion}</done>
    </task>
  </tasks>
</plan>
`+"```"+`

Plans should collectively cover ALL requirements for this phase.`,
		phaseNum, phaseName,
		truncate(projectContent, 1000),
		truncate(requirementsContent, 1500),
		truncate(contextContent, 2000),
		truncate(researchContent, 2000),
		gapsSection,
		phaseNum, phaseNum,
	)
}

func buildPhaseCheckerPrompt(phaseNum int, requirementsContent, allPlanXML string) string {
	return fmt.Sprintf(`Review these implementation plans for phase %d.

Requirements to cover:
%s

Plans:
%s

Check:
1. Does every requirement have at least one task covering it?
2. Is every <action> specific enough to execute?
3. Are there file conflicts between plans?
4. Do dependency declarations match task relationships?

Output PASS if all good, or FAIL with specific issues listed.`,
		phaseNum,
		truncate(requirementsContent, 2000),
		truncate(allPlanXML, 4000),
	)
}

func buildPhaseFixPrompt(phaseNum int, phaseName, issues, allPlanXML string) string {
	return fmt.Sprintf(`Fix the following issues found in implementation plans for phase %d (%s).

Issues identified:
%s

Original plans:
%s

Output the corrected plans using the same XML format (inside `+"```xml"+` fences).
Only include plans that needed fixing — keep unchanged plans as-is.`,
		phaseNum, phaseName,
		truncate(issues, 2000),
		truncate(allPlanXML, 3000),
	)
}
