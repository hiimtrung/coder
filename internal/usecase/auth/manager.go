package ucauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	authdomain "github.com/trungtran/coder/internal/domain/auth"
)

// Manager implements authdomain.AuthManager.
type Manager struct {
	repo       authdomain.AuthRepository
	secureMode bool
}

func NewManager(repo authdomain.AuthRepository, secureMode bool) *Manager {
	return &Manager{repo: repo, secureMode: secureMode}
}

func (m *Manager) IsSecureMode() bool { return m.secureMode }

// GetBootstrapToken loads the existing bootstrap token hash from DB.
// If none exists, generates a new one, stores the hash, returns the raw token.
// After first generation it returns empty (already shown).
func (m *Manager) GetBootstrapToken(ctx context.Context) (string, error) {
	if !m.secureMode || m.repo == nil {
		return "", nil
	}
	existing, err := m.repo.GetBootstrapTokenHash(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to check bootstrap token: %w", err)
	}
	if existing != "" {
		// Already set — return empty so it's not re-displayed
		return "", nil
	}
	// First startup: generate and store
	raw, err := generateToken()
	if err != nil {
		return "", err
	}
	hash := sha256Hex(raw)
	if err := m.repo.SetBootstrapTokenHash(ctx, hash); err != nil {
		return "", fmt.Errorf("failed to store bootstrap token: %w", err)
	}
	return raw, nil
}

func (m *Manager) RegisterClient(ctx context.Context, bootstrapToken, gitName, gitEmail string) (string, error) {
	if !m.secureMode {
		return "", fmt.Errorf("server is not in secure mode")
	}
	if m.repo == nil {
		return "", fmt.Errorf("auth repository not configured")
	}
	// Validate bootstrap token
	expectedHash, err := m.repo.GetBootstrapTokenHash(ctx)
	if err != nil || expectedHash == "" {
		return "", fmt.Errorf("server bootstrap token not configured")
	}
	if sha256Hex(bootstrapToken) != expectedHash {
		return "", fmt.Errorf("invalid bootstrap token")
	}
	// Generate access token
	rawToken, err := generateToken()
	if err != nil {
		return "", err
	}
	now := time.Now()
	c := &authdomain.Client{
		ID:          uuid.New().String(),
		GitName:     gitName,
		GitEmail:    gitEmail,
		AccessToken: rawToken, // repo will hash this
		CreatedAt:   now,
		LastSeenAt:  now,
	}
	if err := m.repo.RegisterClient(ctx, c); err != nil {
		return "", fmt.Errorf("failed to register client: %w", err)
	}
	return rawToken, nil
}

func (m *Manager) ValidateToken(ctx context.Context, rawToken string) (*authdomain.Client, error) {
	if !m.secureMode {
		// Open mode: return a dummy client so middleware passes
		return &authdomain.Client{ID: "anonymous", GitName: "anonymous", GitEmail: ""}, nil
	}
	if m.repo == nil {
		return nil, fmt.Errorf("auth repository not configured")
	}
	hash := sha256Hex(rawToken)
	c, err := m.repo.GetClientByTokenHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	// Update last seen in background (ignore error)
	go func() {
		_ = m.repo.UpdateLastSeen(context.Background(), c.ID, time.Now())
	}()
	return c, nil
}

func (m *Manager) LogActivity(ctx context.Context, rawToken, command, repo, branch string) error {
	if !m.secureMode || m.repo == nil {
		return nil
	}
	hash := sha256Hex(rawToken)
	c, err := m.repo.GetClientByTokenHash(ctx, hash)
	if err != nil {
		return nil // silently ignore invalid token for activity log
	}
	act := &authdomain.Activity{
		ID:        uuid.New().String(),
		ClientID:  c.ID,
		Command:   command,
		Repo:      repo,
		Branch:    branch,
		Timestamp: time.Now(),
	}
	return m.repo.LogActivity(ctx, act)
}

// RegenerateBootstrapToken invalidates the current bootstrap token hash,
// generates a brand-new token, stores the hash, and returns the raw token.
// Use this when the original token was missed or needs to be rotated.
func (m *Manager) RegenerateBootstrapToken(ctx context.Context) (string, error) {
	if !m.secureMode || m.repo == nil {
		return "", fmt.Errorf("server is not in secure mode")
	}
	// Delete the old hash first
	if err := m.repo.DeleteBootstrapTokenHash(ctx); err != nil {
		return "", fmt.Errorf("failed to clear old bootstrap token: %w", err)
	}
	// Generate and store a new one
	raw, err := generateToken()
	if err != nil {
		return "", err
	}
	if err := m.repo.SetBootstrapTokenHash(ctx, sha256Hex(raw)); err != nil {
		return "", fmt.Errorf("failed to store new bootstrap token: %w", err)
	}
	return raw, nil
}

// HasBootstrapToken reports whether a bootstrap token hash is currently stored.
func (m *Manager) HasBootstrapToken(ctx context.Context) (bool, error) {
	if !m.secureMode || m.repo == nil {
		return false, nil
	}
	hash, err := m.repo.GetBootstrapTokenHash(ctx)
	if err != nil {
		return false, err
	}
	return hash != "", nil
}

func (m *Manager) ListClients(ctx context.Context) ([]authdomain.Client, error) {
	if m.repo == nil {
		return []authdomain.Client{}, nil
	}
	return m.repo.ListClients(ctx)
}

// RevokeClient removes a client by ID.
func (m *Manager) RevokeClient(ctx context.Context, clientID string) error {
	if m.repo == nil {
		return fmt.Errorf("auth repository not configured")
	}
	return m.repo.DeleteClient(ctx, clientID)
}

// RotateToken generates a new access token for the given client,
// atomically replacing the old hash in the database.
// The new raw token is returned once — it is never persisted in plaintext.
func (m *Manager) RotateToken(ctx context.Context, clientID string) (string, error) {
	if !m.secureMode {
		return "", fmt.Errorf("server is not in secure mode")
	}
	if m.repo == nil {
		return "", fmt.Errorf("auth repository not configured")
	}
	rawToken, err := generateToken()
	if err != nil {
		return "", err
	}
	if err := m.repo.UpdateAccessTokenHash(ctx, clientID, sha256Hex(rawToken)); err != nil {
		return "", fmt.Errorf("failed to rotate token: %w", err)
	}
	return rawToken, nil
}

// GetAllActivities returns paginated activity log with client email info.
func (m *Manager) GetAllActivities(ctx context.Context, filter authdomain.ActivityFilter) ([]authdomain.Activity, int, error) {
	if m.repo == nil {
		return nil, 0, nil
	}
	return m.repo.GetAllActivities(ctx, filter)
}

// GetActivityStats returns aggregated dashboard statistics.
func (m *Manager) GetActivityStats(ctx context.Context, days int) (authdomain.ActivityStats, error) {
	if m.repo == nil {
		return authdomain.ActivityStats{}, nil
	}
	return m.repo.GetActivityStats(ctx, days)
}

// generateToken creates a cryptographically secure random 32-byte hex token.
func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}
