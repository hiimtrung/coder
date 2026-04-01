package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const sessionUsage = `Usage: coder session <subcommand> [flags]

Save and restore working context — current task, open files, recent decisions,
and next steps. Solves context rot: AI loses context after restart.

SUBCOMMANDS:
  save [description]   Save current working context
  resume               Show context and prepare to continue
  list                 List saved sessions
  show <id>            Show session details
  delete <id>          Delete a session
  export <id> -o file  Export session as a context file for any AI

EXAMPLES:
  coder session save "implementing JWT tokens — need rotation logic"
  coder session resume
  coder session list
  coder session export ses-abc123 -o context.md

`

func runSession(args []string) {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		fmt.Fprint(os.Stderr, sessionUsage)
		os.Exit(1)
	}

	logActivity("session")

	switch args[0] {
	case "save":
		runSessionSave(args[1:])
	case "resume":
		runSessionResume(args[1:])
	case "list":
		runSessionList()
	case "show":
		runSessionShow(args[1:])
	case "delete":
		runSessionDelete(args[1:])
	case "export":
		runSessionExport(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand: %s\nRun `coder session --help` for usage.\n", args[0])
		os.Exit(1)
	}
}

func runSessionSave(args []string) {
	fs := flag.NewFlagSet("session save", flag.ExitOnError)
	fs.Parse(args)

	var description string
	if len(fs.Args()) > 0 {
		description = strings.Join(fs.Args(), " ")
	}

	scanner := bufio.NewScanner(os.Stdin)

	if description == "" {
		fmt.Print("Current task description: ")
		scanner.Scan()
		description = strings.TrimSpace(scanner.Text())
	}

	fmt.Print("Next steps (one per line, blank to finish):\n")
	var nextSteps []string
	for {
		fmt.Printf("  [%d] ", len(nextSteps)+1)
		scanner.Scan()
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			break
		}
		nextSteps = append(nextSteps, line)
	}

	fmt.Print("Open files (one per line, blank to finish):\n")
	var openFiles []string
	for {
		fmt.Printf("  [%d] ", len(openFiles)+1)
		scanner.Scan()
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			break
		}
		openFiles = append(openFiles, line)
	}

	fmt.Print("Recent decisions (one per line, blank to finish):\n")
	var decisions []string
	for {
		fmt.Printf("  [%d] ", len(decisions)+1)
		scanner.Scan()
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			break
		}
		decisions = append(decisions, line)
	}

	now := time.Now()
	id := fmt.Sprintf("ses-%d", now.Unix())

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("---\nid: %s\nsaved: %s\nstatus: active\n---\n\n", id, now.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("# Session: %s\n\n", description))
	sb.WriteString(fmt.Sprintf("## Current Task\n%s\n\n", description))

	if len(nextSteps) > 0 {
		sb.WriteString("## Next Steps\n")
		for i, s := range nextSteps {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, s))
		}
		sb.WriteString("\n")
	}

	if len(openFiles) > 0 {
		sb.WriteString("## Open Files\n")
		for _, f := range openFiles {
			sb.WriteString(fmt.Sprintf("- %s\n", f))
		}
		sb.WriteString("\n")
	}

	if len(decisions) > 0 {
		sb.WriteString("## Recent Decisions\n")
		for _, d := range decisions {
			sb.WriteString(fmt.Sprintf("- %s\n", d))
		}
		sb.WriteString("\n")
	}

	// Write to .coder/sessions/
	sessDir := filepath.Join(".coder", "sessions")
	os.MkdirAll(sessDir, 0755)
	path := filepath.Join(sessDir, id+".md")
	if err := os.WriteFile(path, []byte(sb.String()), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving session: %v\n", err)
		os.Exit(1)
	}

	// Also write .coder/session.md as the "active" session
	os.WriteFile(filepath.Join(".coder", "session.md"), []byte(sb.String()), 0644)

	fmt.Printf("\nSession saved: %s\n", id)
	fmt.Printf("Path: %s\n", path)
}

func runSessionResume(args []string) {
	fs := flag.NewFlagSet("session resume", flag.ExitOnError)
	fs.Parse(args)

	var path string
	if len(fs.Args()) > 0 {
		path = filepath.Join(".coder", "sessions", fs.Args()[0]+".md")
	} else {
		path = filepath.Join(".coder", "session.md")
		if _, err := os.Stat(path); os.IsNotExist(err) {
			path = findLastSession()
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Println("No active session found. Create one with: coder session save")
		return
	}

	fmt.Println("\n" + strings.Repeat("═", 58))
	fmt.Println("  ACTIVE SESSION")
	fmt.Println(strings.Repeat("═", 58))
	fmt.Println(string(data))
	fmt.Println(strings.Repeat("─", 58))
	fmt.Println("\nTip: resume work with your agent using this session context and re-run coder skill/memory retrieval as needed.")
}

func runSessionList() {
	sessDir := filepath.Join(".coder", "sessions")
	entries, err := os.ReadDir(sessDir)
	if err != nil || len(entries) == 0 {
		fmt.Println("No sessions found. Create one with: coder session save")
		return
	}
	fmt.Printf("\n  %-20s  %s\n", "SESSION ID", "SAVED")
	fmt.Printf("  %s\n", strings.Repeat("─", 45))
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		info, _ := e.Info()
		id := strings.TrimSuffix(e.Name(), ".md")
		fmt.Printf("  %-20s  %s\n", id, info.ModTime().Format("2006-01-02 15:04"))
	}
	fmt.Println()
}

func runSessionShow(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: coder session show <id>")
		os.Exit(1)
	}
	path := filepath.Join(".coder", "sessions", args[0]+".md")
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Session not found: %s\n", args[0])
		os.Exit(1)
	}
	fmt.Println(string(data))
}

func runSessionDelete(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: coder session delete <id>")
		os.Exit(1)
	}
	path := filepath.Join(".coder", "sessions", args[0]+".md")
	if err := os.Remove(path); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Session %s deleted.\n", args[0])
}

func runSessionExport(args []string) {
	fs := flag.NewFlagSet("session export", flag.ExitOnError)
	out := fs.String("o", "", "Output file path")
	fs.Parse(args)

	var path string
	if len(fs.Args()) > 0 {
		path = filepath.Join(".coder", "sessions", fs.Args()[0]+".md")
	} else {
		path = findLastSession()
	}

	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Session not found: %v\n", err)
		os.Exit(1)
	}

	content := "# Context for AI\n\nPaste this at the start of your conversation:\n\n---\n\n" + string(data)

	if *out != "" {
		os.WriteFile(*out, []byte(content), 0644)
		fmt.Printf("Session exported to: %s\n", *out)
	} else {
		fmt.Println(content)
	}
}

func findLastSession() string {
	sessDir := filepath.Join(".coder", "sessions")
	entries, err := os.ReadDir(sessDir)
	if err != nil {
		return ""
	}
	var last string
	var lastTime time.Time
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		info, _ := e.Info()
		if info.ModTime().After(lastTime) {
			lastTime = info.ModTime()
			last = filepath.Join(sessDir, e.Name())
		}
	}
	return last
}
