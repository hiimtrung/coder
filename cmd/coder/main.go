package main

import (
	"fmt"
	"os"
)

const usage = `coder — Professional AI Agent Orchestration CLI

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

AI WORKFLOW COMMANDS:
  chat                Q&A with AI, context-enriched from memory + skills
  review              Structured AI code review of diff, files, or PR
  debug               Root cause analysis for errors, logs, and stack traces
  plan                Generate a structured implementation plan with Q&A
  qa                  UAT verification workflow with persistent progress
  session             Save and restore working context across sessions
  workflow            Auto-chain: plan → review → qa → fix → done

PROJECT LIFECYCLE COMMANDS:
  new-project         AI-guided project init: requirements → roadmap → STATE.md
  map-codebase        Analyse existing codebase → STACK/ARCH/CONVENTIONS/CONCERNS docs
  discuss-phase <N>   Gray-area Q&A for phase N → CONTEXT.md
  plan-phase <N>      Research + generate XML implementation plans for phase N
  execute-phase <N>   Execute phase N plans with atomic git commits per task
  ship [N]            Create PR for phase N via gh CLI with AI-generated body
  progress            Show project progress (phases, step, blockers, PRs)
  next                Print the next recommended command based on current state
  milestone <action>  Manage phase lifecycle: audit | complete | archive | next

PROJECT UTILITIES:
  todo                Manage project backlog (list / add / done / clear)
  stats               Show project statistics (phases, commits, plans, files)
  health              Check project health (artifacts, blockers, stale state)
  note <text>         Record a decision, blocker, or backlog item to STATE.md
  do <task>           Run a one-off AI task with full project context

MAINTENANCE:
  check-update        Search for newer versions on GitHub
  self-update         Upgrade coder to the latest version automatically
  login               Configure coder-node connection (protocol and URL)
  token               Manage your access token (show, rotate)
  skill               Manage skills in vector DB (search, ingest, list)
  memory              Manage semantic memory (Vector DB)

GLOBAL FLAGS:
  --target, -t <dir>  Path to project directory (default: ".")
  --help, -h          Show this help message

EXAMPLES:
  coder new-project "build a CLI task manager in Go"
  coder map-codebase
  coder discuss-phase 1
  coder plan-phase 1
  coder execute-phase 1
  coder ship 1
  coder progress
  coder next
  coder milestone complete 1
  coder todo add "investigate rate limiting"
  coder note "decided to use JWT with refresh tokens"
  coder chat "explain JWT refresh"
  coder review
  coder memory search "auth"
  coder skill search "error"

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
	case "chat":
		runChat(os.Args[2:])
	case "review":
		runReview(os.Args[2:])
	case "debug":
		runDebug(os.Args[2:])
	case "plan":
		runPlan(os.Args[2:])
	case "qa":
		runQA(os.Args[2:])
	case "session":
		runSession(os.Args[2:])
	case "workflow":
		runWorkflow(os.Args[2:])
	case "new-project":
		runNewProject(os.Args[2:])
	case "map-codebase":
		runMapCodebase(os.Args[2:])
	case "discuss-phase":
		runDiscussPhase(os.Args[2:])
	case "plan-phase":
		runPlanPhase(os.Args[2:])
	case "execute-phase":
		runExecutePhase(os.Args[2:])
	case "ship":
		runShip(os.Args[2:])
	case "progress":
		runProgress(os.Args[2:])
	case "next":
		runNext(os.Args[2:])
	case "milestone":
		runMilestone(os.Args[2:])
	case "todo":
		runTodo(os.Args[2:])
	case "stats":
		runStats(os.Args[2:])
	case "health":
		runHealth(os.Args[2:])
	case "note":
		runNote(os.Args[2:])
	case "do":
		runDo(os.Args[2:])
	case "help", "--help", "-h":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown command %q\n\n", cmd)
		fmt.Print(usage)
		os.Exit(1)
	}
}
