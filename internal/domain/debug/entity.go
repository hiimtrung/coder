package debug

// DebugRequest is the input to the debug use case.
type DebugRequest struct {
	ErrorMessage string
	FileContext  string // optional source file content
	DiffContext  string // optional git diff
	Context      DebugContext
}

// DebugContext configures context injection.
type DebugContext struct {
	InjectMemory bool
	InjectSkills bool
}

// DebugResult is the structured output of a debug analysis.
type DebugResult struct {
	RootCause    string      `json:"root_cause"`
	Location     string      `json:"location,omitempty"`
	Confidence   string      `json:"confidence"` // HIGH | MEDIUM | LOW
	SuggestedFix string      `json:"suggested_fix"`
	SimilarIssues []string   `json:"similar_issues,omitempty"`
	FollowUp     string      `json:"follow_up,omitempty"`
	ContextUsed  ContextUsed `json:"context_used"`
	Model        string      `json:"model"`
}

// ContextUsed records injected context entries.
type ContextUsed struct {
	MemoryHits []string `json:"memory_hits"`
	SkillHits  []string `json:"skill_hits"`
}
