package main

import (
	"fmt"
	"os"
)

const usage = `coder — Professional AI Agent Orchestration CLI

USAGE:
  coder <command> [arguments] [flags]

CORE COMMANDS:
  install <profile>   Install agent skills, rules, and workflows (e.g., 'be', 'fe', 'golang')
  update [profile]    Sync/Force-update existing project configuration
  list [profile]      Explore available profiles or specific skill details
  version             Display CLI version and build information

MAINTENANCE:
  check-update        Search for newer versions on GitHub
  self-update         Upgrade coder to the latest version automatically
  login               Configure coder-node connection (protocol and URL)
  skill               Manage skills in vector DB (search, ingest, list)
  memory              Manage semantic memory (Vector DB)

GLOBAL FLAGS:
  --target, -t <dir>  Path to project directory (default: ".")
  --help, -h          Show this help message

EXAMPLES:
  coder install be                # Setup backend patterns
  coder update                    # Refresh current project skills
  coder memory search "auth"      # Search semantic memory
  coder skill search "error"      # Search ingested skills
  coder skill ingest --source local # Ingest local skills into vector DB
  coder --version                 # Check version

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
	case "skill":
		runSkill(os.Args[2:])
	case "memory":
		runMemory(os.Args[2:])
	case "help", "--help", "-h":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown command %q\n\n", cmd)
		fmt.Print(usage)
		os.Exit(1)
	}
}
