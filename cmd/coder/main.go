package main

import (
	"fmt"
	"os"
)

const usage = `coder — AI Agent Memory & Skill CLI

USAGE:
  coder <command> [arguments] [flags]

CORE COMMANDS:
  install <profile>          Install agent configs per-project (e.g., 'be', 'fe', 'fullstack')
  install global [profile]   Install agent configs globally for the current user
  update [profile]           Sync/Force-update existing project configuration
  update global              Sync/Force-update globally installed configs
  remove global              Remove globally installed configs
  list [profile]             Explore available profiles or specific skill details
  version                    Display CLI version and build information

MEMORY & SKILL COMMANDS:
  memory                     Manage semantic memory (store, search, list, compact)
  skill                      Manage skills in vector DB (search, ingest, list)
  session                    Save and restore working context across sessions

PROJECT STATE COMMANDS:
  progress                   Show project progress (phases, step, blockers, PRs)
  next                       Print the next recommended command based on current state
  milestone <action>         Manage phase lifecycle: audit | complete | archive | next

MAINTENANCE:
  check-update        Search for newer versions on GitHub
  self-update         Upgrade coder to the latest version automatically
  login               Configure coder-node connection (protocol and URL)
  token               Manage your access token (show, rotate)

GLOBAL FLAGS:
  --target, -t <dir>  Path to project directory (default: ".")
  --help, -h          Show this help message

EXAMPLES:
  coder memory search "auth pattern"
  coder memory store "JWT Refresh" "Use short-lived access tokens..." --tags "auth,backend"
  coder skill search "error handling"
  coder session save
  coder progress
  coder next
  coder milestone complete 1
  coder note "decided to use JWT with refresh tokens"

Run 'coder <command> --help' for specific command details.
`

func main() {
	if len(os.Args) < 2 {
		fmt.Print(usage)
		os.Exit(1)
	}

	cmd := os.Args[1]
	switch cmd {
	case "install":
		runInstall(os.Args[2:])
	case "update":
		runUpdate(os.Args[2:])
	case "list":
		runList(os.Args[2:])
	case "version", "--version", "-v":
		printVersion()
	case "check-update":
		runCheckUpdate()
	case "self-update":
		runSelfUpdate()
	case "login":
		runLogin(os.Args[2:])
	case "token":
		runToken(os.Args[2:])
	case "skill":
		runSkill(os.Args[2:])
	case "memory":
		runMemory(os.Args[2:])
	case "remove":
		runRemove(os.Args[2:])
	case "session":
		runSession(os.Args[2:])
	case "progress":
		runProgress(os.Args[2:])
	case "next":
		runNext(os.Args[2:])
	case "milestone":
		runMilestone(os.Args[2:])
	case "help", "--help", "-h":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown command %q\n\n", cmd)
		fmt.Print(usage)
		os.Exit(1)
	}
}
