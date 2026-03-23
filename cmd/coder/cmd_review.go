package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	reviewdomain "github.com/trungtran/coder/internal/domain/review"
	httpclient "github.com/trungtran/coder/internal/transport/http/client"
)

const reviewUsage = `Usage: coder review [file...] [flags]

AI-powered structured code review. Reads a git diff, specific files, or a GitHub PR
and returns categorised feedback (Summary, Strengths, Concerns, Suggestions).

EXAMPLES:
  coder review                        # review current git diff (staged + unstaged)
  coder review --staged               # review only staged changes
  coder review src/auth/service.go    # review a specific file
  coder review --pr 123               # review a GitHub PR (requires gh CLI)
  coder review --focus security       # focus on security concerns only
  coder review --format json          # machine-readable JSON output
  coder review -o review.md           # save output to file
  coder review --min-severity high    # show HIGH concerns only

FLAGS:
`

func runReview(args []string) {
	fs := flag.NewFlagSet("review", flag.ExitOnError)
	staged := fs.Bool("staged", false, "Review only staged changes")
	pr := fs.String("pr", "", "GitHub PR number or URL to review")
	focus := fs.String("focus", "", "Focus area: security, performance, error-handling, etc.")
	format := fs.String("format", "text", "Output format: text | json")
	output := fs.String("o", "", "Save output to file")
	minSeverity := fs.String("min-severity", "", "Minimum severity to display: high | medium | low")
	noMemory := fs.Bool("no-memory", false, "Disable memory context injection")
	noSkills := fs.Bool("no-skills", false, "Disable skill context injection")

	fs.Usage = func() {
		fmt.Fprint(os.Stderr, reviewUsage)
		fs.PrintDefaults()
	}
	fs.Parse(args)

	logActivity("review")

	cfg, _ := loadConfig()
	if cfg == nil {
		cfg = &Config{}
	}

	// --- Gather content to review ---
	var content, reviewType string

	switch {
	case *pr != "":
		// Review a GitHub PR via gh CLI
		reviewType = "pr"
		diff, err := fetchPRDiff(*pr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching PR diff: %v\n", err)
			os.Exit(1)
		}
		content = diff

	case len(fs.Args()) > 0:
		// Review specific files
		reviewType = "file"
		var sb strings.Builder
		for _, f := range fs.Args() {
			data, err := os.ReadFile(f)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", f, err)
				os.Exit(1)
			}
			sb.WriteString(fmt.Sprintf("// File: %s\n", f))
			sb.Write(data)
			sb.WriteString("\n\n")
		}
		content = sb.String()

	default:
		// Review git diff
		reviewType = "diff"
		diff, err := getGitDiff(*staged)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting git diff: %v\n", err)
			os.Exit(1)
		}
		if strings.TrimSpace(diff) == "" {
			fmt.Println("No changes to review.")
			return
		}
		content = diff
	}

	if strings.TrimSpace(content) == "" {
		fmt.Println("Nothing to review.")
		return
	}

	// Truncate very large diffs to avoid LLM context overflow
	const maxContentLen = 12000
	if len(content) > maxContentLen {
		content = content[:maxContentLen] + "\n\n[... truncated for review ...]"
	}

	// --- Call coder-node /v1/review ---
	baseURL := cfg.Memory.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	client := httpclient.NewReviewClient(baseURL, cfg.Auth.AccessToken)

	ctx := context.Background()

	fmt.Fprintf(os.Stderr, "  ⟳ Reviewing with AI")
	if *focus != "" {
		fmt.Fprintf(os.Stderr, " (focus: %s)", *focus)
	}
	fmt.Fprintln(os.Stderr, "...")

	result, err := client.Review(ctx, reviewType, content, *focus, !*noMemory, !*noSkills)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// --- Render output ---
	var rendered string
	if *format == "json" {
		data, _ := json.MarshalIndent(result, "", "  ")
		rendered = string(data)
	} else {
		rendered = renderReviewText(result, *minSeverity)
	}

	if *output != "" {
		if err := os.WriteFile(*output, []byte(rendered), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving output: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Review saved to: %s\n", *output)
	} else {
		fmt.Println(rendered)
	}
}

// renderReviewText formats the review result as human-readable terminal output.
func renderReviewText(r *reviewdomain.ReviewResult, minSeverity string) string {
	var sb strings.Builder
	line := strings.Repeat("═", 58)
	dash := strings.Repeat("─", 58)

	sb.WriteString("\n" + line + "\n")
	sb.WriteString("  CODE REVIEW\n")
	sb.WriteString(line + "\n\n")

	// Summary
	sb.WriteString("SUMMARY\n")
	for _, l := range wrapText(r.Summary, 56, "  ") {
		sb.WriteString(l + "\n")
	}
	sb.WriteString("\n")

	// Strengths
	if len(r.Strengths) > 0 {
		sb.WriteString("STRENGTHS\n")
		for _, s := range r.Strengths {
			sb.WriteString("  ✓ " + s + "\n")
		}
		sb.WriteString("\n")
	}

	// Concerns (filtered by minSeverity)
	minLevel := severityLevel(minSeverity)
	filtered := filterConcerns(r.Concerns, minLevel)
	if len(filtered) > 0 {
		sb.WriteString("CONCERNS\n")
		for _, c := range filtered {
			sev := fmt.Sprintf("[%-6s]", strings.ToUpper(c.Severity))
			sb.WriteString(fmt.Sprintf("  ● %s %s\n", sev, c.Description))
			if c.Location != "" {
				sb.WriteString(fmt.Sprintf("             File: %s\n", c.Location))
			}
			if c.Suggestion != "" {
				sb.WriteString(fmt.Sprintf("             Fix:  %s\n", c.Suggestion))
			}
			sb.WriteString("\n")
		}
	}

	// Suggestions
	if len(r.Suggestions) > 0 {
		sb.WriteString("SUGGESTIONS\n")
		for i, s := range r.Suggestions {
			sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, s))
		}
		sb.WriteString("\n")
	}

	// Footer
	sb.WriteString(dash + "\n")
	sb.WriteString(fmt.Sprintf("  %d file(s) · %d concerns (%d HIGH, %d MEDIUM, %d LOW)\n",
		r.Stats.FilesReviewed,
		r.Stats.ConcernsHigh+r.Stats.ConcernsMedium+r.Stats.ConcernsLow,
		r.Stats.ConcernsHigh, r.Stats.ConcernsMedium, r.Stats.ConcernsLow,
	))
	if r.Model != "" {
		sb.WriteString(fmt.Sprintf("  Model: %s", r.Model))
		if len(r.ContextUsed.SkillHits) > 0 || len(r.ContextUsed.MemoryHits) > 0 {
			sb.WriteString(" · Context injected")
		}
		sb.WriteString("\n")
	}
	sb.WriteString(line + "\n")

	return sb.String()
}

// --- git helpers ---

func getGitDiff(stagedOnly bool) (string, error) {
	var cmd *exec.Cmd
	if stagedOnly {
		cmd = exec.Command("git", "diff", "--cached")
	} else {
		// staged + unstaged
		cmd = exec.Command("git", "diff", "HEAD")
	}
	out, err := cmd.Output()
	if err != nil {
		// Not a git repo or no HEAD — try just diff
		out2, _ := exec.Command("git", "diff").Output()
		return string(out2), nil
	}
	return string(out), nil
}

func fetchPRDiff(prRef string) (string, error) {
	// Extract PR number from URL if needed
	number := prRef
	if strings.Contains(prRef, "/pull/") {
		parts := strings.Split(prRef, "/pull/")
		number = strings.Split(parts[1], "/")[0]
	}

	out, err := exec.Command("gh", "pr", "diff", number).Output()
	if err != nil {
		return "", fmt.Errorf("gh CLI error: %w\nInstall gh: https://cli.github.com", err)
	}
	return string(out), nil
}

// --- severity helpers ---

func severityLevel(s string) int {
	switch strings.ToLower(s) {
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	default:
		return 0
	}
}

func filterConcerns(concerns []reviewdomain.ReviewConcern, minLevel int) []reviewdomain.ReviewConcern {
	if minLevel <= 0 {
		return concerns
	}
	var out []reviewdomain.ReviewConcern
	for _, c := range concerns {
		if severityLevel(c.Severity) >= minLevel {
			out = append(out, c)
		}
	}
	return out
}

// wrapText wraps a string at maxWidth chars with a given indent.
func wrapText(s string, maxWidth int, indent string) []string {
	words := strings.Fields(s)
	var lines []string
	var cur strings.Builder
	cur.WriteString(indent)
	for _, w := range words {
		if cur.Len()+len(w)+1 > maxWidth+len(indent) {
			lines = append(lines, cur.String())
			cur.Reset()
			cur.WriteString(indent)
		}
		if cur.Len() > len(indent) {
			cur.WriteString(" ")
		}
		cur.WriteString(w)
	}
	if cur.Len() > len(indent) {
		lines = append(lines, cur.String())
	}
	return lines
}
