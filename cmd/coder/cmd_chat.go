package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	grpcclient "github.com/trungtran/coder/internal/transport/grpc/client"
	httpclient "github.com/trungtran/coder/internal/transport/http/client"
)

const chatUsage = `Usage: coder chat [message] [flags]

Start an interactive Q&A session with AI, automatically enriched with your
memory and skill context.

EXAMPLES:
  coder chat                                    # interactive REPL
  coder chat "how should I handle JWT refresh?" # single question
  coder chat --resume                           # resume last session
  coder chat --session abc123                   # resume specific session
  coder chat --list                             # list recent sessions
  coder chat --delete abc123                    # delete a session
  coder chat --no-memory --no-skills "..."      # raw mode, no context injection
  coder chat --file src/auth.go "review this"   # include file as context

SLASH COMMANDS (interactive mode):
  /help                  show commands
  /sessions              list recent sessions
  /resume <id>           load a session
  /clear                 clear history for this session
  /context               show what context was injected last turn
  /exit  or  Ctrl+C      exit and auto-save

FLAGS:
`

func runChat(args []string) {
	fs := flag.NewFlagSet("chat", flag.ExitOnError)
	resume := fs.Bool("resume", false, "Resume the last session")
	sessionID := fs.String("session", "", "Resume a specific session by ID")
	list := fs.Bool("list", false, "List recent sessions")
	delete := fs.String("delete", "", "Delete a session by ID")
	noMemory := fs.Bool("no-memory", false, "Disable memory context injection")
	noSkills := fs.Bool("no-skills", false, "Disable skill context injection")
	fileFlag := fs.String("file", "", "Include file content as context in the prompt")

	fs.Usage = func() {
		fmt.Fprint(os.Stderr, chatUsage)
		fs.PrintDefaults()
	}
	fs.Parse(args)

	logActivity("chat")

	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load config: %v\n", err)
		cfg = &Config{}
	}

	client := getChatClient(cfg)

	ctx := context.Background()

	// --list
	if *list {
		runChatList(ctx, client)
		return
	}

	// --delete <id>
	if *delete != "" {
		if err := client.DeleteSession(ctx, *delete); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Session %s deleted.\n", *delete)
		return
	}

	// Resolve session ID
	sid := *sessionID
	if *resume && sid == "" {
		// Pick the most recent session
		sessions, err := client.ListSessions(ctx)
		if err == nil && len(sessions) > 0 {
			sid = sessions[0].ID
			fmt.Printf("Resuming session: %s — %q\n", sid[:8], sessions[0].Title)
		}
	}

	injectMemory := !*noMemory
	injectSkills := !*noSkills

	// Single-question mode
	if len(fs.Args()) > 0 {
		question := strings.Join(fs.Args(), " ")

		// Prepend file content if --file is provided
		if *fileFlag != "" {
			fileContent, err := os.ReadFile(*fileFlag)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading file %q: %v\n", *fileFlag, err)
				os.Exit(1)
			}
			question = fmt.Sprintf("File: `%s`\n\n```\n%s\n```\n\n%s", *fileFlag, string(fileContent), question)
		}

		runChatSingle(ctx, client, question, sid, injectMemory, injectSkills)
		return
	}

	// Interactive REPL (--file is not used without a message — print hint)
	if *fileFlag != "" {
		fmt.Fprintf(os.Stderr, "Note: --file requires a message. Example: coder chat --file src/auth.go \"review this\"\n")
		os.Exit(1)
	}

	// Interactive REPL
	runChatREPL(ctx, client, sid, injectMemory, injectSkills)
}

// runChatSingle sends one message and prints the streamed response.
func runChatSingle(ctx context.Context, client httpclient.ChatClientIface, message, sessionID string, injectMemory, injectSkills bool) {
	fmt.Print("\n")
	var lastCtx *httpclient.ChatStreamDelta
	var err error

	lastCtx, err = client.ChatStream(ctx, message, sessionID, injectMemory, injectSkills, func(delta string) {
		fmt.Print(delta)
	})

	fmt.Print("\n")

	if err != nil {
		fmt.Fprintf(os.Stderr, "\nError: %v\n", err)
		os.Exit(1)
	}

	if lastCtx != nil && len(lastCtx.ContextUsed.MemoryHits)+len(lastCtx.ContextUsed.SkillHits) > 0 {
		fmt.Printf("\n  %s Context: %s\n", dim("──"), formatContext(lastCtx.ContextUsed))
	}
}

// runChatREPL starts the interactive chat loop.
func runChatREPL(ctx context.Context, client httpclient.ChatClientIface, sessionID string, injectMemory, injectSkills bool) {
	printChatBanner(sessionID)

	scanner := bufio.NewScanner(os.Stdin)
	var lastCtxUsed *httpclient.ContextUsed

	for {
		fmt.Print(bold("You") + " › ")
		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		// Handle slash commands
		if strings.HasPrefix(input, "/") {
			handled, newSID := handleSlashCommand(ctx, client, input, sessionID, lastCtxUsed)
			if handled {
				if newSID != "" {
					sessionID = newSID
				}
				continue
			}
			// /exit
			break
		}

		// Show context search indicator
		fmt.Print("  ⟳ Searching context...\n\n")

		fmt.Print(bold("Assistant") + " › ")

		var result *httpclient.ChatStreamDelta
		var err error

		result, err = client.ChatStream(ctx, input, sessionID, injectMemory, injectSkills, func(delta string) {
			fmt.Print(delta)
		})
		fmt.Print("\n")

		if err != nil {
			fmt.Fprintf(os.Stderr, "\n  Error: %v\n\n", err)
			continue
		}

		if result != nil {
			sessionID = result.SessionID
			if len(result.ContextUsed.MemoryHits)+len(result.ContextUsed.SkillHits) > 0 {
				lastCtxUsed = &result.ContextUsed
				fmt.Printf("\n  %s Context: %s\n", dim("──"), formatContext(result.ContextUsed))
			}
		}
		fmt.Println()
	}

	fmt.Printf("\nSession saved: %s\n", sessionID)
}

// handleSlashCommand processes /commands in the REPL.
// Returns (handled bool, newSessionID string).
func handleSlashCommand(ctx context.Context, client httpclient.ChatClientIface, cmd, _ string, lastCtx *httpclient.ContextUsed) (bool, string) {
	parts := strings.Fields(cmd)
	switch parts[0] {
	case "/help":
		fmt.Println(`
  /help               show this help
  /sessions           list recent sessions
  /resume <id>        load a session
  /clear              start fresh (new session ID)
  /context            show last injected context
  /exit               exit and save`)
		return true, ""

	case "/sessions":
		runChatList(ctx, client)
		return true, ""

	case "/resume":
		if len(parts) < 2 {
			fmt.Println("  Usage: /resume <session-id>")
			return true, ""
		}
		sid := parts[1]
		_, msgs, err := client.GetSession(ctx, sid)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  Error: %v\n", err)
			return true, ""
		}
		fmt.Printf("  Loaded session %s (%d messages)\n\n", sid[:min(8, len(sid))], len(msgs))
		return true, sid

	case "/clear":
		fmt.Println("  Starting new session.")
		return true, "" // returning empty string keeps old session; new session on next message

	case "/context":
		if lastCtx == nil {
			fmt.Println("  No context injected yet.")
		} else {
			fmt.Printf("  Memory: %v\n  Skills: %v\n", lastCtx.MemoryHits, lastCtx.SkillHits)
		}
		return true, ""

	case "/exit":
		return false, "" // break the loop

	default:
		fmt.Printf("  Unknown command: %s. Type /help for commands.\n", parts[0])
		return true, ""
	}
}

// runChatList prints recent sessions.
func runChatList(ctx context.Context, client httpclient.ChatClientIface) {
	sessions, err := client.ListSessions(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing sessions: %v\n", err)
		return
	}
	if len(sessions) == 0 {
		fmt.Println("  No sessions yet. Start chatting with: coder chat")
		return
	}

	fmt.Printf("\n  %-36s  %-40s  %s\n", "ID", "TITLE", "UPDATED")
	fmt.Printf("  %s\n", strings.Repeat("─", 90))
	for _, s := range sessions {
		title := s.Title
		if title == "" {
			title = dim("(untitled)")
		}
		if len(title) > 40 {
			title = title[:37] + "..."
		}
		fmt.Printf("  %-36s  %-40s  %s\n", s.ID, title, s.UpdatedAt.Format(time.RFC3339))
	}
	fmt.Println()
}

// getChatClient returns a configured ChatClient from the CLI config.
func getChatClient(cfg *Config) httpclient.ChatClientIface {
	if cfg.Memory.Protocol == "grpc" {
		addr := cfg.Memory.BaseURL
		if addr == "" {
			addr = "localhost:50051"
		}
		client, err := grpcclient.NewChatClient(addr, cfg.Auth.AccessToken)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to create gRPC chat client: %v — falling back to HTTP\n", err)
		} else {
			return client
		}
	}
	return httpclient.NewChatClient(getHTTPBaseURL(cfg), cfg.Auth.AccessToken)
}

// printChatBanner displays the welcome banner.
func printChatBanner(sessionID string) {
	sessLabel := "new"
	if sessionID != "" {
		sessLabel = sessionID[:min(8, len(sessionID))]
	}
	fmt.Printf(`
╔══════════════════════════════════════════╗
║  coder chat  ·  session: %-14s  ║
║  /help · /sessions · /clear · /exit      ║
╚══════════════════════════════════════════╝

`, sessLabel)
}

// --- formatting helpers ---

func formatContext(ctx httpclient.ContextUsed) string {
	var parts []string
	for _, h := range ctx.SkillHits {
		parts = append(parts, dim(h))
	}
	for _, h := range ctx.MemoryHits {
		parts = append(parts, dim("mem:"+h))
	}
	return strings.Join(parts, " · ")
}

func bold(s string) string  { return "\033[1m" + s + "\033[0m" }
func dim(s string) string   { return "\033[2m" + s + "\033[0m" }

