package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	debugdomain "github.com/trungtran/coder/internal/domain/debug"
	httpclient "github.com/trungtran/coder/internal/transport/http/client"
)

const debugUsage = `Usage: coder debug [error-message] [flags]

AI-powered root cause analysis. Accepts an error message, log file, or stdin
and returns a structured diagnosis with root cause, location, and suggested fix.

EXAMPLES:
  coder debug "panic: nil pointer dereference"
  coder debug --file error.log
  cat crash.log | coder debug
  coder debug --context src/auth/manager.go "nil pointer on line 89"
  coder debug --diff HEAD~1
  coder debug --interactive

FLAGS:
`

func runDebug(args []string) {
	fs := flag.NewFlagSet("debug", flag.ExitOnError)
	filePath := fs.String("file", "", "Read error from a log file")
	contextFile := fs.String("context", "", "Source file to include as context")
	diffRef := fs.String("diff", "", "Git ref to diff against (e.g. HEAD~1)")
	interactive := fs.Bool("interactive", false, "Start interactive debug REPL")
	noMemory := fs.Bool("no-memory", false, "Disable memory context injection")
	noSkills := fs.Bool("no-skills", false, "Disable skill context injection")

	fs.Usage = func() {
		fmt.Fprint(os.Stderr, debugUsage)
		fs.PrintDefaults()
	}
	fs.Parse(args)

	logActivity("debug")

	cfg, _ := loadConfig()
	if cfg == nil {
		cfg = &Config{}
	}

	debugClient := getDebugClient(cfg)
	ctx := context.Background()

	// Interactive mode
	if *interactive {
		runDebugInteractive(ctx, getChatClient(cfg), !*noMemory, !*noSkills)
		return
	}

	// Gather error message
	var errorMsg string
	switch {
	case *filePath != "":
		data, err := os.ReadFile(*filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
			os.Exit(1)
		}
		errorMsg = string(data)
	case len(fs.Args()) > 0:
		errorMsg = strings.Join(fs.Args(), " ")
	default:
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			data, _ := io.ReadAll(os.Stdin)
			errorMsg = string(data)
		}
	}

	if strings.TrimSpace(errorMsg) == "" {
		fs.Usage()
		os.Exit(1)
	}

	// Gather optional context
	var fileCtx, diffCtx string
	if *contextFile != "" {
		data, err := os.ReadFile(*contextFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: cannot read context file: %v\n", err)
		} else {
			fileCtx = string(data)
		}
	}
	if *diffRef != "" {
		out, _ := exec.Command("git", "diff", *diffRef).Output()
		diffCtx = string(out)
	}

	// Truncate
	const maxLen = 8000
	if len(errorMsg) > maxLen {
		errorMsg = errorMsg[:maxLen] + "\n[... truncated ...]"
	}
	if len(fileCtx) > 4000 {
		fileCtx = fileCtx[:4000] + "\n[... truncated ...]"
	}

	fmt.Fprintln(os.Stderr, "  ⟳ Analysing root cause...")

	result, err := debugClient.Debug(ctx, errorMsg, fileCtx, diffCtx, !*noMemory, !*noSkills)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	printDebugResult(result)
}

func printDebugResult(r *debugdomain.DebugResult) {
	line := strings.Repeat("═", 58)
	dash := strings.Repeat("─", 58)

	fmt.Println("\n" + line)
	fmt.Println("  DEBUG ANALYSIS")
	fmt.Println(line + "\n")

	fmt.Printf("ROOT CAUSE  (confidence: %s)\n", strings.ToUpper(r.Confidence))
	for _, l := range wrapText(r.RootCause, 56, "  ") {
		fmt.Println(l)
	}
	fmt.Println()

	if r.Location != "" {
		fmt.Printf("LOCATION\n  %s\n\n", r.Location)
	}

	if r.SuggestedFix != "" {
		fmt.Println("SUGGESTED FIX")
		for _, l := range wrapText(r.SuggestedFix, 56, "  ") {
			fmt.Println(l)
		}
		fmt.Println()
	}

	if len(r.SimilarIssues) > 0 {
		fmt.Println("SIMILAR PAST ISSUES")
		for _, s := range r.SimilarIssues {
			fmt.Printf("  ● %s\n", s)
		}
		fmt.Println()
	}

	if r.FollowUp != "" {
		fmt.Printf("FOLLOW UP\n  %s\n\n", r.FollowUp)
	}

	fmt.Println(dash)
	ctxLabel := ""
	if len(r.ContextUsed.MemoryHits) > 0 {
		ctxLabel = fmt.Sprintf(" · Context: %d memory hits", len(r.ContextUsed.MemoryHits))
	}
	fmt.Printf("  Confidence: %s · Model: %s%s\n", r.Confidence, r.Model, ctxLabel)
	fmt.Println(line)
}

// runDebugInteractive starts an interactive debug REPL using the chat endpoint.
func runDebugInteractive(ctx context.Context, chatClient httpclient.ChatClientIface, injectMemory, injectSkills bool) {

	fmt.Print("\ncoder debug — interactive mode\nDescribe the bug or paste an error. Type /done when resolved, /exit to quit.\n\n")

	scanner := bufio.NewScanner(os.Stdin)
	sessionID := ""

	for {
		fmt.Print(bold("You") + " › ")
		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		switch input {
		case "/done":
			fmt.Println("\nSession saved. Storing fix to memory...")
			fmt.Println("✓ Tip: run `coder memory store` to save this solution for future reference.")
			return
		case "/exit":
			return
		}

		fmt.Print("  ⟳ Searching context...\n\n")
		fmt.Print(bold("Assistant") + " › ")

		result, err := chatClient.ChatStream(ctx, input, sessionID, injectMemory, injectSkills, func(delta string) {
			fmt.Print(delta)
		})
		fmt.Print("\n\n")

		if err != nil {
			fmt.Fprintf(os.Stderr, "  Error: %v\n\n", err)
			continue
		}
		if result != nil {
			sessionID = result.SessionID
		}
	}
}
