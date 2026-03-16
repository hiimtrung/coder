package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	tasagent "github.com/trungtran/coder"
	"github.com/trungtran/coder/internal/skill"
	"github.com/trungtran/coder/internal/version"
)

func runSkill(args []string) {
	if len(args) < 1 || args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		fmt.Fprintln(os.Stderr, "Usage: coder skill <subcommand> [arguments] [flags]")
		fmt.Fprintln(os.Stderr, "\nSUBCOMMANDS:")
		fmt.Fprintln(os.Stderr, "  search <query>          Semantic search across ingested skills")
		fmt.Fprintln(os.Stderr, "  ingest                  Ingest skills into vector DB")
		fmt.Fprintln(os.Stderr, "  list                    List all ingested skills")
		fmt.Fprintln(os.Stderr, "  info <name>             Show details of a specific skill")
		fmt.Fprintln(os.Stderr, "  delete <name>           Remove a skill from vector DB")
		fmt.Fprintln(os.Stderr, "\nEXAMPLES:")
		fmt.Fprintln(os.Stderr, "  coder skill search \"error handling in golang\" --limit 3")
		fmt.Fprintln(os.Stderr, "  coder skill ingest --source local")
		fmt.Fprintln(os.Stderr, "  coder skill ingest --source github --repo sickn33/antigravity-awesome-skills")
		fmt.Fprintln(os.Stderr, "  coder skill list --category core")
		fmt.Fprintln(os.Stderr, "  coder skill info nestjs")
		os.Exit(1)
	}

	sub := args[0]
	switch sub {
	case "search":
		runSkillSearch(args[1:])
	case "ingest":
		runSkillIngest(args[1:])
	case "list":
		runSkillList(args[1:])
	case "info":
		runSkillInfo(args[1:])
	case "delete":
		runSkillDelete(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown skill subcommand %q\n", sub)
		os.Exit(1)
	}
}

// ── skill search ─────────────────────────────────────────────────────────────

func runSkillSearch(args []string) {
	fs := flag.NewFlagSet("skill search", flag.ExitOnError)
	limit := fs.Int("limit", 5, "Maximum number of results")

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
		fmt.Printf("No results found for %q\n", query)
		return
	}

	fmt.Printf("Found %d skill(s) matching %q:\n\n", len(results), query)
	for _, sr := range results {
		fmt.Printf("━━━ %s (Score: %.4f) ━━━\n", sr.Skill.Name, sr.Score)
		fmt.Printf("  Category: %s | Source: %s\n", sr.Skill.Category, sr.Skill.Source)
		if sr.Skill.Description != "" {
			desc := sr.Skill.Description
			if len(desc) > 120 {
				desc = desc[:120] + "..."
			}
			fmt.Printf("  %s\n", desc)
		}
		fmt.Printf("  Matching chunks (%d):\n", len(sr.Chunks))
		for _, ch := range sr.Chunks {
			title := ch.Title
			if title == "" {
				title = ch.ChunkType
			}
			content := ch.Content
			if len(content) > 80 {
				content = content[:80] + "..."
			}
			fmt.Printf("    [%s] %s\n", ch.ChunkType, title)
			fmt.Printf("           %s\n", content)
		}
		fmt.Println()
	}
}

// ── skill ingest ─────────────────────────────────────────────────────────────

func runSkillIngest(args []string) {
	fs := flag.NewFlagSet("skill ingest", flag.ExitOnError)
	source := fs.String("source", "local", "Ingestion source: local, github")
	repo := fs.String("repo", version.RepoOwner+"/"+version.RepoName, "GitHub repo (e.g., hiimtrung/coder)")
	skillFilter := fs.String("skills", "", "Comma-separated skill names to ingest (default: all)")

	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: coder skill ingest [flags]")
		fmt.Fprintln(os.Stderr, "\nFLAGS:")
		fs.PrintDefaults()
		fmt.Fprintln(os.Stderr, "\nEXAMPLES:")
		fmt.Fprintln(os.Stderr, "  coder skill ingest --source local")
		fmt.Fprintln(os.Stderr, "  coder skill ingest --source github --repo sickn33/antigravity-awesome-skills")
	}

	fs.Parse(args)
	ctx := context.Background()

	client := getSkillClient()
	defer client.Close()

	switch *source {
	case "local":
		runIngestLocal(ctx, client)

	case "github":
		if *repo == "" {
			fmt.Fprintln(os.Stderr, "Error: --repo is required for github source")
			fmt.Fprintln(os.Stderr, "Example: coder skill ingest --source github --repo sickn33/antigravity-awesome-skills")
			os.Exit(1)
		}
		runIngestGitHub(ctx, client, *repo, *skillFilter)

	default:
		fmt.Fprintf(os.Stderr, "Error: unknown source %q (supported: local, github)\n", *source)
		os.Exit(1)
	}
}

// runIngestLocal reads all embedded skills from the binary and sends them to coder-node
// for chunking, embedding, and storage in the vector DB.
func runIngestLocal(ctx context.Context, client skill.Client) {
	skillDirs, err := tasagent.AgentFS.ReadDir(".agents/skills")
	if err != nil || len(skillDirs) == 0 {
		fmt.Println("No embedded skills found in binary. Fetching from official repository...")
		repo := version.RepoOwner + "/" + version.RepoName
		runIngestGitHub(ctx, client, repo, "")
		return
	}

	fmt.Println("Ingesting local embedded skills...")

	successCount := 0
	failCount := 0

	for _, entry := range skillDirs {
		if !entry.IsDir() {
			continue
		}
		skillName := entry.Name()

		// Read SKILL.md
		skillMDPath := ".agents/skills/" + skillName + "/SKILL.md"
		skillMDData, err := tasagent.AgentFS.ReadFile(skillMDPath)
		if err != nil {
			fmt.Printf("  ⚠ %s: no SKILL.md found, skipping\n", skillName)
			continue
		}

		// Read rule files from rules/ directory
		rules := readLocalRules(skillName)

		result, err := client.IngestSkill(ctx, skillName, string(skillMDData), rules, "local", "embedded", "")
		if err != nil {
			fmt.Printf("  ✗ %s: %v\n", skillName, err)
			failCount++
			continue
		}

		fmt.Printf("  ✓ %s (%d chunks)\n", skillName, result.ChunksTotal)
		successCount++
	}

	fmt.Printf("\nLocal ingestion complete: %d succeeded, %d failed\n", successCount, failCount)
}

// readLocalRules reads all rule markdown files from the embedded skill's rules/ directory.
func readLocalRules(skillName string) []skill.RuleFile {
	var rules []skill.RuleFile
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
		rules = append(rules, skill.RuleFile{
			Path:    re.Name(),
			Content: string(ruleData),
		})
	}
	return rules
}

// runIngestGitHub fetches skills from a GitHub repository, including their rule files,
// and sends them to coder-node for ingestion.
func runIngestGitHub(ctx context.Context, client skill.Client, repo string, skillFilter string) {
	fmt.Printf("Fetching skills index from %s...\n", repo)

	fetcher := skill.NewGitHubFetcher()
	entries, err := fetcher.FetchSkillIndex(repo)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Apply filter if specified
	if skillFilter != "" {
		var names []string
		for _, n := range strings.Split(skillFilter, ",") {
			names = append(names, strings.TrimSpace(n))
		}
		entries = skill.FilterSkills(entries, names)
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
func fetchGitHubRules(fetcher *skill.GitHubFetcher, repo, skillPath string) []skill.RuleFile {
	var rules []skill.RuleFile

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

		rules = append(rules, skill.RuleFile{
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
	if len(args) < 1 || args[0] == "-h" || args[0] == "--help" {
		fmt.Fprintln(os.Stderr, "Usage: coder skill info <name>")
		os.Exit(1)
	}

	name := args[0]
	client := getSkillClient()
	defer client.Close()

	sk, chunks, err := client.GetSkill(context.Background(), name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

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
