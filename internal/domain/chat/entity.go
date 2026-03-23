package chat

import "time"

// Session represents a conversation session between a client and the LLM.
type Session struct {
	ID        string    `json:"id"`
	ClientID  string    `json:"client_id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Message is a single turn in a conversation session.
type Message struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id"`
	Role      string    `json:"role"` // "user" | "assistant" | "system"
	Content   string    `json:"content"`
	TokensIn  int       `json:"tokens_in"`
	TokensOut int       `json:"tokens_out"`
	CreatedAt time.Time `json:"created_at"`
}

// ChatRequest is the input to the chat use case.
type ChatRequest struct {
	Message   string
	SessionID string      // optional — resume an existing session
	Context   ChatContext // optional overrides
}

// ChatContext configures context injection behaviour for a single request.
type ChatContext struct {
	InjectMemory bool
	InjectSkills bool
	MemoryLimit  int
	SkillLimit   int
	ExtraSystem  string // appended verbatim to the system prompt
}

// ChatResponse is returned after the LLM completes a full (non-streaming) response.
type ChatResponse struct {
	Reply       string
	SessionID   string
	ContextUsed ContextUsed
	Model       string
	Tokens      TokenUsage
}

// ContextUsed records which memory/skill entries were injected into the prompt.
type ContextUsed struct {
	MemoryHits []string `json:"memory_hits"`
	SkillHits  []string `json:"skill_hits"`
}

// TokenUsage holds prompt and completion token counts.
type TokenUsage struct {
	Prompt     int `json:"prompt"`
	Completion int `json:"completion"`
}

// LLMMessage is a single message sent to the LLM (maps to Ollama message format).
type LLMMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// LLMResponse is the structured response from the LLM provider.
type LLMResponse struct {
	Content   string
	TokensIn  int
	TokensOut int
}
