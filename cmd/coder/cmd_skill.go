package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	tasagent "github.com/trungtran/coder"
	skilldomain "github.com/trungtran/coder/internal/domain/skill"
	githubclient "github.com/trungtran/coder/internal/infra/github"
	ucskill "github.com/trungtran/coder/internal/usecase/skill"
	"github.com/trungtran/coder/internal/version"
)

func runSkill(args []string) {
	if len(args) < 1 || args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		fmt.Fprintln(os.Stderr, "Usage: coder skill <subcommand> [arguments] [flags]")
		fmt.Fprintln(os.Stderr, "\nSUBCOMMANDS:")
		fmt.Fprintln(os.Stderr, "  search <query>          Semantic search across ingested skills")
		fmt.Fprintln(os.Stderr, "  resolve <task>          Resolve the active skill set for the current task")
		fmt.Fprintln(os.Stderr, "  active                  Show the current active skill state")
		fmt.Fprintln(os.Stderr, "  ingest                  Ingest skills into vector DB")
		fmt.Fprintln(os.Stderr, "  list                    List all ingested skills")
		fmt.Fprintln(os.Stderr, "  info <name>             Show details of a specific skill")
		fmt.Fprintln(os.Stderr, "  delete <name>           Remove a skill from vector DB")
		fmt.Fprintln(os.Stderr, "\nEXAMPLES:")
		fmt.Fprintln(os.Stderr, "  coder skill search \"error handling in golang\" --limit 3")
		fmt.Fprintln(os.Stderr, "  coder skill resolve \"implement grpc auth flow\" --trigger execution --budget 3")
		fmt.Fprintln(os.Stderr, "  coder skill active --format json")
		fmt.Fprintln(os.Stderr, "  coder skill ingest --source local")
		fmt.Fprintln(os.Stderr, "  coder skill ingest --source github --repo sickn33/antigravity-awesome-skills")
		fmt.Fprintln(os.Stderr, "  coder skill list --category core")
		fmt.Fprintln(os.Stderr, "  coder skill info nestjs")
		fmt.Fprintln(os.Stderr, "  coder skill cache pull ui-ux-pro-max")
		fmt.Fprintln(os.Stderr, "  coder skill cache pull --all")
		fmt.Fprintln(os.Stderr, "  coder skill cache list")
		fmt.Fprintln(os.Stderr, "  coder skill cache clear ui-ux-pro-max")
		fmt.Fprintln(os.Stderr, "  index                   Generate skills_index.json from .agents/skills/")
		os.Exit(1)
	}

	sub := args[0]
	switch sub {
	case "search":
		runSkillSearch(args[1:])
	case "resolve":
		runSkillResolve(args[1:])
	case "active":
		runSkillActive(args[1:])
	case "ingest":
		runSkillIngest(args[1:])
	case "list":
		runSkillList(args[1:])
	case "info":
		runSkillInfo(args[1:])
	case "delete":
		runSkillDelete(args[1:])
	case "cache":
		runSkillCache(args[1:])
	case "index":
		runSkillIndex(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown skill subcommand %q\n", sub)
		os.Exit(1)
	}
}

// ── skill search ─────────────────────────────────────────────────────────────

func runSkillSearch(args []string) {
	logActivity("skill search")
	fs := flag.NewFlagSet("skill search", flag.ExitOnError)
	limit := fs.Int("limit", 10, "Maximum number of results")
	format := fs.String("format", "text", "Output format: text, json")

	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: coder skill search <query> [flags]")
		fmt.Fprintln(os.Stderr, "\nFLAGS:")
		fs.PrintDefaults()
	}

	if len(args) < 1 {
		fs.Usage()
		os.Exit(1)
	}

	query := args[0]
	fs.Parse(args[1:])

	client := getSkillClient()
	defer client.Close()

	results, err := client.SearchSkills(context.Background(), query, *limit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(results) == 0 {
		if *format == "json" {
			writeJSON(struct {
				Query   string                          `json:"query"`
				Results []skilldomain.SkillSearchResult `json:"results"`
			}{
				Query:   query,
				Results: []skilldomain.SkillSearchResult{},
			})
			return
		}
		fmt.Printf("No results found for %q\n", query)
		return
	}

	switch *format {
	case "text":
		// human-readable output handled below
	case "json":
		writeJSON(struct {
			Query   string                          `json:"query"`
			Results []skilldomain.SkillSearchResult `json:"results"`
		}{
			Query:   query,
			Results: results,
		})
		return
	default:
		fmt.Fprintf(os.Stderr, "Error: unsupported format %q (supported: text, json)\n", *format)
		os.Exit(1)
	}

	fmt.Printf("Found %d skill(s) matching %q:\n\n", len(results), query)
	for _, sr := range results {
		fmt.Printf("━━━ %s (Score: %.4f) ━━━\n", sr.Skill.Name, sr.Score)
		fmt.Printf("  Category: %s | Source: %s\n", sr.Skill.Category, sr.Skill.Source)
		if sr.Skill.Description != "" {
			fmt.Printf("  %s\n", sr.Skill.Description)
		}
		fmt.Printf("  Matching chunks (%d):\n", len(sr.Chunks))
		for _, ch := range sr.Chunks {
			title := ch.Title
			if title == "" {
				title = ch.ChunkType
			}
			fmt.Printf("    [%s] %s\n", ch.ChunkType, title)
			fmt.Printf("           %s\n", ch.Content)
		}
		fmt.Println()
	}
}

// ── skill resolve ────────────────────────────────────────────────────────────

func runSkillResolve(args []string) {
	logActivity("skill resolve")
	fs := flag.NewFlagSet("skill resolve", flag.ExitOnError)
	trigger := fs.String("trigger", "initial", "Resolve trigger: initial, clarified, execution, error-recovery, review")
	current := fs.String("current", "", "Comma-separated active skills to compare against (default: load from .coder/active-skills.json)")
	budget := fs.Int("budget", 3, "Maximum number of active skills to keep")
	limit := fs.Int("limit", 0, "Search candidate limit before resolve (default: max(budget*3, 6))")
	format := fs.String("format", "text", "Output format: text, json, raw")
	noSave := fs.Bool("no-save", false, "Do not update .coder/active-skills.json")

	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: coder skill resolve <task> [flags]")
		fmt.Fprintln(os.Stderr, "\nFLAGS:")
		fs.PrintDefaults()
	}

	fs.Parse(args)
	if fs.NArg() < 1 {
		fs.Usage()
		os.Exit(1)
	}
	task := fs.Arg(0)

	currentSkills, err := currentSkillsFromFlagOrState(*current)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to load active skill state: %v\n", err)
		os.Exit(1)
	}

	searchLimit := *limit
	if searchLimit <= 0 {
		searchLimit = max(*budget*3, 6)
	}

	client := getSkillClient()
	defer client.Close()

	results, err := client.SearchSkills(context.Background(), task, searchLimit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	output, state := buildResolveOutput(task, *trigger, *budget, currentSkills, results)
	if !*noSave {
		if err := saveActiveSkillState(state); err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to save active skill state: %v\n", err)
			os.Exit(1)
		}
	}

	switch *format {
	case "text":
		if len(output.Skills) == 0 {
			fmt.Printf("No active skills resolved for %q\n", task)
			return
		}
		fmt.Printf("Resolved %d active skill(s) for %q [%s]:\n\n", len(output.Skills), task, output.Trigger)
		for _, skill := range output.Skills {
			fmt.Printf("━━━ %s (Score: %.4f) ━━━\n", skill.Name, skill.Score)
			fmt.Printf("  Category: %s | Chunks: %d\n", skill.Category, len(skill.Chunks))
			fmt.Printf("  %s\n\n", skill.Reason)
		}
		fmt.Printf("Keep: %s\n", fallbackList(output.Keep))
		fmt.Printf("Add:  %s\n", fallbackList(output.Add))
		fmt.Printf("Drop: %s\n", fallbackList(output.Drop))
	case "json":
		writeJSON(output)
	case "raw":
		selectedByName := make(map[string]skilldomain.SkillSearchResult, len(results))
		for _, result := range results {
			selectedByName[strings.ToLower(result.Skill.Name)] = result
		}
		selected := make([]skilldomain.SkillSearchResult, 0, len(output.Skills))
		for _, skill := range output.Skills {
			if result, ok := selectedByName[strings.ToLower(skill.Name)]; ok {
				selected = append(selected, result)
			}
		}
		fmt.Print(renderRawSkillContext(selected))
	default:
		fmt.Fprintf(os.Stderr, "Error: unsupported format %q (supported: text, json, raw)\n", *format)
		os.Exit(1)
	}
}

// ── skill active ─────────────────────────────────────────────────────────────

func runSkillActive(args []string) {
	logActivity("skill active")
	fs := flag.NewFlagSet("skill active", flag.ExitOnError)
	format := fs.String("format", "text", "Output format: text, json")

	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: coder skill active [flags]")
		fmt.Fprintln(os.Stderr, "\nFLAGS:")
		fs.PrintDefaults()
	}

	fs.Parse(args)

	state, err := loadActiveSkillState()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to load active skill state: %v\n", err)
		os.Exit(1)
	}
	if state == nil {
		empty := &activeSkillState{Skills: []activeSkillEntry{}, Keep: []string{}, Add: []string{}, Drop: []string{}}
		if *format == "json" {
			writeJSON(empty)
			return
		}
		fmt.Printf("No active skill state found. Run `coder skill resolve \"<task>\"` first.\n")
		return
	}

	switch *format {
	case "text":
		fmt.Printf("Active skills for %q [%s]\n", state.Task, state.Trigger)
		if !state.ResolvedAt.IsZero() {
			fmt.Printf("Resolved: %s\n", state.ResolvedAt.Format("2006-01-02 15:04:05"))
		}
		fmt.Printf("Budget:   %d\n\n", state.Budget)
		for _, skill := range state.Skills {
			fmt.Printf("━━━ %s (Score: %.4f) ━━━\n", skill.Name, skill.Score)
			fmt.Printf("  Category: %s | Chunks: %d\n", skill.Category, skill.ChunkCount)
			fmt.Printf("  %s\n\n", skill.Reason)
		}
		fmt.Printf("Keep: %s\n", fallbackList(state.Keep))
		fmt.Printf("Add:  %s\n", fallbackList(state.Add))
		fmt.Printf("Drop: %s\n", fallbackList(state.Drop))
	case "json":
		writeJSON(state)
	default:
		fmt.Fprintf(os.Stderr, "Error: unsupported format %q (supported: text, json)\n", *format)
		os.Exit(1)
	}
}

// ── skill ingest ─────────────────────────────────────────────────────────────

func runSkillIngest(args []string) {
	logActivity("skill ingest")
	fs := flag.NewFlagSet("skill ingest", flag.ExitOnError)
	source := fs.String("source", "auto", "Ingestion source: local, github, auto")
	repo := fs.String("repo", version.RepoOwner+"/"+version.RepoName, "GitHub repo (e.g., hiimtrung/coder)")
	skillFilter := fs.String("skills", "", "Comma-separated skill names to ingest (default: all)")
	includeFiles := fs.Bool("include-files", false, "Also store scripts/data files into DB for cache extraction")

	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: coder skill ingest [flags]")
		fmt.Fprintln(os.Stderr, "\nSOURCES:")
		fmt.Fprintln(os.Stderr, "  auto    Try .agents/skills/ → embedded binary → GitHub (default)")
		fmt.Fprintln(os.Stderr, "  local   Only .agents/skills/ or embedded binary. Error if not found.")
		fmt.Fprintln(os.Stderr, "  github  Fetch from a GitHub repo (requires --repo)")
		fmt.Fprintln(os.Stderr, "\nFLAGS:")
		fs.PrintDefaults()
		fmt.Fprintln(os.Stderr, "\nEXAMPLES:")
		fmt.Fprintln(os.Stderr, "  coder skill ingest                                              # auto (default)")
		fmt.Fprintln(os.Stderr, "  coder skill ingest --source local --include-files               # local only")
		fmt.Fprintln(os.Stderr, "  coder skill ingest --source github --repo owner/repo            # from GitHub")
	}

	fs.Parse(args)
	ctx := context.Background()

	client := getSkillClient()
	defer client.Close()

	switch *source {
	case "local":
		runIngestLocal(ctx, client, *includeFiles)

	case "github":
		if *repo == "" {
			fmt.Fprintln(os.Stderr, "Error: --repo is required for github source")
			fmt.Fprintln(os.Stderr, "Example: coder skill ingest --source github --repo sickn33/antigravity-awesome-skills")
			os.Exit(1)
		}
		runIngestGitHub(ctx, client, *repo, *skillFilter)

	case "auto":
		runIngestAuto(ctx, client, *repo, *includeFiles)

	default:
		fmt.Fprintf(os.Stderr, "Error: unknown source %q (supported: local, github, auto)\n", *source)
		os.Exit(1)
	}
}

// runIngestLocal ingests skills strictly from the local project directory or embedded binary.
// It does NOT fall back to GitHub. If no local skills are found it exits with a clear error.
// Use `coder skill ingest --source auto` for the try-local-then-GitHub behaviour.
func runIngestLocal(ctx context.Context, client skilldomain.SkillClient, includeFiles bool) {
	if includeFiles {
		fmt.Println("(--include-files: scripts/data will be stored for cache extraction)")
	}

	// 1. Try OS filesystem (must be run from a project directory with .agents/skills/)
	if entries, err := os.ReadDir(".agents/skills"); err == nil && len(entries) > 0 {
		fmt.Println("Ingesting skills from project directory (.agents/skills/)...")
		ingestFromFS(ctx, client, entries, readFSSkillMD, readFSRules, includeFiles)
		return
	}

	// 2. Try embedded FS (skills baked into the binary)
	if entries, err := tasagent.AgentFS.ReadDir(".agents/skills"); err == nil && len(entries) > 0 {
		fmt.Println("Ingesting embedded skills from binary...")
		ingestFromFS(ctx, client, entries, readEmbeddedSkillMD, readEmbeddedRules, includeFiles)
		return
	}

	// No local skills — give the user a clear, actionable message
	fmt.Fprintln(os.Stderr, "Error: no local skills found in .agents/skills/")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "To fix this, either:")
	fmt.Fprintln(os.Stderr, "  1. Run from a project where you installed a profile:")
	fmt.Fprintln(os.Stderr, "       coder install <profile>          # then re-run ingest")
	fmt.Fprintln(os.Stderr, "  2. Pull skills from GitHub:")
	fmt.Fprintln(os.Stderr, "       coder skill ingest --source github")
	fmt.Fprintln(os.Stderr, "  3. Try local first then fall back to GitHub automatically:")
	fmt.Fprintln(os.Stderr, "       coder skill ingest --source auto")
	os.Exit(1)
}

// runIngestAuto tries local → embedded → GitHub in order.
// This is the "best effort" mode for users who want skills ingested regardless of environment.
func runIngestAuto(ctx context.Context, client skilldomain.SkillClient, repo string, includeFiles bool) {
	if includeFiles {
		fmt.Println("(--include-files: scripts/data will be stored for cache extraction)")
	}

	// 1. OS filesystem
	if entries, err := os.ReadDir(".agents/skills"); err == nil && len(entries) > 0 {
		fmt.Println("Ingesting skills from project directory (.agents/skills/)...")
		ingestFromFS(ctx, client, entries, readFSSkillMD, readFSRules, includeFiles)
		return
	}

	// 2. Embedded FS
	if entries, err := tasagent.AgentFS.ReadDir(".agents/skills"); err == nil && len(entries) > 0 {
		fmt.Println("Ingesting embedded skills from binary...")
		ingestFromFS(ctx, client, entries, readEmbeddedSkillMD, readEmbeddedRules, includeFiles)
		return
	}

	// 3. GitHub fallback
	fmt.Println("No local skills found. Fetching from GitHub...")
	if repo == "" {
		repo = version.RepoOwner + "/" + version.RepoName
	}
	runIngestGitHub(ctx, client, repo, "")
}

func ingestFromFS(
	ctx context.Context,
	client skilldomain.SkillClient,
	entries []os.DirEntry,
	readMD func(name string) (string, error),
	readRules func(name string) []skilldomain.RuleFile,
	includeFiles bool,
) {
	successCount, failCount := 0, 0

	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		skillName := entry.Name()

		skillMDData, err := readMD(skillName)
		if err != nil {
			fmt.Printf("  ⚠ %s: no SKILL.md found, skipping\n", skillName)
			continue
		}

		rules := readRules(skillName)

		result, err := client.IngestSkill(ctx, skillName, skillMDData, rules, "local", "embedded", "")
		if err != nil {
			fmt.Printf("  ✗ %s: %v\n", skillName, err)
			failCount++
			continue
		}

		filesMsg := ""
		if includeFiles {
			files := readLocalSkillFiles(skillName)
			if len(files) > 0 {
				stored, storeErr := client.StoreSkillFiles(ctx, skillName, files)
				if storeErr != nil {
					filesMsg = fmt.Sprintf(" [files: %v]", storeErr)
				} else {
					filesMsg = fmt.Sprintf(" + %d files", stored)
				}
			}
		}

		fmt.Printf("  ✓ %s (%d chunks%s)\n", skillName, result.ChunksTotal, filesMsg)
		successCount++
	}

	fmt.Printf("\nLocal ingestion complete: %d succeeded, %d failed\n", successCount, failCount)
}

// readFSSkillMD reads SKILL.md from the OS filesystem.
func readFSSkillMD(skillName string) (string, error) {
	data, err := os.ReadFile(filepath.Join(".agents", "skills", skillName, "SKILL.md"))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// readFSRules reads rule files from the OS filesystem.
func readFSRules(skillName string) []skilldomain.RuleFile {
	var rules []skilldomain.RuleFile
	rulesDir := filepath.Join(".agents", "skills", skillName, "rules")
	entries, err := os.ReadDir(rulesDir)
	if err != nil {
		return rules
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(rulesDir, e.Name()))
		if err != nil {
			continue
		}
		rules = append(rules, skilldomain.RuleFile{Path: e.Name(), Content: string(data)})
	}
	return rules
}

// readEmbeddedSkillMD reads SKILL.md from the embedded FS.
func readEmbeddedSkillMD(skillName string) (string, error) {
	data, err := tasagent.AgentFS.ReadFile(".agents/skills/" + skillName + "/SKILL.md")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// readEmbeddedRules reads rule files from the embedded FS.
func readEmbeddedRules(skillName string) []skilldomain.RuleFile {
	return readLocalRules(skillName)
}

// readLocalRules reads all rule markdown files from the embedded skill's rules/ directory.
func readLocalRules(skillName string) []skilldomain.RuleFile {
	var rules []skilldomain.RuleFile
	rulesDir := ".agents/skills/" + skillName + "/rules"

	ruleEntries, err := tasagent.AgentFS.ReadDir(rulesDir)
	if err != nil {
		return rules // rules/ directory may not exist
	}

	for _, re := range ruleEntries {
		if re.IsDir() || !strings.HasSuffix(re.Name(), ".md") {
			continue
		}
		rulePath := rulesDir + "/" + re.Name()
		ruleData, err := tasagent.AgentFS.ReadFile(rulePath)
		if err != nil {
			continue
		}
		rules = append(rules, skilldomain.RuleFile{
			Path:    re.Name(),
			Content: string(ruleData),
		})
	}
	return rules
}

// runIngestGitHub fetches skills from a GitHub repository, including their rule files,
// and sends them to coder-node for ingestion.
func runIngestGitHub(ctx context.Context, client skilldomain.SkillClient, repo string, skillFilter string) {
	fmt.Printf("Fetching skills index from %s...\n", repo)

	fetcher := githubclient.NewGitHubFetcher()
	entries, err := fetcher.FetchSkillIndex(repo)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Apply filter if specified
	if skillFilter != "" {
		var names []string
		for n := range strings.SplitSeq(skillFilter, ",") {
			names = append(names, strings.TrimSpace(n))
		}
		entries = githubclient.FilterSkills(entries, names)
	}

	fmt.Printf("Found %d skill(s) to ingest\n\n", len(entries))

	successCount := 0
	failCount := 0

	for _, entry := range entries {
		// Fetch SKILL.md
		skillMD, err := fetcher.FetchSkillMD(repo, entry.Path)
		if err != nil {
			fmt.Printf("  ✗ %s: %v\n", entry.Name, err)
			failCount++
			continue
		}

		// Fetch rule files from rules/ directory
		rules := fetchGitHubRules(fetcher, repo, entry.Path)
		if len(rules) > 0 {
			fmt.Printf("  … %s: fetched %d rule file(s)\n", entry.Name, len(rules))
		}

		result, err := client.IngestSkill(ctx, entry.Name, skillMD, rules, "github", repo, entry.Category)
		if err != nil {
			fmt.Printf("  ✗ %s: %v\n", entry.Name, err)
			failCount++
			continue
		}

		fmt.Printf("  ✓ %s [%s] (%d chunks)\n", entry.Name, entry.Category, result.ChunksTotal)
		successCount++
	}

	fmt.Printf("\nGitHub ingestion complete: %d succeeded, %d failed\n", successCount, failCount)
}

// fetchGitHubRules attempts to discover and fetch rule files from a skill's rules/
// directory on GitHub. It uses the GitHub Trees API to list files, then fetches each one.
func fetchGitHubRules(fetcher *githubclient.GitHubFetcher, repo, skillPath string) []skilldomain.RuleFile {
	var rules []skilldomain.RuleFile

	// Try to fetch a rules index or discover rule files via the GitHub Contents API
	rulesIndexPath := skillPath + "/rules"

	// Use GitHub Contents API to list files in the rules directory
	ruleFiles, err := fetcher.ListDirectory(repo, "main", rulesIndexPath)
	if err != nil {
		// rules/ directory may not exist — this is normal
		return rules
	}

	for _, rf := range ruleFiles {
		if !strings.HasSuffix(rf, ".md") {
			continue
		}

		filePath := rulesIndexPath + "/" + rf
		content, err := fetcher.FetchSingleFile(repo, "main", filePath)
		if err != nil {
			continue // Skip individual file failures
		}

		rules = append(rules, skilldomain.RuleFile{
			Path:    rf,
			Content: content,
		})
	}

	return rules
}

// ── skill list ───────────────────────────────────────────────────────────────

func runSkillList(args []string) {
	fs := flag.NewFlagSet("skill list", flag.ExitOnError)
	source := fs.String("source", "", "Filter by source (local, github)")
	category := fs.String("category", "", "Filter by category")
	limit := fs.Int("limit", 100, "Maximum number of results")

	fs.Parse(args)

	client := getSkillClient()
	defer client.Close()

	skills, err := client.ListSkills(context.Background(), *source, *category, *limit, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(skills) == 0 {
		fmt.Println("No skills found in vector DB.")
		fmt.Println("Run 'coder skill ingest --source local' to ingest embedded skills.")
		return
	}

	fmt.Printf("Ingested skills (%d):\n\n", len(skills))
	fmt.Printf("  %-25s %-15s %-10s %-8s %s\n", "NAME", "CATEGORY", "SOURCE", "CHUNKS", "UPDATED")
	fmt.Printf("  %-25s %-15s %-10s %-8s %s\n", "────", "────────", "──────", "──────", "───────")
	for _, sk := range skills {
		updated := sk.UpdatedAt.Format("2006-01-02")
		fmt.Printf("  %-25s %-15s %-10s %-8d %s\n", sk.Name, sk.Category, sk.Source, sk.ChunkCount, updated)
	}
}

// ── skill info ───────────────────────────────────────────────────────────────

func runSkillInfo(args []string) {
	fs := flag.NewFlagSet("skill info", flag.ExitOnError)
	format := fs.String("format", "text", "Output format: text, json, raw")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: coder skill info <name> [flags]")
		fmt.Fprintln(os.Stderr, "\nFLAGS:")
		fs.PrintDefaults()
	}
	fs.Parse(args)
	if fs.NArg() < 1 {
		fs.Usage()
		os.Exit(1)
	}

	name := fs.Arg(0)
	client := getSkillClient()
	defer client.Close()

	sk, chunks, err := client.GetSkill(context.Background(), name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	switch *format {
	case "text":
		fmt.Printf("Skill: %s\n", sk.Name)
		fmt.Printf("  ID:          %s\n", sk.ID)
		fmt.Printf("  Category:    %s\n", sk.Category)
		fmt.Printf("  Source:      %s (%s)\n", sk.Source, sk.SourceRepo)
		fmt.Printf("  Risk:        %s\n", sk.Risk)
		fmt.Printf("  Version:     %s\n", sk.Version)
		if len(sk.Tags) > 0 {
			fmt.Printf("  Tags:        %s\n", strings.Join(sk.Tags, ", "))
		}
		fmt.Printf("  Description: %s\n", truncate(sk.Description, 200))
		fmt.Printf("  Created:     %s\n", sk.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("  Updated:     %s\n", sk.UpdatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("\n  Chunks (%d):\n", len(chunks))
		for _, ch := range chunks {
			title := ch.Title
			if title == "" {
				title = "(untitled)"
			}
			fmt.Printf("    [%d] [%s] %s (%d chars)\n", ch.ChunkIndex, ch.ChunkType, title, len(ch.Content))
		}
	case "json":
		writeJSON(struct {
			Skill  *skilldomain.Skill       `json:"skill"`
			Chunks []skilldomain.SkillChunk `json:"chunks"`
		}{
			Skill:  sk,
			Chunks: chunks,
		})
	case "raw":
		fmt.Print(renderRawSkillInfo(sk, chunks))
	default:
		fmt.Fprintf(os.Stderr, "Error: unsupported format %q (supported: text, json, raw)\n", *format)
		os.Exit(1)
	}
}

// ── skill delete ─────────────────────────────────────────────────────────────

func runSkillDelete(args []string) {
	if len(args) < 1 || args[0] == "-h" || args[0] == "--help" {
		fmt.Fprintln(os.Stderr, "Usage: coder skill delete <name>")
		os.Exit(1)
	}

	name := args[0]
	client := getSkillClient()
	defer client.Close()

	if err := client.DeleteSkill(context.Background(), name); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Skill %q deleted from vector DB.\n", name)
}

// ── skill cache ───────────────────────────────────────────────────────────────

func runSkillCache(args []string) {
	if len(args) < 1 || args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		fmt.Fprintln(os.Stderr, "Usage: coder skill cache <subcommand> [arguments]")
		fmt.Fprintln(os.Stderr, "\nSUBCOMMANDS:")
		fmt.Fprintln(os.Stderr, "  pull [<name>|--all]   Extract skill files to ~/.coder/cache/")
		fmt.Fprintln(os.Stderr, "  list                  Show cached skills and their status")
		fmt.Fprintln(os.Stderr, "  clear [<name>|--all]  Remove cached files")
		fmt.Fprintln(os.Stderr, "\nEXAMPLES:")
		fmt.Fprintln(os.Stderr, "  coder skill cache pull ui-ux-pro-max")
		fmt.Fprintln(os.Stderr, "  coder skill cache pull --all")
		fmt.Fprintln(os.Stderr, "  coder skill cache list")
		fmt.Fprintln(os.Stderr, "  coder skill cache clear ui-ux-pro-max")
		os.Exit(1)
	}

	sub := args[0]
	rest := args[1:]

	// Cache commands use the same skilldomain.SkillClient transport (gRPC or HTTP) as
	// every other skill operation — no direct postgres connection required.
	client := getSkillClient()
	cache := ucskill.NewCacheManager(client)

	switch sub {
	case "pull":
		runCachePull(cache, rest)
	case "list":
		runCacheList(cache)
	case "clear":
		runCacheClear(cache, rest)
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown cache subcommand %q\n", sub)
		os.Exit(1)
	}
}

func runCachePull(cache *ucskill.CacheManager, args []string) {
	fs := flag.NewFlagSet("cache pull", flag.ExitOnError)
	all := fs.Bool("all", false, "Pull all skills that have stored files")
	fs.Parse(args)

	ctx := context.Background()

	if *all {
		ok, fail, err := cache.PullAll(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Cache pull complete: %d extracted, %d failed\n", ok, fail)
		return
	}

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Usage: coder skill cache pull <name> | --all")
		os.Exit(1)
	}

	skillName := fs.Arg(0)
	dir, err := cache.Pull(ctx, skillName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ %s → %s\n", skillName, dir)
}

func runCacheList(cache *ucskill.CacheManager) {
	entries := cache.ListCached()
	if len(entries) == 0 {
		fmt.Println("No skills cached. Run: coder skill cache pull --all")
		return
	}

	home, _ := os.UserHomeDir()
	fmt.Printf("Cached skills in ~/.coder/cache/ (%d):\n\n", len(entries))
	fmt.Printf("  %-25s %-12s %-6s %s\n", "SKILL", "VERSION", "FILES", "CACHED AT")
	fmt.Printf("  %-25s %-12s %-6s %s\n", "─────", "───────", "─────", "─────────")
	for name, e := range entries {
		cacheDir := filepath.Join(home, ".coder", "cache", name)
		status := "✓"
		if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
			status = "✗ (missing)"
		}
		fmt.Printf("  %-25s %-12s %-6d %s %s\n",
			name, e.Version, e.FileCount,
			e.CachedAt.Format("2006-01-02 15:04"), status,
		)
	}
}

func runCacheClear(cache *ucskill.CacheManager, args []string) {
	fs := flag.NewFlagSet("cache clear", flag.ExitOnError)
	all := fs.Bool("all", false, "Clear all cached skills")
	fs.Parse(args)

	if *all {
		if err := cache.Clear(""); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("All skill caches cleared.")
		return
	}

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Usage: coder skill cache clear <name> | --all")
		os.Exit(1)
	}

	skillName := fs.Arg(0)
	if err := cache.Clear(skillName); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Cache cleared for %q.\n", skillName)
}

// ── file ingestion helpers ────────────────────────────────────────────────────

// ingestableExts maps file extensions to MIME types for files worth storing.
var ingestableExts = map[string]string{
	".py":   "text/x-python",
	".csv":  "text/csv",
	".json": "application/json",
	".md":   "text/markdown",
	".txt":  "text/plain",
	".js":   "text/javascript",
	".cjs":  "text/javascript",
	".sh":   "text/x-sh",
	".sql":  "text/x-sql",
}

// ingestableDirs lists subdirectories of a skill to scan for files.
var ingestableDirs = []string{"scripts", "data", "references", "templates"}

const maxFileBytes = 5 * 1024 * 1024 // 5 MB

// readLocalSkillFiles scans a skill's subdirectories in the embedded FS
// and returns all ingestable files as SkillFile records.
func readLocalSkillFiles(skillName string) []skilldomain.SkillFile {
	var files []skilldomain.SkillFile
	now := time.Now()

	for _, dir := range ingestableDirs {
		dirPath := ".agents/skills/" + skillName + "/" + dir

		// Walk the embedded FS directory
		_ = fs.WalkDir(tasagent.AgentFS, dirPath, func(fpath string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}

			ext := strings.ToLower(filepath.Ext(fpath))
			contentType, ok := ingestableExts[ext]
			if !ok {
				return nil // skip binary / unknown files
			}

			data, err := tasagent.AgentFS.ReadFile(fpath)
			if err != nil || len(data) == 0 || len(data) > maxFileBytes {
				return nil
			}

			// rel_path relative to the skill root, e.g. "scripts/search.py"
			relPath := path.Join(dir, strings.TrimPrefix(fpath, dirPath+"/"))

			h := sha256.Sum256(data)
			files = append(files, skilldomain.SkillFile{
				RelPath:     relPath,
				ContentType: contentType,
				Content:     data,
				ContentHash: hex.EncodeToString(h[:]),
				SizeBytes:   len(data),
				CreatedAt:   now,
			})
			return nil
		})
	}

	return files
}

// ── skill index ───────────────────────────────────────────────────────────────

func runSkillIndex(args []string) {
	fs := flag.NewFlagSet("skill index", flag.ExitOnError)
	output := fs.String("output", "skills_index.json", "Output file path")
	fs.Parse(args)

	entries, err := os.ReadDir(".agents/skills")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: could not read .agents/skills/: %v\n", err)
		os.Exit(1)
	}

	type indexEntry struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Path        string `json:"path"`
		Category    string `json:"category"`
		Description string `json:"description"`
		Risk        string `json:"risk"`
		Source      string `json:"source"`
		DateAdded   string `json:"date_added"`
	}

	var index []indexEntry
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		name := entry.Name()
		skillPath := filepath.Join(".agents", "skills", name)

		// Parse frontmatter from SKILL.md for description/category
		description := ""
		category := "uncategorized"
		mdData, err := os.ReadFile(filepath.Join(skillPath, "SKILL.md"))
		if err == nil {
			parsed := skilldomain.ParseSkillMD(name, string(mdData))
			if parsed.Description != "" {
				description = parsed.Description
				if len(description) > 120 {
					description = description[:120] + "..."
				}
			}
			if parsed.Category != "" {
				category = parsed.Category
			}
		}

		index = append(index, indexEntry{
			ID:          name,
			Name:        name,
			Path:        ".agents/skills/" + name,
			Category:    category,
			Description: description,
			Risk:        "safe",
			Source:      "local",
			DateAdded:   time.Now().Format("2006-01-02"),
		})
	}

	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(*output, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generated %s with %d skills\n", *output, len(index))
}

func writeJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to encode JSON: %v\n", err)
		os.Exit(1)
	}
}

func fallbackList(items []string) string {
	if len(items) == 0 {
		return "(none)"
	}
	return strings.Join(items, ", ")
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
