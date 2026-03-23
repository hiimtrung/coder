package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	httpclient "github.com/trungtran/coder/internal/transport/http/client"
)

const qaUsage = `Usage: coder qa [feature-description] [flags]

UAT verification workflow. Loads acceptance criteria from a PLAN.md, presents
each test case one at a time, tracks pass/fail, and auto-diagnoses failures.
State is persisted in UAT.md — resume safely after Ctrl+C.

EXAMPLES:
  coder qa --plan .coder/plans/PLAN-auth-jwt.md
  coder qa "user authentication feature"    # auto-generate test cases
  coder qa --resume                         # resume last session
  coder qa --session qa-abc123              # resume specific session
  coder qa --list                           # list QA sessions
  coder qa --report                         # print report for last session

FLAGS:
`

type qaTest struct {
	Number   int
	Title    string
	Expected string
	Result   string // "pending" | "pass" | "issue" | "skip"
	Reported string
	Severity string
	RootCause string
}

type qaState struct {
	ID          string
	PlanFile    string
	Status      string // "new" | "in_progress" | "complete"
	Started     time.Time
	Updated     time.Time
	Tests       []qaTest
	CurrentTest int
}

func runQA(args []string) {
	fs := flag.NewFlagSet("qa", flag.ExitOnError)
	planFile := fs.String("plan", "", "PLAN.md file to load test cases from")
	resume := fs.Bool("resume", false, "Resume last QA session")
	sessionID := fs.String("session", "", "Resume specific QA session")
	list := fs.Bool("list", false, "List QA sessions")
	report := fs.Bool("report", false, "Print report for last session")
	reportOut := fs.String("o", "", "Save report to file")

	fs.Usage = func() {
		fmt.Fprint(os.Stderr, qaUsage)
		fs.PrintDefaults()
	}
	fs.Parse(args)

	logActivity("qa")

	cfg, _ := loadConfig()
	if cfg == nil {
		cfg = &Config{}
	}

	baseURL := cfg.Memory.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	chatClient := httpclient.NewChatClient(baseURL, cfg.Auth.AccessToken)
	debugClient := httpclient.NewDebugClient(baseURL, cfg.Auth.AccessToken)
	ctx := context.Background()

	// List sessions
	if *list {
		listQASessions()
		return
	}

	// Report
	if *report {
		sid := *sessionID
		if sid == "" {
			sid = findLastQASession()
		}
		if sid == "" {
			fmt.Println("No QA session found.")
			return
		}
		printQAReport(loadQAState(sid), *reportOut)
		return
	}

	// Load or create state
	var state *qaState

	if *resume || *sessionID != "" {
		sid := *sessionID
		if sid == "" {
			sid = findLastQASession()
		}
		if sid != "" {
			state = loadQAState(sid)
			if state != nil {
				fmt.Printf("Resuming QA session: %s\n\n", sid)
			}
		}
	}

	if state == nil {
		// Generate test cases
		if *planFile != "" {
			state = qaFromPlan(ctx, chatClient, *planFile)
		} else if len(fs.Args()) > 0 {
			feature := strings.Join(fs.Args(), " ")
			state = qaFromFeature(ctx, chatClient, feature)
		} else {
			fs.Usage()
			os.Exit(1)
		}
	}

	if state == nil {
		fmt.Fprintln(os.Stderr, "Failed to create QA session.")
		os.Exit(1)
	}

	runQASession(ctx, state, chatClient, debugClient)
}

// qaFromPlan loads acceptance criteria from a PLAN.md and generates test cases.
func qaFromPlan(ctx context.Context, chatClient *httpclient.ChatClient, planFile string) *qaState {
	data, err := os.ReadFile(planFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading plan: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Loading test cases from plan...")
	prompt := fmt.Sprintf(`Based on this implementation plan, generate a list of concrete test cases for UAT (User Acceptance Testing).

PLAN:
%s

For each task and acceptance criterion, create ONE test case with:
- A clear title
- The exact expected behaviour to verify

Format as a numbered list:
1. [Title]
   Expected: [what to verify — specific, measurable]

Generate 5-10 test cases covering the most important scenarios.`, truncate(string(data), 6000))

	var response strings.Builder
	result, err := chatClient.ChatStream(ctx, prompt, "", true, true, func(delta string) {
		response.WriteString(delta)
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	_ = result

	tests := parseTestCases(response.String())
	sid := fmt.Sprintf("qa-%d", time.Now().Unix())
	state := &qaState{
		ID:       sid,
		PlanFile: planFile,
		Status:   "new",
		Started:  time.Now(),
		Updated:  time.Now(),
		Tests:    tests,
	}
	saveQAState(state)
	return state
}

// qaFromFeature generates test cases from a feature description.
func qaFromFeature(ctx context.Context, chatClient *httpclient.ChatClient, feature string) *qaState {
	fmt.Printf("Generating test cases for: %s\n\n", feature)
	prompt := fmt.Sprintf(`Generate UAT test cases for this feature: "%s"

For each key scenario, create a test case:
1. [Title]
   Expected: [exact behaviour to verify]

Generate 5-8 test cases.`, feature)

	var response strings.Builder
	chatClient.ChatStream(ctx, prompt, "", true, true, func(delta string) {
		response.WriteString(delta)
	})

	tests := parseTestCases(response.String())
	sid := fmt.Sprintf("qa-%d", time.Now().Unix())
	state := &qaState{
		ID:      sid,
		Status:  "new",
		Started: time.Now(),
		Updated: time.Now(),
		Tests:   tests,
	}
	saveQAState(state)
	return state
}

// runQASession drives the interactive test loop.
func runQASession(ctx context.Context, state *qaState, chatClient *httpclient.ChatClient, debugClient *httpclient.DebugClient) {
	state.Status = "in_progress"
	total := len(state.Tests)

	fmt.Printf("\n╔══════════════════════════════════════════╗\n")
	fmt.Printf("║  coder qa  ·  %d test cases               ║\n", total)
	fmt.Printf("║  Type \"pass\", describe issue, or \"skip\"   ║\n")
	fmt.Printf("╚══════════════════════════════════════════╝\n\n")

	scanner := bufio.NewScanner(os.Stdin)

	for i := range state.Tests {
		t := &state.Tests[i]
		if t.Result != "pending" && t.Result != "" {
			continue // already done
		}

		fmt.Printf("  ┌─────────────────────────────────────────────────────┐\n")
		fmt.Printf("  │  TEST %d/%d: %-42s│\n", i+1, total, truncate(t.Title, 42))
		fmt.Printf("  │                                                     │\n")
		fmt.Printf("  │  Expected:                                          │\n")
		for _, line := range wrapText(t.Expected, 51, "  │  ") {
			fmt.Printf("%-57s│\n", line)
		}
		fmt.Printf("  │                                                     │\n")
		fmt.Printf("  │  → Type \"pass\", describe the issue, or \"skip\"       │\n")
		fmt.Printf("  └─────────────────────────────────────────────────────┘\n\n")

		fmt.Print("  Result › ")
		if !scanner.Scan() {
			break
		}
		answer := strings.TrimSpace(scanner.Text())

		switch strings.ToLower(answer) {
		case "pass", "p", "ok", "":
			t.Result = "pass"
			fmt.Printf("  ✓ PASS\n\n")
		case "skip", "s":
			t.Result = "skip"
			fmt.Printf("  ⊘ SKIPPED\n\n")
		default:
			t.Result = "issue"
			t.Reported = answer
			t.Severity = inferSeverity(answer)
			fmt.Printf("  ✗ ISSUE logged (%s)\n\n", strings.ToUpper(t.Severity))

			// Auto-diagnose
			fmt.Print("  ⟳ Diagnosing root cause...")
			dr, err := debugClient.Debug(ctx, answer, "", "", true, true)
			if err == nil && dr.RootCause != "" {
				t.RootCause = dr.RootCause
				fmt.Printf("\r  → Root cause: %s\n\n", truncate(dr.RootCause, 70))
			} else {
				fmt.Println()
			}
		}

		state.Tests[i] = *t
		state.CurrentTest = i + 1
		state.Updated = time.Now()
		saveQAState(state)
	}

	state.Status = "complete"
	state.Updated = time.Now()
	saveQAState(state)

	printQAReport(state, "")
}

func printQAReport(state *qaState, outFile string) {
	if state == nil {
		return
	}
	var sb strings.Builder

	passed, issues, skipped := 0, 0, 0
	for _, t := range state.Tests {
		switch t.Result {
		case "pass":
			passed++
		case "issue":
			issues++
		case "skip":
			skipped++
		}
	}

	sb.WriteString(fmt.Sprintf("\n%s\n", strings.Repeat("═", 46)))
	sb.WriteString(fmt.Sprintf("  QA COMPLETE — %d passed, %d issues, %d skipped\n", passed, issues, skipped))
	sb.WriteString(fmt.Sprintf("%s\n\n", strings.Repeat("═", 46)))

	if issues > 0 {
		sb.WriteString("ISSUES:\n")
		for _, t := range state.Tests {
			if t.Result != "issue" {
				continue
			}
			sb.WriteString(fmt.Sprintf("  ● [%s] %s\n", strings.ToUpper(t.Severity), t.Title))
			sb.WriteString(fmt.Sprintf("    Reported: %q\n", t.Reported))
			if t.RootCause != "" {
				sb.WriteString(fmt.Sprintf("    Root cause: %s\n", truncate(t.RootCause, 70)))
			}
			sb.WriteString("\n")
		}
	}

	report := sb.String()
	fmt.Print(report)

	if outFile != "" {
		os.WriteFile(outFile, []byte(report), 0644)
		fmt.Printf("Report saved: %s\n", outFile)
	}
}

// --- helpers ---

func inferSeverity(description string) string {
	desc := strings.ToLower(description)
	criticalWords := []string{"crash", "panic", "data loss", "security", "broken", "not work", "cannot", "fail", "error", "exception"}
	for _, w := range criticalWords {
		if strings.Contains(desc, w) {
			return "major"
		}
	}
	minorWords := []string{"wrong", "incorrect", "missing", "bad", "issue", "problem"}
	for _, w := range minorWords {
		if strings.Contains(desc, w) {
			return "minor"
		}
	}
	return "minor"
}

var testCaseRe = regexp.MustCompile(`(?m)^\s*\d+\.\s+(.+)`)
var expectedRe = regexp.MustCompile(`(?i)Expected:\s*(.+)`)

func parseTestCases(text string) []qaTest {
	var tests []qaTest
	lines := strings.Split(text, "\n")

	var current *qaTest
	num := 1

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// New test case: starts with a number
		if matched := testCaseRe.FindStringSubmatch("   " + line); len(matched) > 1 {
			if current != nil {
				tests = append(tests, *current)
			}
			current = &qaTest{
				Number:   num,
				Title:    strings.TrimSpace(matched[1]),
				Result:   "pending",
			}
			num++
			continue
		}

		// Expected line
		if current != nil {
			if matched := expectedRe.FindStringSubmatch(line); len(matched) > 1 {
				current.Expected = strings.TrimSpace(matched[1])
			} else if current.Expected == "" && !strings.HasPrefix(line, "#") {
				// Continuation of title or first detail
				if current.Expected == "" {
					current.Expected = line
				}
			}
		}
	}

	if current != nil {
		tests = append(tests, *current)
	}

	// Fallback: if no tests parsed, create generic ones
	if len(tests) == 0 {
		tests = []qaTest{
			{Number: 1, Title: "Feature works as expected", Expected: "The feature behaves as described", Result: "pending"},
		}
	}

	return tests
}

// --- state persistence in .coder/qa/ ---

func qaStatePath(id string) string {
	return filepath.Join(".coder", "qa", id+".md")
}

func saveQAState(s *qaState) {
	os.MkdirAll(filepath.Join(".coder", "qa"), 0755)
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("---\nid: %s\nplan: %s\nstatus: %s\nstarted: %s\nupdated: %s\n---\n\n",
		s.ID, s.PlanFile, s.Status,
		s.Started.Format(time.RFC3339), s.Updated.Format(time.RFC3339),
	))
	sb.WriteString(fmt.Sprintf("## Progress\ntotal: %d · passed: %d · issues: %d · skipped: %d · current: %d\n\n",
		len(s.Tests), countResult(s.Tests, "pass"), countResult(s.Tests, "issue"),
		countResult(s.Tests, "skip"), s.CurrentTest,
	))
	sb.WriteString("## Tests\n\n")
	for _, t := range s.Tests {
		sb.WriteString(fmt.Sprintf("### %d. %s\nexpected: %s\nresult: %s\n", t.Number, t.Title, t.Expected, t.Result))
		if t.Reported != "" {
			sb.WriteString(fmt.Sprintf("reported: %q\nseverity: %s\n", t.Reported, t.Severity))
		}
		if t.RootCause != "" {
			sb.WriteString(fmt.Sprintf("root_cause: %q\n", t.RootCause))
		}
		sb.WriteString("\n")
	}
	os.WriteFile(qaStatePath(s.ID), []byte(sb.String()), 0644)
}

func loadQAState(id string) *qaState {
	data, err := os.ReadFile(qaStatePath(id))
	if err != nil {
		return nil
	}
	// Simple parse — just rebuild a state from the markdown
	s := &qaState{ID: id, Status: "in_progress"}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "plan: "):
			s.PlanFile = strings.TrimPrefix(line, "plan: ")
		case strings.HasPrefix(line, "status: "):
			s.Status = strings.TrimPrefix(line, "status: ")
		}
	}
	// Re-parse tests
	var current *qaTest
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "### ") {
			if current != nil {
				s.Tests = append(s.Tests, *current)
			}
			parts := strings.SplitN(strings.TrimPrefix(line, "### "), ". ", 2)
			current = &qaTest{Result: "pending"}
			if len(parts) > 1 {
				current.Title = parts[1]
			}
		} else if current != nil {
			switch {
			case strings.HasPrefix(line, "expected: "):
				current.Expected = strings.TrimPrefix(line, "expected: ")
			case strings.HasPrefix(line, "result: "):
				current.Result = strings.TrimPrefix(line, "result: ")
			case strings.HasPrefix(line, "severity: "):
				current.Severity = strings.TrimPrefix(line, "severity: ")
			case strings.HasPrefix(line, "reported: "):
				current.Reported = strings.Trim(strings.TrimPrefix(line, "reported: "), `"`)
			case strings.HasPrefix(line, "root_cause: "):
				current.RootCause = strings.Trim(strings.TrimPrefix(line, "root_cause: "), `"`)
			}
		}
	}
	if current != nil {
		s.Tests = append(s.Tests, *current)
	}
	return s
}

func findLastQASession() string {
	dir := filepath.Join(".coder", "qa")
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) == 0 {
		return ""
	}
	// Return the most recent .md file
	var last string
	var lastTime time.Time
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		info, _ := e.Info()
		if info.ModTime().After(lastTime) {
			lastTime = info.ModTime()
			last = strings.TrimSuffix(e.Name(), ".md")
		}
	}
	return last
}

func listQASessions() {
	dir := filepath.Join(".coder", "qa")
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) == 0 {
		fmt.Println("No QA sessions found.")
		return
	}
	fmt.Printf("\n  %-20s  %-10s  %s\n", "SESSION ID", "STATUS", "MODIFIED")
	fmt.Printf("  %s\n", strings.Repeat("─", 50))
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		info, _ := e.Info()
		id := strings.TrimSuffix(e.Name(), ".md")
		fmt.Printf("  %-20s  %-10s  %s\n", id, "—", info.ModTime().Format("2006-01-02 15:04"))
	}
	fmt.Println()
}

func countResult(tests []qaTest, result string) int {
	n := 0
	for _, t := range tests {
		if t.Result == result {
			n++
		}
	}
	return n
}
