package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tasagent "github.com/trungtran/coder"
	"github.com/trungtran/coder/internal/installer"
	"github.com/trungtran/coder/internal/memory"
	"github.com/trungtran/coder/internal/profiles"
	"github.com/trungtran/coder/internal/updater"
	"github.com/trungtran/coder/internal/version"
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
  memory              Manage semantic memory (Vector DB)

GLOBAL FLAGS:
  --target, -t <dir>  Path to project directory (default: ".")
  --help, -h          Show this help message

EXAMPLES:
  coder install be                # Setup backend patterns
  coder update                    # Refresh current project skills
  coder memory search "auth"      # Search semantic memory
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

// ── version ──────────────────────────────────────────────────────────────────

func printVersion() {
	fmt.Printf("coder %s\n", version.Version)
	fmt.Printf("  commit:     %s\n", version.Commit)
	fmt.Printf("  build date: %s\n", version.BuildDate)
	fmt.Printf("  releases:   %s\n", version.GitHubReleasesURL())
}

// ── check-update ─────────────────────────────────────────────────────────────

func runCheckUpdate() {
	fmt.Printf("Checking for updates (current: %s)...\n", version.Version)
	release, err := updater.CheckLatestRelease()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	latest := release.TagName
	if updater.IsNewer(version.Version, latest) {
		fmt.Printf("\n✓ New version available: %s → %s\n", version.Version, latest)
		fmt.Printf("  Run 'coder self-update' to upgrade.\n")
		fmt.Printf("  Or download manually: %s\n", release.HTMLURL)
	} else {
		fmt.Printf("✓ You are up to date (%s)\n", version.Version)
	}
}

// ── self-update ───────────────────────────────────────────────────────────────

func runSelfUpdate() {
	fmt.Printf("Checking for updates (current: %s)...\n", version.Version)
	release, err := updater.CheckLatestRelease()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	latest := release.TagName
	if !updater.IsNewer(version.Version, latest) {
		fmt.Printf("✓ Already up to date (%s)\n", version.Version)
		return
	}

	fmt.Printf("New version: %s → %s\n", version.Version, latest)

	asset, ok := updater.FindAsset(release)
	if !ok {
		fmt.Fprintf(os.Stderr,
			"Error: no binary found for your platform (%s)\n"+
				"Download manually: %s\n",
			updater.CurrentPlatformAsset(), release.HTMLURL)
		os.Exit(1)
	}

	if err := updater.SelfUpdate(asset); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ Updated to %s. Run 'coder version' to verify.\n", latest)
}

// ── install ───────────────────────────────────────────────────────────────────

func runInstall(args []string) {
	fs := flag.NewFlagSet("install", flag.ExitOnError)
	target := fs.String("target", "", "Target directory (default: current directory)")
	fs.StringVar(target, "t", "", "Target directory (shorthand)")
	force := fs.Bool("force", false, "Overwrite existing files")
	fs.BoolVar(force, "f", false, "Overwrite existing files (shorthand)")
	dryRun := fs.Bool("dry-run", false, "Show what would be installed without making changes")

	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: coder install <profile> [flags]")
		fs.PrintDefaults()
	}

	if len(args) < 1 || strings.HasPrefix(args[0], "-") {
		fmt.Fprintln(os.Stderr, "Error: profile argument required")
		fmt.Fprintln(os.Stderr, "Usage: coder install <profile> [flags]")
		fmt.Fprintln(os.Stderr, "Run 'coder list' to see available profiles")
		os.Exit(1)
	}
	profileName := args[0]

	if err := fs.Parse(args[1:]); err != nil {
		os.Exit(1)
	}

	targetDir := resolveTargetDir(*target)
	profile, err := profiles.Get(profileName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	opts := installer.Options{DryRun: *dryRun, Force: *force}
	if err := installer.Install(tasagent.AgentFS, profile, targetDir, opts); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// ── update ────────────────────────────────────────────────────────────────────

func runUpdate(args []string) {
	fs := flag.NewFlagSet("update", flag.ExitOnError)
	target := fs.String("target", "", "Target directory (default: current directory)")
	fs.StringVar(target, "t", "", "Target directory (shorthand)")
	dryRun := fs.Bool("dry-run", false, "Show what would be updated without making changes")

	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: coder update [profile] [flags]")
		fs.PrintDefaults()
	}

	// Profile is optional for update — read from manifest if omitted
	var profileName string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		profileName = args[0]
		args = args[1:]
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	targetDir := resolveTargetDir(*target)

	if profileName == "" {
		// Read profile from manifest
		manifest, err := installer.ReadManifest(targetDir)
		if err != nil {
			fmt.Fprintf(os.Stderr,
				"Error: no profile specified and no manifest found in %s\n"+
					"Run 'coder install <profile>' first, or specify a profile: coder update <profile>\n",
				targetDir)
			os.Exit(1)
		}
		profileName = manifest.Profile
		fmt.Printf("Using profile from manifest: %s (installed %s)\n\n",
			manifest.Profile, manifest.InstalledAt.Format("2006-01-02"))
	}

	profile, err := profiles.Get(profileName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Update always overwrites
	opts := installer.Options{DryRun: *dryRun, Force: true}
	if err := installer.Install(tasagent.AgentFS, profile, targetDir, opts); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// ── list ──────────────────────────────────────────────────────────────────────

func runList(args []string) {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if fs.NArg() > 0 {
		profile, err := profiles.Get(fs.Arg(0))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		profiles.PrintProfile(profile)
	} else {
		profiles.PrintAll()
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func resolveTargetDir(flag string) string {
	if flag != "" {
		return flag
	}
	dir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to get current directory: %v\n", err)
		os.Exit(1)
	}
	return dir
}

// ── memory ────────────────────────────────────────────────────────────────────

func runMemory(args []string) {
	if len(args) < 1 || args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		fmt.Fprintln(os.Stderr, "Usage: coder memory <subcommand> [arguments] [flags]")
		fmt.Fprintln(os.Stderr, "\nSUBCOMMANDS:")
		fmt.Fprintln(os.Stderr, "  store <title> <content>   Save a new memory (semantic chunking enabled)")
		fmt.Fprintln(os.Stderr, "  search <query>            Search memory using semantic similarity")
		fmt.Fprintln(os.Stderr, "  list                      List recent memory entries")
		fmt.Fprintln(os.Stderr, "  delete <id>               Remove a memory by its ID")
		fmt.Fprintln(os.Stderr, "  compact                   Optimize DB (re-vectoring, duplicate removal)")
		fmt.Fprintln(os.Stderr, "\nEXAMPLES:")
		fmt.Fprintln(os.Stderr, "  coder memory store \"Go Interfaces\" \"Context on interfaces...\" --tags \"go,pattern\"")
		fmt.Fprintln(os.Stderr, "  coder memory search \"how to handle errors\" --limit 3")
		os.Exit(1)
	}

	sub := args[0]
	switch sub {
	case "store":
		runMemoryStore(args[1:])
	case "search":
		runMemorySearch(args[1:])
	case "list":
		runMemoryList(args[1:])
	case "delete":
		runMemoryDelete(args[1:])
	case "compact":
		runMemoryCompact(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown memory subcommand %q\n", sub)
		os.Exit(1)
	}
}

type Config struct {
	Memory struct {
		Provider     string `json:"provider"`
		DatabaseType string `json:"database_type"`
		APIKey       string `json:"api_key"`
		BaseURL      string `json:"base_url"`
		Model        string `json:"model"`
		PostgresDSN  string `json:"postgres_dsn"`
	} `json:"memory"`
}

func loadConfig() (*Config, error) {
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".coder", "config.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func getMemoryManager() *memory.Manager {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load config: %v\n", err)
		cfg = &Config{}
	}

	var db memory.MemoryService

	dbType := cfg.Memory.DatabaseType
	if dbType == "" {
		dbType = "postgres"
	}

	if dbType == "postgres" {
		dsn := cfg.Memory.PostgresDSN
		if dsn == "" {
			dsn = os.Getenv("POSTGRES_DSN")
		}
		if dsn == "" {
			fmt.Fprintf(os.Stderr, "Error: postgres_dsn is not configured\n")
			os.Exit(1)
		}
		pgdb, err := memory.NewPostgresMemory(dsn)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to open postgres database: %v\n", err)
			os.Exit(1)
		}
		db = pgdb
	} else {
		fmt.Fprintf(os.Stderr, "Error: unsupported database type. Only postgres is supported.\n")
		os.Exit(1)
	}

	providerType := cfg.Memory.Provider
	if providerType == "" {
		providerType = "openai"
	}

	var provider memory.EmbeddingProvider

	if providerType == "ollama" {
		baseURL := cfg.Memory.BaseURL
		if baseURL == "" {
			baseURL = os.Getenv("OLLAMA_BASE_URL")
		}
		if baseURL == "" {
			fmt.Fprintf(os.Stderr, "Error: ollama base_url is not configured. Local ollama is not supported.\n")
			os.Exit(1)
		}
		model := cfg.Memory.Model
		if model == "" {
			model = os.Getenv("OLLAMA_EMBEDDING_MODEL")
		}
		if model == "" {
			model = "mxbai-embed-base" // dimension 1024
		}
		provider = &memory.OllamaEmbeddingProvider{
			BaseURL: baseURL,
			Model:   model,
		}
	} else {
		apiKey := cfg.Memory.APIKey
		if apiKey == "" {
			apiKey = os.Getenv("OPENAI_API_KEY")
		}

		baseURL := cfg.Memory.BaseURL
		if baseURL == "" {
			baseURL = os.Getenv("OPENAI_BASE_URL")
		}

		model := cfg.Memory.Model
		if model == "" {
			model = os.Getenv("OPENAI_EMBEDDING_MODEL")
		}
		if model == "" {
			model = "text-embedding-3-small"
		}

		provider = &memory.OpenAIEmbeddingProvider{
			APIKey:  apiKey,
			BaseURL: baseURL,
			Model:   model,
		}
	}

	return memory.NewManager(db, provider)
}

func runMemoryStore(args []string) {
	fs := flag.NewFlagSet("memory store", flag.ExitOnError)
	tags := fs.String("tags", "", "Comma-separated tags")
	scope := fs.String("scope", "", "Memory scope")
	memType := fs.String("type", "document", "Memory type (fact, rule, preference, skill, event, document)")
	meta := fs.String("meta", "", "JSON string for metadata")

	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: coder memory store <title> <content> [flags]")
		fmt.Fprintln(os.Stderr, "\nFLAGS:")
		fs.PrintDefaults()
	}

	if len(args) < 2 {
		fs.Usage()
		os.Exit(1)
	}

	title := args[0]
	content := args[1]
	fs.Parse(args[2:])

	mgr := getMemoryManager()
	defer mgr.Close()

	tagList := []string{}
	if *tags != "" {
		for _, t := range strings.Split(*tags, ",") {
			tagList = append(tagList, strings.TrimSpace(t))
		}
	}

	var metadata map[string]interface{}
	if *meta != "" {
		json.Unmarshal([]byte(*meta), &metadata)
	}

	id, err := mgr.Store(context.Background(), title, content, memory.MemoryType(*memType), metadata, *scope, tagList)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Memory stored successfully. ID: %s\n", id)
}

func runMemorySearch(args []string) {
	fs := flag.NewFlagSet("memory search", flag.ExitOnError)
	limit := fs.Int("limit", 5, "Number of results to return")
	scope := fs.String("scope", "", "Memory scope")
	memType := fs.String("type", "", "Filter by Memory type")
	meta := fs.String("meta", "", "Filter by JSON metadata")

	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: coder memory search <query> [flags]")
		fmt.Fprintln(os.Stderr, "\nFLAGS:")
		fs.PrintDefaults()
	}

	if len(args) < 1 {
		fs.Usage()
		os.Exit(1)
	}

	query := args[0]
	fs.Parse(args[1:])

	mgr := getMemoryManager()
	defer mgr.Close()

	var metaFilters map[string]interface{}
	if *meta != "" {
		json.Unmarshal([]byte(*meta), &metaFilters)
	}

	results, err := mgr.Search(context.Background(), query, *scope, nil, memory.MemoryType(*memType), metaFilters, *limit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Found %d results for %q:\n\n", len(results), query)
	for _, res := range results {
		fmt.Printf("[%s] %s (Score: %.4f)\n", res.ID[:8], res.Title, res.Score)
		fmt.Printf("   Tags: %s\n", strings.Join(res.Tags, ", "))
		fmt.Printf("   Content: %s\n\n", truncate(res.Content, 100))
	}
}

func runMemoryList(args []string) {
	fs := flag.NewFlagSet("memory list", flag.ExitOnError)
	limit := fs.Int("limit", 10, "Number of results to return")

	fs.Parse(args)

	mgr := getMemoryManager()
	defer mgr.Close()

	items, err := mgr.List(context.Background(), *limit, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Recent memories:\n\n")
	for _, item := range items {
		fmt.Printf("[%s] %s\n", item.ID[:8], item.Title)
		fmt.Printf("   Scope: %s | Created: %s\n\n", item.Scope, item.CreatedAt.Format("2006-01-02"))
	}
}

func runMemoryDelete(args []string) {
	if len(args) < 1 || args[0] == "-h" || args[0] == "--help" {
		fmt.Fprintln(os.Stderr, "Usage: coder memory delete <id>")
		os.Exit(1)
	}

	id := args[0]
	mgr := getMemoryManager()
	defer mgr.Close()

	if err := mgr.Delete(context.Background(), id); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Memory %s deleted.\n", id)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
func runMemoryCompact(args []string) {
	fs := flag.NewFlagSet("memory compact", flag.ExitOnError)
	threshold := fs.Float64("threshold", 0.0, "Similarity threshold for compaction (0.0 for auto)")
	revector := fs.Bool("revector", false, "Re-generate all embeddings")

	fs.Parse(args)

	mgr := getMemoryManager()
	defer mgr.Close()

	if *revector {
		fmt.Println("Re-vectoring all items... this may take a while.")
		if err := mgr.Revector(context.Background()); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Re-vectoring completed.")
	}

	t := float32(*threshold)
	removed, err := mgr.Compact(context.Background(), t)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Compaction completed. Items removed: %d\n", removed)
}
