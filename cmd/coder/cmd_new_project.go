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

const newProjectUsage = `Usage: coder new-project "idea description" [flags]

Initialize a new project with AI-guided requirements and roadmap generation.
Runs deep Q&A to clarify scope, then generates REQUIREMENTS.md and ROADMAP.md.

EXAMPLES:
  coder new-project "build a CLI task manager in Go"
  coder new-project --auto "REST API for a blog with auth"
  coder new-project --prd path/to/feature.md
  coder new-project --resume

FLAGS:
`

func runNewProject(args []string) {
	fs := flag.NewFlagSet("new-project", flag.ExitOnError)
	auto := fs.Bool("auto", false, "Skip Q&A, extract requirements from provided text/PRD")
	prdFile := fs.String("prd", "", "Read idea from PRD file")
	resume := fs.Bool("resume", false, "Continue interrupted initialization")

	fs.Usage = func() {
		fmt.Fprint(os.Stderr, newProjectUsage)
		fs.PrintDefaults()
	}
	fs.Parse(args)

	logActivity("new-project")

	// Check for existing project (skip if --resume)
	if !*resume && projectExists() {
		fmt.Fprintln(os.Stderr, "Error: project already initialized. Use: coder progress")
		os.Exit(1)
	}

	cfg, _ := loadConfig()
	if cfg == nil {
		cfg = &Config{}
	}

	chatClient := getChatClient(cfg)
	ctx := context.Background()

	// Gather idea
	var idea string
	if *prdFile != "" {
		data, err := os.ReadFile(*prdFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading PRD file: %v\n", err)
			os.Exit(1)
		}
		idea = string(data)
	} else if len(fs.Args()) > 0 {
		idea = strings.Join(fs.Args(), " ")
	} else if *resume {
		// On resume, try to read from existing PROJECT.md
		data, err := os.ReadFile(coderPath("PROJECT.md"))
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error: --resume requires an existing .coder/PROJECT.md")
			os.Exit(1)
		}
		// Extract project name from first line
		lines := strings.SplitN(string(data), "\n", 3)
		if len(lines) > 0 {
			idea = strings.TrimPrefix(lines[0], "# Project: ")
		}
		if idea == "" {
			idea = "project"
		}
	} else {
		fmt.Fprint(os.Stderr, "Usage: coder new-project \"idea description\" [flags]\nRun `coder new-project --help` for details.\n")
		os.Exit(1)
	}

	// Banner
	title := truncate(idea, 44)
	fmt.Printf("\n╔══════════════════════════════════════════════════════╗\n")
	fmt.Printf("║  coder new-project  ·  %-31s║\n", title)
	fmt.Printf("╚══════════════════════════════════════════════════════╝\n\n")

	var collectedAnswers []string
	sessionID := ""

	// Step 6: Interactive Q&A (unless --auto or --prd)
	if !*auto && *prdFile == "" && !*resume {
		fmt.Printf("Starting deep questioning to understand your project...\n\n")

		qaPrompt := fmt.Sprintf(`I want to build: %s

Please ask me 5-7 focused questions to understand:
- Goals and success criteria
- Tech stack preferences and constraints
- v1 scope (must-have) vs v2 (nice-to-have)
- Specific edge cases or requirements
- Performance / scale expectations

Ask questions one section at a time. Be specific, offer concrete options where possible.`, idea)

		result, err := chatClient.ChatStream(ctx, qaPrompt, sessionID, true, true, func(delta string) {
			fmt.Print(delta)
		})
		fmt.Println()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error during Q&A: %v\n", err)
			os.Exit(1)
		}
		if result != nil {
			sessionID = result.SessionID
		}

		// Collect answers
		fmt.Println("\n" + strings.Repeat("─", 54))
		fmt.Println("Answer the questions above (type 'done' or leave blank to finish):")
		fmt.Println(strings.Repeat("─", 54))

		scanner := bufio.NewScanner(os.Stdin)
		emptyCount := 0
		qNum := 1
		for {
			fmt.Printf("\n[%d] Your answer: ", qNum)
			if !scanner.Scan() {
				break
			}
			ans := strings.TrimSpace(scanner.Text())
			if ans == "done" {
				break
			}
			if ans == "" {
				emptyCount++
				if emptyCount >= 2 {
					break
				}
				continue
			}
			emptyCount = 0
			collectedAnswers = append(collectedAnswers, fmt.Sprintf("Q%d: %s", qNum, ans))
			qNum++
		}

		fmt.Println("\nAnswers captured. Generating requirements...")
	} else if *auto {
		fmt.Println("Auto mode — extracting requirements from description...")
	} else if *resume {
		fmt.Println("Resuming initialization...")
	}

	// Step 7: Generate REQUIREMENTS.md
	discussionText := strings.Join(collectedAnswers, "\n")
	reqPrompt := fmt.Sprintf(`Based on this project idea and our discussion:

Project: %s
Discussion:
%s

Generate a REQUIREMENTS.md with:
## v1 Requirements (must-have for launch)
- numbered list
## v2 Requirements (nice-to-have, future)
- numbered list
## Out of Scope
- what we explicitly will NOT build

Each requirement should be specific and testable.`, idea, discussionText)

	fmt.Printf("\nGenerating REQUIREMENTS.md...\n\n")
	fmt.Println(strings.Repeat("═", 58))

	var reqContent strings.Builder
	reqContent.WriteString(fmt.Sprintf("# Requirements: %s\n\nGenerated: %s\n\n", idea, time.Now().Format("2006-01-02")))

	result, err := chatClient.ChatStream(ctx, reqPrompt, sessionID, true, true, func(delta string) {
		fmt.Print(delta)
		reqContent.WriteString(delta)
	})
	fmt.Printf("\n%s\n", strings.Repeat("═", 58))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating requirements: %v\n", err)
		os.Exit(1)
	}
	if result != nil {
		sessionID = result.SessionID
	}

	// Step 8: Approve requirements
	fmt.Print("\nApprove requirements? [Y/n/edit] > ")
	approveScanner := bufio.NewScanner(os.Stdin)
	approveScanner.Scan()
	reqAnswer := strings.TrimSpace(strings.ToLower(approveScanner.Text()))
	if reqAnswer == "n" {
		fmt.Println("Requirements discarded. Run `coder new-project` again to retry.")
		return
	}

	// Step 9: Generate ROADMAP.md
	roadmapPrompt := fmt.Sprintf(`Based on these requirements, create a ROADMAP.md with:

## Roadmap
### Phase 1 — [Name] (~Xd)
Delivers: [2-3 requirements from v1]

### Phase 2 — [Name] (~Xd)
...

Keep phases small (1-3 days each). Each phase should deliver working, testable functionality.
Maximum 6 phases for v1.

Requirements:
%s`, truncate(reqContent.String(), 3000))

	fmt.Printf("\nGenerating ROADMAP.md...\n\n")
	fmt.Println(strings.Repeat("═", 58))

	var roadmapContent strings.Builder
	roadmapContent.WriteString(fmt.Sprintf("# Roadmap: %s\n\nGenerated: %s\n\n", idea, time.Now().Format("2006-01-02")))

	result, err = chatClient.ChatStream(ctx, roadmapPrompt, sessionID, true, true, func(delta string) {
		fmt.Print(delta)
		roadmapContent.WriteString(delta)
	})
	fmt.Printf("\n%s\n", strings.Repeat("═", 58))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating roadmap: %v\n", err)
		os.Exit(1)
	}
	if result != nil {
		sessionID = result.SessionID
	}

	// Step 10: Approve roadmap
	fmt.Print("\nApprove roadmap? [Y/n/edit] > ")
	roadmapScanner := bufio.NewScanner(os.Stdin)
	roadmapScanner.Scan()
	roadmapAnswer := strings.TrimSpace(strings.ToLower(roadmapScanner.Text()))
	if roadmapAnswer == "n" {
		fmt.Println("Roadmap discarded. Run `coder new-project` again to retry.")
		return
	}

	// Step 11: Write all documents
	os.MkdirAll(coderDir, 0755)

	// Derive project title from idea (first sentence or truncated)
	projectTitle := idea
	if idx := strings.IndexAny(idea, ".!?\n"); idx > 0 {
		projectTitle = strings.TrimSpace(idea[:idx])
	}
	if len(projectTitle) > 60 {
		projectTitle = projectTitle[:60]
	}

	// Build decisions from Q&A answers
	decisionsSection := ""
	for _, ans := range collectedAnswers {
		decisionsSection += fmt.Sprintf("- %s\n", ans)
	}

	// PROJECT.md
	projectMD := fmt.Sprintf(`# Project: %s

## Vision
%s

## Tech Stack
(to be defined during planning)

## Constraints
(to be defined during planning)

## Key Decisions
%s
## Non-Goals
(see REQUIREMENTS.md — Out of Scope)

Initialized: %s
`, projectTitle, truncate(idea, 200), decisionsSection, time.Now().Format("2006-01-02"))

	if err := os.WriteFile(coderPath("PROJECT.md"), []byte(projectMD), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing PROJECT.md: %v\n", err)
		os.Exit(1)
	}

	// REQUIREMENTS.md
	if err := os.WriteFile(coderPath("REQUIREMENTS.md"), []byte(reqContent.String()), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing REQUIREMENTS.md: %v\n", err)
		os.Exit(1)
	}

	// ROADMAP.md
	if err := os.WriteFile(coderPath("ROADMAP.md"), []byte(roadmapContent.String()), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing ROADMAP.md: %v\n", err)
		os.Exit(1)
	}

	// STATE.md
	state := &ProjectState{
		Project:      projectTitle,
		CurrentPhase: 1,
		Step:         "discuss",
		LastAction:   "new-project initialized",
		Updated:      time.Now(),
		PRs:          make(map[int]string),
	}
	for _, ans := range collectedAnswers {
		state.Decisions = append(state.Decisions, ans)
	}
	if err := saveState(state); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing STATE.md: %v\n", err)
		os.Exit(1)
	}

	// Step 12: Completion message
	fmt.Printf("\n%s\n", strings.Repeat("═", 58))
	fmt.Println("  PROJECT INITIALIZED")
	fmt.Println(strings.Repeat("═", 58))
	fmt.Printf("\n  Project   : %s\n", projectTitle)
	fmt.Printf("  Documents : %s\n", coderDir+"/")
	fmt.Printf("    - PROJECT.md\n")
	fmt.Printf("    - REQUIREMENTS.md\n")
	fmt.Printf("    - ROADMAP.md\n")
	fmt.Printf("    - STATE.md  (phase 1, step: discuss)\n")
	fmt.Printf("\n  Next steps:\n")
	fmt.Printf("    coder map-codebase       # analyze existing code\n")
	fmt.Printf("    coder plan \"phase 1\"      # plan first phase\n")
	fmt.Printf("    coder chat               # discuss architecture\n")
	fmt.Println("\n" + strings.Repeat("═", 58))
}
