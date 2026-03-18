package postgres

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	authdomain "github.com/trungtran/coder/internal/domain/auth"
)

type postgresAuth struct {
	db *sql.DB
}

// NewPostgresAuth creates the auth repository and runs migrations.
func NewPostgresAuth(db *sql.DB) (authdomain.AuthRepository, error) {
	a := &postgresAuth{db: db}
	if err := a.init(); err != nil {
		return nil, err
	}
	return a, nil
}

func (a *postgresAuth) init() error {
	_, err := a.db.Exec(`
	CREATE TABLE IF NOT EXISTS coder_server_config (
		key   TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS coder_clients (
		id                TEXT PRIMARY KEY,
		git_name          TEXT NOT NULL,
		git_email         TEXT NOT NULL,
		access_token_hash TEXT NOT NULL UNIQUE,
		created_at        TIMESTAMP NOT NULL,
		last_seen_at      TIMESTAMP NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_coder_clients_token ON coder_clients(access_token_hash);
	CREATE TABLE IF NOT EXISTS coder_client_activity (
		id        TEXT PRIMARY KEY,
		client_id TEXT NOT NULL REFERENCES coder_clients(id) ON DELETE CASCADE,
		command   TEXT NOT NULL,
		repo      TEXT,
		branch    TEXT,
		timestamp TIMESTAMP NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_coder_activity_client ON coder_client_activity(client_id);
	CREATE INDEX IF NOT EXISTS idx_coder_activity_ts     ON coder_client_activity(timestamp DESC);
	`)
	return err
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func (a *postgresAuth) GetBootstrapTokenHash(ctx context.Context) (string, error) {
	var v string
	err := a.db.QueryRowContext(ctx, `SELECT value FROM coder_server_config WHERE key = 'bootstrap_token_hash'`).Scan(&v)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return v, err
}

func (a *postgresAuth) SetBootstrapTokenHash(ctx context.Context, tokenHash string) error {
	_, err := a.db.ExecContext(ctx,
		`INSERT INTO coder_server_config (key, value) VALUES ('bootstrap_token_hash', $1)
		 ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, tokenHash)
	return err
}

func (a *postgresAuth) RegisterClient(ctx context.Context, c *authdomain.Client) error {
	_, err := a.db.ExecContext(ctx,
		`INSERT INTO coder_clients (id, git_name, git_email, access_token_hash, created_at, last_seen_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		c.ID, c.GitName, c.GitEmail, hashToken(c.AccessToken), c.CreatedAt, c.LastSeenAt,
	)
	return err
}

func (a *postgresAuth) GetClientByTokenHash(ctx context.Context, tokenHash string) (*authdomain.Client, error) {
	var c authdomain.Client
	err := a.db.QueryRowContext(ctx,
		`SELECT id, git_name, git_email, created_at, last_seen_at FROM coder_clients WHERE access_token_hash = $1`,
		tokenHash,
	).Scan(&c.ID, &c.GitName, &c.GitEmail, &c.CreatedAt, &c.LastSeenAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("invalid access token")
	}
	return &c, err
}

func (a *postgresAuth) ListClients(ctx context.Context) ([]authdomain.Client, error) {
	rows, err := a.db.QueryContext(ctx,
		`SELECT id, git_name, git_email, created_at, last_seen_at FROM coder_clients ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var clients []authdomain.Client
	for rows.Next() {
		var c authdomain.Client
		if err := rows.Scan(&c.ID, &c.GitName, &c.GitEmail, &c.CreatedAt, &c.LastSeenAt); err != nil {
			return nil, err
		}
		clients = append(clients, c)
	}
	return clients, nil
}

func (a *postgresAuth) UpdateLastSeen(ctx context.Context, clientID string, t time.Time) error {
	_, err := a.db.ExecContext(ctx, `UPDATE coder_clients SET last_seen_at = $1 WHERE id = $2`, t, clientID)
	return err
}

func (a *postgresAuth) DeleteClient(ctx context.Context, clientID string) error {
	_, err := a.db.ExecContext(ctx, `DELETE FROM coder_clients WHERE id = $1`, clientID)
	return err
}

func (a *postgresAuth) LogActivity(ctx context.Context, act *authdomain.Activity) error {
	_, err := a.db.ExecContext(ctx,
		`INSERT INTO coder_client_activity (id, client_id, command, repo, branch, timestamp)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		act.ID, act.ClientID, act.Command, act.Repo, act.Branch, act.Timestamp,
	)
	return err
}

func (a *postgresAuth) GetActivities(ctx context.Context, clientID string, limit int) ([]authdomain.Activity, error) {
	rows, err := a.db.QueryContext(ctx,
		`SELECT id, client_id, command, repo, branch, timestamp
		 FROM coder_client_activity WHERE client_id = $1 ORDER BY timestamp DESC LIMIT $2`,
		clientID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var acts []authdomain.Activity
	for rows.Next() {
		var act authdomain.Activity
		if err := rows.Scan(&act.ID, &act.ClientID, &act.Command, &act.Repo, &act.Branch, &act.Timestamp); err != nil {
			return nil, err
		}
		acts = append(acts, act)
	}
	return acts, nil
}
