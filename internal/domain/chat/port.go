package chat

import "context"

// Repository handles session and message persistence.
type Repository interface {
	CreateSession(ctx context.Context, s *Session) error
	GetSession(ctx context.Context, id string) (*Session, error)
	UpdateSession(ctx context.Context, s *Session) error
	ListSessions(ctx context.Context, clientID string, limit int) ([]Session, error)
	DeleteSession(ctx context.Context, id string) error

	AppendMessage(ctx context.Context, m *Message) error
	GetMessages(ctx context.Context, sessionID string, limit int) ([]Message, error)
}

// LLMProvider is the port for LLM inference (implemented by Ollama, OpenAI-compat, etc.).
type LLMProvider interface {
	// Chat performs a blocking (non-streaming) completion.
	Chat(ctx context.Context, model string, messages []LLMMessage) (*LLMResponse, error)
	// ChatStream streams response deltas; onDelta is called for each token chunk.
	ChatStream(ctx context.Context, model string, messages []LLMMessage, onDelta func(string)) (*LLMResponse, error)
}

// Manager is the application-level chat use-case interface.
type Manager interface {
	// Chat runs context injection + LLM completion and returns the full reply.
	Chat(ctx context.Context, clientID string, req ChatRequest) (*ChatResponse, error)
	// ChatStream runs context injection + LLM streaming; onDelta receives token chunks.
	ChatStream(ctx context.Context, clientID string, req ChatRequest, onDelta func(string)) (*ChatResponse, error)
	// ListSessions returns recent sessions for a client.
	ListSessions(ctx context.Context, clientID string, limit int) ([]Session, error)
	// GetSession returns full message history for a session.
	GetSession(ctx context.Context, id string) (*Session, []Message, error)
	// DeleteSession removes a session and all its messages.
	DeleteSession(ctx context.Context, id string) error
}
