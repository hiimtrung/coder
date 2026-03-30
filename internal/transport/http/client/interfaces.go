package httpclient

import (
	"context"

	debugdomain "github.com/trungtran/coder/internal/domain/debug"
	reviewdomain "github.com/trungtran/coder/internal/domain/review"
)

// ChatClientIface is the transport-agnostic interface for chat operations.
// Both the HTTP ChatClient and the gRPC ChatClient implement this.
type ChatClientIface interface {
	// ChatStream runs a streaming completion. onDelta is called for each token chunk.
	// Returns final metadata (session_id, context_used, tokens) when the stream ends.
	ChatStream(ctx context.Context, message, sessionID string, injectMemory, injectSkills bool, onDelta func(string)) (*ChatStreamDelta, error)

	// Chat runs a blocking (non-streaming) completion.
	Chat(ctx context.Context, message, sessionID string, injectMemory, injectSkills bool) (*ChatResponse, error)

	// ListSessions returns recent sessions for the authenticated client.
	ListSessions(ctx context.Context) ([]ChatSession, error)

	// GetSession returns a session with its full message history.
	GetSession(ctx context.Context, id string) (*ChatSession, []ChatMessage, error)

	// DeleteSession removes a session.
	DeleteSession(ctx context.Context, id string) error
}

// DebugClientIface is the transport-agnostic interface for debug operations.
type DebugClientIface interface {
	Debug(ctx context.Context, errorMsg, fileContext, diffContext string, injectMemory, injectSkills bool) (*debugdomain.DebugResult, error)
}

// ReviewClientIface is the transport-agnostic interface for review operations.
type ReviewClientIface interface {
	Review(ctx context.Context, reviewType, content, focus string, injectMemory, injectSkills bool) (*reviewdomain.ReviewResult, error)
}
