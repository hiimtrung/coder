package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	memdomain "github.com/trungtran/coder/internal/domain/memory"
)

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

func runMemoryStore(args []string) {
	logActivity("memory store")
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
		for t := range strings.SplitSeq(*tags, ",") {
			tagList = append(tagList, strings.TrimSpace(t))
		}
	}

	var metadata map[string]any
	if *meta != "" {
		json.Unmarshal([]byte(*meta), &metadata)
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

	var metaFilters map[string]any
	if *meta != "" {
		json.Unmarshal([]byte(*meta), &metaFilters)
	}

	results, err := mgr.Search(context.Background(), query, *scope, nil, memdomain.MemoryType(*memType), metaFilters, *limit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Found %d results for %q:\n\n", len(results), query)
	for _, res := range results {
		fmt.Printf("[%s] %s (Score: %.4f)\n", res.ID[:8], res.Title, res.Score)
		fmt.Printf("   Tags: %s\n", strings.Join(res.Tags, ", "))
		fmt.Printf("   Content: %s\n\n", res.Content)
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
