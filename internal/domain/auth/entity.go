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
	Command   string    `json:"command"` // e.g. "memory search", "skill ingest"
	Repo      string    `json:"repo"`    // git remote.origin.url (sanitised)
	Branch    string    `json:"branch"`  // current git branch
	Timestamp time.Time `json:"timestamp"`
}
