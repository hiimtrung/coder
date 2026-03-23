package review

// ReviewRequest is the input to the review use case.
type ReviewRequest struct {
	Type    string // "diff" | "file" | "pr"
	Content string // raw diff or file content
	Focus   string // optional: "security", "performance", etc.
	Context ReviewContext
}

// ReviewContext configures context injection.
type ReviewContext struct {
	InjectMemory bool
	InjectSkills bool
}

// ReviewResult is the structured output of a code review.
type ReviewResult struct {
	Summary     string          `json:"summary"`
	Strengths   []string        `json:"strengths"`
	Concerns    []ReviewConcern `json:"concerns"`
	Suggestions []string        `json:"suggestions"`
	Stats       ReviewStats     `json:"stats"`
	ContextUsed ContextUsed     `json:"context_used"`
	Model       string          `json:"model"`
}

// ReviewConcern is a single issue found during review.
type ReviewConcern struct {
	Severity    string `json:"severity"`    // HIGH | MEDIUM | LOW
	Description string `json:"description"`
	Location    string `json:"location,omitempty"`
	Suggestion  string `json:"suggestion,omitempty"`
}

// ReviewStats summarises concerns by severity.
type ReviewStats struct {
	FilesReviewed  int `json:"files_reviewed"`
	ConcernsHigh   int `json:"concerns_high"`
	ConcernsMedium int `json:"concerns_medium"`
	ConcernsLow    int `json:"concerns_low"`
}

// ContextUsed shows which memory/skill entries were injected.
type ContextUsed struct {
	MemoryHits []string `json:"memory_hits"`
	SkillHits  []string `json:"skill_hits"`
}
