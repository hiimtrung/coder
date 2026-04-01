package memory

type RecallOptions struct {
	Task        string         `json:"task"`
	Current     []string       `json:"current,omitempty"`
	Trigger     string         `json:"trigger,omitempty"`
	Budget      int            `json:"budget,omitempty"`
	Limit       int            `json:"limit,omitempty"`
	Scope       string         `json:"scope,omitempty"`
	Tags        []string       `json:"tags,omitempty"`
	Type        MemoryType     `json:"type,omitempty"`
	MetaFilters map[string]any `json:"meta_filters,omitempty"`
}

type RecalledMemory struct {
	Result SearchResult `json:"result"`
	Reason string       `json:"reason,omitempty"`
}

type RecallResult struct {
	Task      string           `json:"task"`
	Trigger   string           `json:"trigger,omitempty"`
	Budget    int              `json:"budget"`
	Limit     int              `json:"limit"`
	Coverage  string           `json:"coverage"`
	Keep      []string         `json:"keep,omitempty"`
	Add       []string         `json:"add,omitempty"`
	Drop      []string         `json:"drop,omitempty"`
	Conflicts []string         `json:"conflicts,omitempty"`
	Memories  []RecalledMemory `json:"memories"`
}
