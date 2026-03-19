package auth

import (
	"context"
	"time"
)

// AuthRepository is the persistence port for auth data.
type AuthRepository interface {
	// Bootstrap token (used once during client registration)
	GetBootstrapTokenHash(ctx context.Context) (string, error)
	SetBootstrapTokenHash(ctx context.Context, tokenHash string) error
	DeleteBootstrapTokenHash(ctx context.Context) error

	// Client CRUD
	RegisterClient(ctx context.Context, c *Client) error
	GetClientByTokenHash(ctx context.Context, tokenHash string) (*Client, error)
	ListClients(ctx context.Context) ([]Client, error)
	UpdateLastSeen(ctx context.Context, clientID string, t time.Time) error
	DeleteClient(ctx context.Context, clientID string) error

	// Token management
	UpdateAccessTokenHash(ctx context.Context, clientID, newTokenHash string) error

	// Activity log
	LogActivity(ctx context.Context, a *Activity) error
	GetActivities(ctx context.Context, clientID string, limit int) ([]Activity, error)

	// Dashboard queries
	GetAllActivities(ctx context.Context, filter ActivityFilter) ([]Activity, int, error)
	GetActivityStats(ctx context.Context, days int) (ActivityStats, error)
}

// AuthManager is the application-level service for auth operations.
// When IsSecureMode() == false, every method is a no-op / always succeeds.
type AuthManager interface {
	IsSecureMode() bool

	// RegisterClient validates bootstrapToken and creates a new client.
	// Returns the raw access token (shown once to the user).
	RegisterClient(ctx context.Context, bootstrapToken, gitName, gitEmail string) (rawAccessToken string, err error)

	// ValidateToken returns the client for a raw access token or error if invalid.
	ValidateToken(ctx context.Context, rawToken string) (*Client, error)

	// LogActivity records a command execution (no-op if not secure mode).
	LogActivity(ctx context.Context, rawToken, command, repo, branch string) error

	// ListClients returns all registered clients.
	ListClients(ctx context.Context) ([]Client, error)

	// GetBootstrapToken returns the current bootstrap token (raw, for display at startup).
	// Returns empty string if already shown.
	GetBootstrapToken(ctx context.Context) (string, error)

	// RegenerateBootstrapToken invalidates the existing bootstrap token hash,
	// generates a new one, persists the hash, and returns the raw token once.
	RegenerateBootstrapToken(ctx context.Context) (string, error)

	// HasBootstrapToken returns true if a bootstrap token hash is currently stored.
	HasBootstrapToken(ctx context.Context) (bool, error)

	// RevokeClient removes a client and all its associated data.
	RevokeClient(ctx context.Context, clientID string) error

	// RotateToken generates a new access token for the calling client,
	// atomically replacing the old one. Returns the new raw token (shown once).
	RotateToken(ctx context.Context, clientID string) (rawToken string, err error)

	// GetAllActivities returns paginated activities with client email info.
	GetAllActivities(ctx context.Context, filter ActivityFilter) ([]Activity, int, error)

	// GetActivityStats returns aggregated stats for the dashboard overview.
	GetActivityStats(ctx context.Context, days int) (ActivityStats, error)
}
