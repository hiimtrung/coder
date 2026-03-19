package auth

import "time"

// Client represents a registered coder CLI client.
type Client struct {
	ID          string    `json:"id"`
	GitName     string    `json:"git_name"`
	GitEmail    string    `json:"git_email"`
	AccessToken string    `json:"access_token"` // SHA-256 hex hash stored in DB; raw token returned once
	CreatedAt   time.Time `json:"created_at"`
	LastSeenAt  time.Time `json:"last_seen_at"`
}

// Activity records a single coder CLI command execution.
type Activity struct {
	ID        string    `json:"id"`
	ClientID  string    `json:"client_id"`
	Command   string    `json:"command"`  // e.g. "memory search", "skill ingest"
	Repo      string    `json:"repo"`     // git remote.origin.url (sanitised)
	Branch    string    `json:"branch"`   // current git branch
	Timestamp time.Time `json:"timestamp"`
	// GitEmail is populated only in dashboard queries via JOIN — empty in normal activity logs.
	GitEmail string `json:"git_email,omitempty"`
}

// ActivityFilter for paginated activity queries.
type ActivityFilter struct {
	ClientID string
	Command  string
	Limit    int
	Offset   int
}

// DailyCount is one point for commands-per-day chart.
type DailyCount struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

// CommandCount is one slice of the top-commands donut chart.
type CommandCount struct {
	Command string  `json:"command"`
	Count   int     `json:"count"`
	Percent float64 `json:"percent"`
}

// ActivityStats bundles all dashboard overview data.
type ActivityStats struct {
	TotalClients   int            `json:"total_clients"`
	TotalCommands  int            `json:"total_commands"`
	ActiveToday    int            `json:"active_today"`
	UniqueRepos    int            `json:"unique_repos"`
	CommandsPerDay []DailyCount   `json:"commands_per_day"`
	TopCommands    []CommandCount `json:"top_commands"`
	RecentActivity []Activity     `json:"recent_activity"`
}
