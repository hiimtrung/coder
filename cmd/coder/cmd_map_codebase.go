package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const mapCodebaseUsage = `Usage: coder map-codebase [area] [flags]

Analyze the codebase and generate structured documentation in .coder/codebase/.
Covers tech stack, architecture, conventions, testing, and concerns.

EXAMPLES:
  coder map-codebase
  coder map-codebase auth
  coder map-codebase --refresh

FLAGS:
`

func runMapCodebase(args []string) {
	fs := flag.NewFlagSet("map-codebase", flag.ExitOnError)
	refresh := fs.Bool("refresh", false, "Re-analyze even if .coder/codebase/ exists")

	fs.Usage = func() {
		fmt.Fprint(os.Stderr, mapCodebaseUsage)
		fs.PrintDefaults()
	}
	fs.Parse(args)

	logActivity("map-codebase")

	// Optional positional area argument
	area := ""
	if len(fs.Args()) > 0 {
		area = strings.Join(fs.Args(), " ")
	}

	// Check if .coder/codebase/ already exists
	codebaseDir := coderPath("codebase")
	if _, err := os.Stat(codebaseDir); err == nil && !*refresh {
		fmt.Print("Codebase map exists. Refresh? [y/N] > ")
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		ans := strings.TrimSpace(strings.ToLower(scanner.Text()))
		if ans != "y" && ans != "yes" {
			fmt.Println("Using existing codebase map. Run with --refresh to force re-analysis.")
			return
		}
	}

	cfg, _ := loadConfig()
	if cfg == nil {
		cfg = &Config{}
	}

	chatClient := getChatClient(cfg)
	ctx := context.Background()

	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	// Banner
	fmt.Printf("\n╔══════════════════════════════════════════════════════╗\n")
	fmt.Printf("║  coder map-codebase  ·  %-30s║\n", truncate(cwd, 30))
	fmt.Printf("╚══════════════════════════════════════════════════════╝\n\n")

	os.MkdirAll(codebaseDir, 0755)

	areaHint := ""
	if area != "" {
		areaHint = fmt.Sprintf("\nFocus area: %s", area)
	}

	type focusTask struct {
		name        string
		prompt      string
		primaryFile string
	}

	tasks := []focusTask{
		{
			name: "tech",
			prompt: fmt.Sprintf(`Analyze this codebase's tech stack. Read go.mod (or package.json) and key source files.

Write STACK.md:
# Tech Stack
## Language & Runtime
## Frameworks & Key Libraries (with versions and purpose)
## Databases
## Build & Test Commands

Write INTEGRATIONS.md:
# External Integrations
## APIs Called
## Message Queues / Events
## External Services

Codebase root: %s%s`, cwd, areaHint),
			primaryFile: "STACK.md",
		},
		{
			name: "arch",
			prompt: fmt.Sprintf(`Analyze this codebase's architecture. Read the directory structure and key files.

Write ARCHITECTURE.md:
# Architecture
## Pattern (clean arch / MVC / hexagonal / etc)
## Layers and their responsibilities
## Data flow
## Key design patterns

Write STRUCTURE.md:
# Directory Structure
(annotated tree of important directories and files)

Codebase root: %s%s`, cwd, areaHint),
			primaryFile: "ARCHITECTURE.md",
		},
		{
			name: "quality",
			prompt: fmt.Sprintf(`Analyze coding conventions and test coverage.

Write CONVENTIONS.md:
# Conventions
## Naming (files, functions, types, variables)
## Error handling pattern
## Import ordering
## Comment style

Write TESTING.md:
# Testing
## Test locations and naming
## Coverage areas (unit/integration/e2e)
## How to run tests
## Coverage estimate

Codebase root: %s%s`, cwd, areaHint),
			primaryFile: "CONVENTIONS.md",
		},
		{
			name: "concerns",
			prompt: fmt.Sprintf(`Identify concerns in this codebase.

Write CONCERNS.md:
# Concerns
## Security (missing auth, hardcoded secrets, injection risks)
## Technical Debt (TODOs, deprecated patterns)
## Missing Tests (critical paths without coverage)
## Performance (N+1 queries, unbounded loops, missing indexes)

Codebase root: %s%s`, cwd, areaHint),
			primaryFile: "CONCERNS.md",
		},
	}

	writtenFiles := []string{}

	for _, task := range tasks {
		fmt.Printf("  Analyzing %s...\n", task.name)

		var content strings.Builder
		_, err := chatClient.ChatStream(ctx, task.prompt, "", true, true, func(delta string) {
			content.WriteString(delta)
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "  Warning: %s analysis failed: %v\n", task.name, err)
			continue
		}

		outPath := coderPath("codebase", task.primaryFile)
		if err := os.WriteFile(outPath, []byte(content.String()), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "  Warning: failed to write %s: %v\n", task.primaryFile, err)
			continue
		}

		writtenFiles = append(writtenFiles, task.primaryFile)
		fmt.Printf("  + %s written to .coder/codebase/%s\n", task.name, task.primaryFile)
	}

	// Commit codebase map to git if possible
	commitCodebaseMap(codebaseDir)

	// Update STATE.md if project is initialized
	if projectExists() {
		state, err := loadState()
		if err == nil {
			state.LastAction = "map-codebase completed"
			if saveErr := saveState(state); saveErr != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to update STATE.md: %v\n", saveErr)
			}
		}
	}

	// Summary
	fmt.Printf("\n%s\n", strings.Repeat("═", 58))
	fmt.Println("  CODEBASE MAP COMPLETE")
	fmt.Println(strings.Repeat("═", 58))
	fmt.Printf("\n  Output: .coder/codebase/\n")
	for _, f := range writtenFiles {
		fmt.Printf("    - %s\n", f)
	}
	fmt.Printf("\n  Next steps:\n")
	fmt.Printf("    coder chat               # discuss findings\n")
	fmt.Printf("    coder plan \"phase 1\"      # start planning\n")
	fmt.Println("\n" + strings.Repeat("═", 58))
}

// commitCodebaseMap attempts to commit the codebase map to git.
// Silently skips if git is not initialized or errors occur.
func commitCodebaseMap(codebaseDir string) {
	// Check if we are in a git repo
	checkCmd := exec.Command("git", "rev-parse", "--git-dir")
	if err := checkCmd.Run(); err != nil {
		return
	}

	addCmd := exec.Command("git", "add", codebaseDir)
	if err := addCmd.Run(); err != nil {
		return
	}

	commitCmd := exec.Command("git", "commit", "-m", "chore: map codebase")
	commitCmd.Stdout = nil
	commitCmd.Stderr = nil
	_ = commitCmd.Run() // ignore error — no staged changes is OK
}
