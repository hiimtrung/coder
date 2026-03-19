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

func (a *postgresAuth) DeleteBootstrapTokenHash(ctx context.Context) error {
	_, err := a.db.ExecContext(ctx,
		`DELETE FROM coder_server_config WHERE key = 'bootstrap_token_hash'`)
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

func (a *postgresAuth) UpdateAccessTokenHash(ctx context.Context, clientID, newTokenHash string) error {
	result, err := a.db.ExecContext(ctx,
		`UPDATE coder_clients SET access_token_hash = $1 WHERE id = $2`,
		newTokenHash, clientID,
	)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("client not found: %s", clientID)
	}
	return nil
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

func (a *postgresAuth) GetActivityChartStats(ctx context.Context, filter authdomain.ActivityFilter, days int) (authdomain.ActivityChartStats, error) {
	var stats authdomain.ActivityChartStats

	// Build optional WHERE clause for client/command filters on top of the N-day window.
	args := []any{days}
	where := `timestamp >= NOW() - ($1 || ' days')::INTERVAL`

	if filter.ClientID != "" {
		args = append(args, filter.ClientID)
		where += fmt.Sprintf(` AND client_id = $%d`, len(args))
	}
	if filter.Command != "" {
		args = append(args, filter.Command)
		where += fmt.Sprintf(` AND command = $%d`, len(args))
	}

	// 1. Total matching rows
	if err := a.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM coder_client_activity WHERE `+where, args...,
	).Scan(&stats.Total); err != nil {
		return stats, fmt.Errorf("chart total: %w", err)
	}

	// 2. Commands per day
	dayRows, err := a.db.QueryContext(ctx,
		`SELECT TO_CHAR(DATE_TRUNC('day', timestamp), 'YYYY-MM-DD'), COUNT(*)
		 FROM coder_client_activity WHERE `+where+`
		 GROUP BY DATE_TRUNC('day', timestamp)
		 ORDER BY DATE_TRUNC('day', timestamp)`, args...,
	)
	if err != nil {
		return stats, fmt.Errorf("chart commands per day: %w", err)
	}
	defer dayRows.Close()
	for dayRows.Next() {
		var dc authdomain.DailyCount
		if err := dayRows.Scan(&dc.Date, &dc.Count); err != nil {
			return stats, err
		}
		stats.CommandsPerDay = append(stats.CommandsPerDay, dc)
	}
	dayRows.Close()

	// 3. Top commands by count
	cmdRows, err := a.db.QueryContext(ctx,
		`SELECT command, COUNT(*) AS cnt FROM coder_client_activity WHERE `+where+`
		 GROUP BY command ORDER BY cnt DESC LIMIT 8`, args...,
	)
	if err != nil {
		return stats, fmt.Errorf("chart top commands: %w", err)
	}
	defer cmdRows.Close()
	var totalCmds int
	for cmdRows.Next() {
		var cc authdomain.CommandCount
		if err := cmdRows.Scan(&cc.Command, &cc.Count); err != nil {
			return stats, err
		}
		totalCmds += cc.Count
		stats.TopCommands = append(stats.TopCommands, cc)
	}
	cmdRows.Close()
	if totalCmds > 0 {
		for i := range stats.TopCommands {
			stats.TopCommands[i].Percent = float64(stats.TopCommands[i].Count) / float64(totalCmds) * 100
		}
	}

	// 4. Activity by hour of day 0-23
	hourRows, err := a.db.QueryContext(ctx,
		`SELECT EXTRACT(HOUR FROM timestamp)::INT, COUNT(*)
		 FROM coder_client_activity WHERE `+where+`
		 GROUP BY EXTRACT(HOUR FROM timestamp) ORDER BY 1`, args...,
	)
	if err != nil {
		return stats, fmt.Errorf("chart activity by hour: %w", err)
	}
	defer hourRows.Close()
	for hourRows.Next() {
		var hc authdomain.HourCount
		if err := hourRows.Scan(&hc.Hour, &hc.Count); err != nil {
			return stats, err
		}
		stats.ActivityByHour = append(stats.ActivityByHour, hc)
	}

	return stats, nil
}

func (a *postgresAuth) GetAllActivities(ctx context.Context, filter authdomain.ActivityFilter) ([]authdomain.Activity, int, error) {
	// Count query
	var total int
	err := a.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM coder_client_activity a
		 JOIN coder_clients c ON c.id = a.client_id
		 WHERE ($1 = '' OR a.client_id = $1)
		   AND ($2 = '' OR a.command = $2)`,
		filter.ClientID, filter.Command,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count activities: %w", err)
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}

	rows, err := a.db.QueryContext(ctx,
		`SELECT a.id, a.client_id, a.command, COALESCE(a.repo,''), COALESCE(a.branch,''), a.timestamp, c.git_email
		 FROM coder_client_activity a
		 JOIN coder_clients c ON c.id = a.client_id
		 WHERE ($1 = '' OR a.client_id = $1)
		   AND ($2 = '' OR a.command = $2)
		 ORDER BY a.timestamp DESC
		 LIMIT $3 OFFSET $4`,
		filter.ClientID, filter.Command, limit, filter.Offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("query activities: %w", err)
	}
	defer rows.Close()

	var acts []authdomain.Activity
	for rows.Next() {
		var act authdomain.Activity
		if err := rows.Scan(&act.ID, &act.ClientID, &act.Command, &act.Repo, &act.Branch, &act.Timestamp, &act.GitEmail); err != nil {
			return nil, 0, err
		}
		acts = append(acts, act)
	}
	return acts, total, nil
}

func (a *postgresAuth) GetActivityStats(ctx context.Context, days int) (authdomain.ActivityStats, error) {
	var stats authdomain.ActivityStats

	// 1. Total clients
	if err := a.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM coder_clients`).Scan(&stats.TotalClients); err != nil {
		return stats, fmt.Errorf("total clients: %w", err)
	}

	// 2. Total commands in last N days
	if err := a.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM coder_client_activity WHERE timestamp >= NOW() - ($1 || ' days')::INTERVAL`,
		days,
	).Scan(&stats.TotalCommands); err != nil {
		return stats, fmt.Errorf("total commands: %w", err)
	}

	// 3. Active today (distinct client_id)
	if err := a.db.QueryRowContext(ctx,
		`SELECT COUNT(DISTINCT client_id) FROM coder_client_activity WHERE timestamp >= CURRENT_DATE`,
	).Scan(&stats.ActiveToday); err != nil {
		return stats, fmt.Errorf("active today: %w", err)
	}

	// 4. Active this week (distinct client_id, last 7 days)
	if err := a.db.QueryRowContext(ctx,
		`SELECT COUNT(DISTINCT client_id) FROM coder_client_activity
		 WHERE timestamp >= NOW() - INTERVAL '7 days'`,
	).Scan(&stats.ActiveThisWeek); err != nil {
		return stats, fmt.Errorf("active this week: %w", err)
	}

	// 5. Unique repos (non-empty, last N days)
	if err := a.db.QueryRowContext(ctx,
		`SELECT COUNT(DISTINCT repo) FROM coder_client_activity
		 WHERE repo <> '' AND timestamp >= NOW() - ($1 || ' days')::INTERVAL`,
		days,
	).Scan(&stats.UniqueRepos); err != nil {
		return stats, fmt.Errorf("unique repos: %w", err)
	}

	// 6. Commands growth: compare current N days vs previous N days
	var prevCommands int
	if err := a.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM coder_client_activity
		 WHERE timestamp >= NOW() - ($1 || ' days')::INTERVAL * 2
		   AND timestamp <  NOW() - ($1 || ' days')::INTERVAL`,
		days,
	).Scan(&prevCommands); err != nil {
		return stats, fmt.Errorf("commands growth: %w", err)
	}
	if prevCommands > 0 {
		stats.CommandsGrowth = float64(stats.TotalCommands-prevCommands) / float64(prevCommands) * 100
	}

	// 7. Avg commands per day
	if days > 0 {
		stats.AvgCommandsPerDay = float64(stats.TotalCommands) / float64(days)
	}

	// 8. Commands per day (last N days)
	rows, err := a.db.QueryContext(ctx,
		`SELECT TO_CHAR(DATE_TRUNC('day', timestamp), 'YYYY-MM-DD'), COUNT(*)
		 FROM coder_client_activity
		 WHERE timestamp >= NOW() - ($1 || ' days')::INTERVAL
		 GROUP BY DATE_TRUNC('day', timestamp)
		 ORDER BY DATE_TRUNC('day', timestamp)`,
		days,
	)
	if err != nil {
		return stats, fmt.Errorf("commands per day: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var dc authdomain.DailyCount
		if err := rows.Scan(&dc.Date, &dc.Count); err != nil {
			return stats, err
		}
		stats.CommandsPerDay = append(stats.CommandsPerDay, dc)
	}
	rows.Close()

	// 9. Distinct active clients per day (last N days)
	clientRows, err := a.db.QueryContext(ctx,
		`SELECT TO_CHAR(DATE_TRUNC('day', timestamp), 'YYYY-MM-DD'), COUNT(DISTINCT client_id)
		 FROM coder_client_activity
		 WHERE timestamp >= NOW() - ($1 || ' days')::INTERVAL
		 GROUP BY DATE_TRUNC('day', timestamp)
		 ORDER BY DATE_TRUNC('day', timestamp)`,
		days,
	)
	if err != nil {
		return stats, fmt.Errorf("clients per day: %w", err)
	}
	defer clientRows.Close()
	for clientRows.Next() {
		var dc authdomain.DailyCount
		if err := clientRows.Scan(&dc.Date, &dc.Count); err != nil {
			return stats, err
		}
		stats.ClientsPerDay = append(stats.ClientsPerDay, dc)
	}
	clientRows.Close()

	// 10. Top commands (last N days)
	cmdRows, err := a.db.QueryContext(ctx,
		`SELECT command, COUNT(*) AS cnt
		 FROM coder_client_activity
		 WHERE timestamp >= NOW() - ($1 || ' days')::INTERVAL
		 GROUP BY command
		 ORDER BY cnt DESC
		 LIMIT 8`,
		days,
	)
	if err != nil {
		return stats, fmt.Errorf("top commands: %w", err)
	}
	defer cmdRows.Close()
	var totalCmds int
	for cmdRows.Next() {
		var cc authdomain.CommandCount
		if err := cmdRows.Scan(&cc.Command, &cc.Count); err != nil {
			return stats, err
		}
		totalCmds += cc.Count
		stats.TopCommands = append(stats.TopCommands, cc)
	}
	cmdRows.Close()
	if totalCmds > 0 {
		for i := range stats.TopCommands {
			stats.TopCommands[i].Percent = float64(stats.TopCommands[i].Count) / float64(totalCmds) * 100
		}
	}

	// 11. Top repos by command count (last N days)
	repoRows, err := a.db.QueryContext(ctx,
		`SELECT COALESCE(repo,''), COUNT(*) AS cnt
		 FROM coder_client_activity
		 WHERE repo <> '' AND timestamp >= NOW() - ($1 || ' days')::INTERVAL
		 GROUP BY repo
		 ORDER BY cnt DESC
		 LIMIT 6`,
		days,
	)
	if err != nil {
		return stats, fmt.Errorf("top repos: %w", err)
	}
	defer repoRows.Close()
	for repoRows.Next() {
		var rc authdomain.RepoCount
		if err := repoRows.Scan(&rc.Repo, &rc.Count); err != nil {
			return stats, err
		}
		stats.TopRepos = append(stats.TopRepos, rc)
	}
	repoRows.Close()

	// 12. Activity by hour of day 0–23 (last N days)
	hourRows, err := a.db.QueryContext(ctx,
		`SELECT EXTRACT(HOUR FROM timestamp)::INT, COUNT(*)
		 FROM coder_client_activity
		 WHERE timestamp >= NOW() - ($1 || ' days')::INTERVAL
		 GROUP BY EXTRACT(HOUR FROM timestamp)
		 ORDER BY 1`,
		days,
	)
	if err != nil {
		return stats, fmt.Errorf("activity by hour: %w", err)
	}
	defer hourRows.Close()
	for hourRows.Next() {
		var hc authdomain.HourCount
		if err := hourRows.Scan(&hc.Hour, &hc.Count); err != nil {
			return stats, err
		}
		stats.ActivityByHour = append(stats.ActivityByHour, hc)
	}
	hourRows.Close()

	// 13. Recent activity (last 10 rows with git_email)
	recentRows, err := a.db.QueryContext(ctx,
		`SELECT a.id, a.client_id, a.command, COALESCE(a.repo,''), COALESCE(a.branch,''), a.timestamp, c.git_email
		 FROM coder_client_activity a
		 JOIN coder_clients c ON c.id = a.client_id
		 ORDER BY a.timestamp DESC
		 LIMIT 10`,
	)
	if err != nil {
		return stats, fmt.Errorf("recent activity: %w", err)
	}
	defer recentRows.Close()
	for recentRows.Next() {
		var act authdomain.Activity
		if err := recentRows.Scan(&act.ID, &act.ClientID, &act.Command, &act.Repo, &act.Branch, &act.Timestamp, &act.GitEmail); err != nil {
			return stats, err
		}
		stats.RecentActivity = append(stats.RecentActivity, act)
	}

	return stats, nil
}
