package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	chatdomain "github.com/trungtran/coder/internal/domain/chat"
)

type postgresChatRepo struct {
	db *sql.DB
}

// NewPostgresChatRepo creates the chat repository and runs schema migrations.
func NewPostgresChatRepo(db *sql.DB) (chatdomain.Repository, error) {
	r := &postgresChatRepo{db: db}
	if err := r.init(); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *postgresChatRepo) init() error {
	_, err := r.db.Exec(`
	CREATE TABLE IF NOT EXISTS coder_sessions (
		id         TEXT PRIMARY KEY,
		client_id  TEXT NOT NULL REFERENCES coder_clients(id) ON DELETE CASCADE,
		title      TEXT NOT NULL DEFAULT '',
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_sessions_client ON coder_sessions(client_id, updated_at DESC);

	CREATE TABLE IF NOT EXISTS coder_messages (
		id         TEXT PRIMARY KEY,
		session_id TEXT NOT NULL REFERENCES coder_sessions(id) ON DELETE CASCADE,
		role       TEXT NOT NULL CHECK (role IN ('user', 'assistant', 'system')),
		content    TEXT NOT NULL,
		tokens_in  INT  DEFAULT 0,
		tokens_out INT  DEFAULT 0,
		created_at TIMESTAMP NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_messages_session ON coder_messages(session_id, created_at ASC);
	`)
	return err
}

// --- Session methods ---

func (r *postgresChatRepo) CreateSession(ctx context.Context, s *chatdomain.Session) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO coder_sessions (id, client_id, title, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		s.ID, s.ClientID, s.Title, s.CreatedAt, s.UpdatedAt,
	)
	return err
}

func (r *postgresChatRepo) GetSession(ctx context.Context, id string) (*chatdomain.Session, error) {
	var s chatdomain.Session
	err := r.db.QueryRowContext(ctx,
		`SELECT id, client_id, title, created_at, updated_at FROM coder_sessions WHERE id = $1`, id,
	).Scan(&s.ID, &s.ClientID, &s.Title, &s.CreatedAt, &s.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("session not found: %s", id)
	}
	return &s, err
}

func (r *postgresChatRepo) UpdateSession(ctx context.Context, s *chatdomain.Session) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE coder_sessions SET title = $1, updated_at = $2 WHERE id = $3`,
		s.Title, s.UpdatedAt, s.ID,
	)
	return err
}

func (r *postgresChatRepo) ListSessions(ctx context.Context, clientID string, limit int) ([]chatdomain.Session, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, client_id, title, created_at, updated_at
		 FROM coder_sessions WHERE client_id = $1
		 ORDER BY updated_at DESC LIMIT $2`,
		clientID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []chatdomain.Session
	for rows.Next() {
		var s chatdomain.Session
		if err := rows.Scan(&s.ID, &s.ClientID, &s.Title, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	return sessions, nil
}

func (r *postgresChatRepo) DeleteSession(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM coder_sessions WHERE id = $1`, id)
	return err
}

// --- Message methods ---

func (r *postgresChatRepo) AppendMessage(ctx context.Context, m *chatdomain.Message) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO coder_messages (id, session_id, role, content, tokens_in, tokens_out, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		m.ID, m.SessionID, m.Role, m.Content, m.TokensIn, m.TokensOut, m.CreatedAt,
	)
	return err
}

func (r *postgresChatRepo) GetMessages(ctx context.Context, sessionID string, limit int) ([]chatdomain.Message, error) {
	if limit <= 0 {
		limit = 20
	}
	// Fetch last N messages ordered oldest-first for LLM context
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, session_id, role, content, tokens_in, tokens_out, created_at
		 FROM (
			SELECT * FROM coder_messages WHERE session_id = $1
			ORDER BY created_at DESC LIMIT $2
		 ) sub ORDER BY created_at ASC`,
		sessionID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []chatdomain.Message
	for rows.Next() {
		var m chatdomain.Message
		if err := rows.Scan(&m.ID, &m.SessionID, &m.Role, &m.Content, &m.TokensIn, &m.TokensOut, &m.CreatedAt); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, nil
}

// getFirstUserMessage returns the first user message in a session (for auto-title).
func (r *postgresChatRepo) getFirstUserMessage(ctx context.Context, sessionID string) string {
	var content string
	r.db.QueryRowContext(ctx,
		`SELECT content FROM coder_messages WHERE session_id = $1 AND role = 'user' ORDER BY created_at ASC LIMIT 1`,
		sessionID,
	).Scan(&content)
	return content
}

// AutoTitleSession sets the session title from the first user message if not already set.
func AutoTitleSession(ctx context.Context, repo chatdomain.Repository, sessionID string) {
	s, err := repo.GetSession(ctx, sessionID)
	if err != nil || s.Title != "" {
		return
	}
	// We need the concrete type to call helper methods — use a type assertion
	type firstMsgGetter interface {
		getFirstUserMessage(ctx context.Context, sessionID string) string
	}
	if getter, ok := repo.(firstMsgGetter); ok {
		msg := getter.getFirstUserMessage(ctx, sessionID)
		if msg == "" {
			return
		}
		// Truncate to 60 chars for the title
		title := msg
		if len(title) > 60 {
			title = title[:57] + "..."
		}
		s.Title = title
		s.UpdatedAt = time.Now()
		repo.UpdateSession(ctx, s)
	}
}
