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

// DailyCount is one point for a time-series chart (commands or clients per day).
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

// RepoCount is one bar in the top-repos chart.
type RepoCount struct {
	Repo  string `json:"repo"`
	Count int    `json:"count"`
}

// HourCount is one bucket in the activity-by-hour bar chart (0–23).
type HourCount struct {
	Hour  int `json:"hour"`
	Count int `json:"count"`
}

// ActivityChartStats holds aggregated chart data for the activity log page,
// optionally filtered by client and/or command.
type ActivityChartStats struct {
	Total          int            `json:"total"`
	CommandsPerDay []DailyCount   `json:"commands_per_day"`
	TopCommands    []CommandCount `json:"top_commands"`
	ActivityByHour []HourCount    `json:"activity_by_hour"`
}

// ActivityStats bundles all dashboard overview data.
type ActivityStats struct {
	// KPI cards
	TotalClients      int     `json:"total_clients"`
	TotalCommands     int     `json:"total_commands"`
	ActiveToday       int     `json:"active_today"`
	ActiveThisWeek    int     `json:"active_this_week"`
	UniqueRepos       int     `json:"unique_repos"`
	AvgCommandsPerDay float64 `json:"avg_commands_per_day"`
	CommandsGrowth    float64 `json:"commands_growth"` // % change vs previous period

	// Time-series charts
	CommandsPerDay []DailyCount `json:"commands_per_day"`
	ClientsPerDay  []DailyCount `json:"clients_per_day"`

	// Distribution charts
	TopCommands    []CommandCount `json:"top_commands"`
	TopRepos       []RepoCount    `json:"top_repos"`
	ActivityByHour []HourCount    `json:"activity_by_hour"`

	// Table
	RecentActivity []Activity `json:"recent_activity"`
}
