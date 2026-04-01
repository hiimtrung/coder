package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	memdomain "github.com/trungtran/coder/internal/domain/memory"
)

func runMemory(args []string) {
	if len(args) < 1 || args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		fmt.Fprintln(os.Stderr, "Usage: coder memory <subcommand> [arguments] [flags]")
		fmt.Fprintln(os.Stderr, "\nSUBCOMMANDS:")
		fmt.Fprintln(os.Stderr, "  store <title> <content>   Save a new memory (semantic chunking enabled)")
		fmt.Fprintln(os.Stderr, "  search <query>            Search memory with lifecycle-aware filtering")
		fmt.Fprintln(os.Stderr, "  recall <task>             Re-recall memory and compute keep/add/drop decisions")
		fmt.Fprintln(os.Stderr, "  active                    Show the current active memory recall state")
		fmt.Fprintln(os.Stderr, "  verify <id>               Refresh verification metadata for a memory/version")
		fmt.Fprintln(os.Stderr, "  supersede <id> <new-id>   Mark one memory/version as replaced by another")
		fmt.Fprintln(os.Stderr, "  audit                     Report lifecycle conflicts and stale active memories")
		fmt.Fprintln(os.Stderr, "  list                      List recent memory entries")
		fmt.Fprintln(os.Stderr, "  delete <id>               Remove a memory by its ID")
		fmt.Fprintln(os.Stderr, "  compact                   Optimize DB (re-vectoring, duplicate removal)")
		fmt.Fprintln(os.Stderr, "\nEXAMPLES:")
		fmt.Fprintln(os.Stderr, "  coder memory store \"Go Interfaces\" \"Context on interfaces...\" --tags \"go,pattern\"")
		fmt.Fprintln(os.Stderr, "  coder memory store \"Auth decision\" \"Use rotating refresh tokens\" --type decision --replace-active")
		fmt.Fprintln(os.Stderr, "  coder memory search \"how to handle errors\" --limit 3 --format json")
		fmt.Fprintln(os.Stderr, "  coder memory recall \"grpc auth flow\" --trigger execution --budget 5")
		fmt.Fprintln(os.Stderr, "  coder memory active --format json")
		fmt.Fprintln(os.Stderr, "  coder memory verify 7f9c4c1e --verified-by phase-3 --confidence 0.9")
		fmt.Fprintln(os.Stderr, "  coder memory supersede old-parent new-parent")
		fmt.Fprintln(os.Stderr, "  coder memory audit --unverified-days 90")
		os.Exit(1)
	}

	sub := args[0]
	switch sub {
	case "store":
		runMemoryStore(args[1:])
	case "search":
		runMemorySearch(args[1:])
	case "recall":
		runMemoryRecall(args[1:])
	case "active":
		runMemoryActive(args[1:])
	case "verify":
		runMemoryVerify(args[1:])
	case "supersede":
		runMemorySupersede(args[1:])
	case "audit":
		runMemoryAudit(args[1:])
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

func runMemoryStore(args []string) {
	logActivity("memory store")
	fs := flag.NewFlagSet("memory store", flag.ExitOnError)
	tags := fs.String("tags", "", "Comma-separated tags")
	scope := fs.String("scope", "", "Memory scope")
	memType := fs.String("type", "document", "Memory type (fact, rule, decision, pattern, event, document; preference/skill are legacy)")
	meta := fs.String("meta", "", "JSON string for metadata")
	status := fs.String("status", "", "Lifecycle status (active, superseded, expired, archived, draft)")
	key := fs.String("key", "", "Canonical key used to version related memories")
	supersedes := fs.String("supersedes", "", "Memory ID or parent ID this memory replaces")
	validFrom := fs.String("valid-from", "", "RFC3339 timestamp when this memory becomes valid")
	validUntil := fs.String("valid-until", "", "RFC3339 timestamp when this memory expires")
	verifiedAt := fs.String("verified-at", "", "RFC3339 timestamp when this memory was last verified")
	verifiedBy := fs.String("verified-by", "", "Actor or workflow that verified this memory")
	confidence := fs.Float64("confidence", -1, "Confidence score between 0 and 1")
	source := fs.String("source", "", "Source reference (PR, commit, doc, issue)")
	replaceActive := fs.Bool("replace-active", false, "Supersede the current active memory with the same canonical key")

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
		for t := range strings.SplitSeq(*tags, ",") {
			tagList = append(tagList, strings.TrimSpace(t))
		}
	}

	var metadata map[string]any
	if *meta != "" {
		json.Unmarshal([]byte(*meta), &metadata)
	}
	metadata = memdomain.EnsureMetadata(metadata)

	if *status != "" {
		metadata[memdomain.MetadataKeyStatus] = *status
	}
	if *key != "" {
		metadata[memdomain.MetadataKeyCanonicalKey] = *key
	}
	if *supersedes != "" {
		metadata[memdomain.MetadataKeySupersedesID] = *supersedes
	}
	if *validFrom != "" {
		at, err := parseRFC3339Flag("valid-from", *validFrom)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		memdomain.SetMetadataTime(metadata, memdomain.MetadataKeyValidFrom, at)
	}
	if *validUntil != "" {
		at, err := parseRFC3339Flag("valid-until", *validUntil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		memdomain.SetMetadataTime(metadata, memdomain.MetadataKeyValidTo, at)
	}
	if *verifiedAt != "" {
		at, err := parseRFC3339Flag("verified-at", *verifiedAt)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		memdomain.SetMetadataTime(metadata, memdomain.MetadataKeyLastVerifiedAt, at)
	}
	if *verifiedBy != "" {
		metadata[memdomain.MetadataKeyVerifiedBy] = *verifiedBy
	}
	if *confidence >= 0 {
		metadata[memdomain.MetadataKeyConfidence] = *confidence
	}
	if *source != "" {
		metadata[memdomain.MetadataKeySourceRef] = *source
	}
	if *replaceActive {
		metadata[memdomain.ControlKeyReplaceActive] = true
	}

	id, err := mgr.Store(context.Background(), title, content, memdomain.MemoryType(*memType), metadata, *scope, tagList)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Memory stored successfully. ID: %s\n", id)
}

func runMemorySearch(args []string) {
	logActivity("memory search")
	fs := flag.NewFlagSet("memory search", flag.ExitOnError)
	limit := fs.Int("limit", 5, "Number of results to return")
	format := fs.String("format", "text", "Output format: text, json, raw")
	scope := fs.String("scope", "", "Memory scope")
	memType := fs.String("type", "", "Filter by Memory type")
	meta := fs.String("meta", "", "Filter by JSON metadata")
	status := fs.String("status", "", "Lifecycle status filter")
	key := fs.String("key", "", "Canonical key filter")
	asOf := fs.String("as-of", "", "RFC3339 timestamp to evaluate validity at a point in time")
	includeStale := fs.Bool("include-stale", false, "Include stale, expired, or superseded memories")
	history := fs.Bool("history", false, "Return multiple versions for the same canonical key")

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

	var metaFilters map[string]any
	if *meta != "" {
		json.Unmarshal([]byte(*meta), &metaFilters)
	}
	metaFilters = memdomain.EnsureMetadata(metaFilters)
	if *status != "" {
		metaFilters[memdomain.FilterKeyStatus] = *status
	}
	if *key != "" {
		metaFilters[memdomain.FilterKeyCanonicalKey] = *key
	}
	if *includeStale {
		metaFilters[memdomain.FilterKeyIncludeStale] = true
	}
	if *history {
		metaFilters[memdomain.FilterKeyHistory] = true
	}
	if *asOf != "" {
		at, err := parseRFC3339Flag("as-of", *asOf)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		metaFilters[memdomain.FilterKeyAsOf] = at.UTC().Format(time.RFC3339)
	}

	results, err := mgr.Search(context.Background(), query, *scope, nil, memdomain.MemoryType(*memType), metaFilters, *limit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	state := buildActiveMemoryState(query, *scope, *memType, *limit, *status, *key, *asOf, *includeStale, *history, results)
	if err := saveActiveMemoryState(state); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to save active memory state: %v\n", err)
		os.Exit(1)
	}

	switch *format {
	case "text":
		fmt.Printf("Found %d results for %q:\n\n", len(results), query)
		for _, res := range results {
			fmt.Printf("[%s] %s (Score: %.4f)\n", shortMemoryID(res.ID), res.Title, res.Score)
			fmt.Printf("   Tags: %s\n", strings.Join(res.Tags, ", "))
			fmt.Printf("   Status: %s | Key: %s\n", memdomain.StatusForKnowledge(res.Knowledge), memdomain.CanonicalKeyForKnowledge(res.Knowledge))
			if memdomain.MetadataBool(res.Metadata, memdomain.MetadataKeyConflictDetected) {
				fmt.Printf("   Conflict: %v active versions\n", res.Metadata[memdomain.MetadataKeyConflictCount])
				if titles := memdomain.MetadataStringSlice(res.Metadata, memdomain.MetadataKeyConflictTitles); len(titles) > 0 {
					fmt.Printf("   Candidates: %s\n", strings.Join(titles, " | "))
				}
			}
			if verifiedAt, ok := memdomain.LastVerifiedAtForKnowledge(res.Knowledge); ok {
				fmt.Printf("   Verified: %s\n", verifiedAt.Format(time.RFC3339))
			}
			fmt.Printf("   Content: %s\n\n", res.Content)
		}
	case "json":
		writeJSON(buildMemorySearchOutput(query, *scope, *memType, *limit, *status, *key, *asOf, *includeStale, *history, results))
	case "raw":
		fmt.Print(renderRawMemoryContext(query, results))
	default:
		fmt.Fprintf(os.Stderr, "Error: unsupported format %q (supported: text, json, raw)\n", *format)
		os.Exit(1)
	}
}

func runMemoryVerify(args []string) {
	logActivity("memory verify")
	fs := flag.NewFlagSet("memory verify", flag.ExitOnError)
	verifiedAt := fs.String("verified-at", "", "RFC3339 timestamp when this memory was verified (defaults to now)")
	verifiedBy := fs.String("verified-by", "", "Actor or workflow that verified this memory")
	confidence := fs.Float64("confidence", -1, "Confidence score between 0 and 1")
	source := fs.String("source", "", "Source reference used for verification")

	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: coder memory verify <id> [flags]")
		fmt.Fprintln(os.Stderr, "\nFLAGS:")
		fs.PrintDefaults()
	}

	if len(args) < 1 {
		fs.Usage()
		os.Exit(1)
	}

	id := args[0]
	fs.Parse(args[1:])

	var opts memdomain.VerifyOptions
	if *verifiedAt != "" {
		at, err := parseRFC3339Flag("verified-at", *verifiedAt)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		opts.VerifiedAt = at
	}
	opts.VerifiedBy = *verifiedBy
	opts.SourceRef = *source
	if *confidence >= 0 {
		opts.Confidence = confidence
	}

	mgr := getMemoryManager()
	defer mgr.Close()

	updated, err := mgr.Verify(context.Background(), id, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Verified %d memory row(s) for %s\n", updated, id)
}

func runMemorySupersede(args []string) {
	logActivity("memory supersede")
	fs := flag.NewFlagSet("memory supersede", flag.ExitOnError)

	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: coder memory supersede <id> <replacement-id>")
	}

	if len(args) < 2 {
		fs.Usage()
		os.Exit(1)
	}

	targetID := args[0]
	replacementID := args[1]
	fs.Parse(args[2:])

	mgr := getMemoryManager()
	defer mgr.Close()

	updated, err := mgr.Supersede(context.Background(), targetID, replacementID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Superseded %s with %s across %d row(s)\n", targetID, replacementID, updated)
}

func runMemoryAudit(args []string) {
	logActivity("memory audit")
	fs := flag.NewFlagSet("memory audit", flag.ExitOnError)
	scope := fs.String("scope", "", "Restrict audit to a memory scope")
	unverifiedDays := fs.Int("unverified-days", 180, "Flag active memories not verified within this many days")
	jsonOut := fs.Bool("json", false, "Print the audit report as JSON")

	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: coder memory audit [flags]")
		fmt.Fprintln(os.Stderr, "\nFLAGS:")
		fs.PrintDefaults()
	}

	fs.Parse(args)

	mgr := getMemoryManager()
	defer mgr.Close()

	report, err := mgr.Audit(context.Background(), memdomain.AuditOptions{
		Scope:          *scope,
		UnverifiedDays: *unverifiedDays,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if *jsonOut {
		data, _ := json.MarshalIndent(report, "", "  ")
		fmt.Println(string(data))
		return
	}

	fmt.Printf("Memory audit generated at %s\n", report.GeneratedAt.Format(time.RFC3339))
	if len(report.Findings) == 0 {
		fmt.Println("\nNo lifecycle findings.")
		return
	}

	fmt.Printf("\nFound %d lifecycle finding(s):\n\n", len(report.Findings))
	for _, finding := range report.Findings {
		fmt.Printf("[%s] %s\n", finding.Type, firstAuditLabel(finding))
		if len(finding.VersionIDs) > 0 {
			fmt.Printf("   Versions: %s\n", strings.Join(finding.VersionIDs, ", "))
		}
		if len(finding.Titles) > 0 {
			fmt.Printf("   Titles: %s\n", strings.Join(finding.Titles, " | "))
		}
		fmt.Printf("   Detail: %s\n\n", finding.Details)
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
		fmt.Printf("   Scope: %s | Status: %s | Created: %s\n\n", item.Scope, memdomain.StatusForKnowledge(item), item.CreatedAt.Format("2006-01-02"))
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

func firstAuditLabel(finding memdomain.AuditFinding) string {
	if finding.CanonicalKey != "" {
		return finding.CanonicalKey
	}
	if len(finding.Titles) > 0 {
		return finding.Titles[0]
	}
	return string(finding.Type)
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

func parseRFC3339Flag(name string, value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("%s must be RFC3339, got %q", name, value)
	}
	return parsed.UTC(), nil
}
