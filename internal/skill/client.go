package skill

import (
	"context"
)

// Client defines the interface for communicating with the coder-node skill service remotely.
type Client interface {
	IngestSkill(ctx context.Context, name, skillMD string, rules []RuleFile, source, sourceRepo, category string) (*IngestResult, error)
	SearchSkills(ctx context.Context, query string, limit int) ([]SkillSearchResult, error)
	ListSkills(ctx context.Context, source, category string, limit, offset int) ([]Skill, error)
	GetSkill(ctx context.Context, name string) (*Skill, []SkillChunk, error)
	DeleteSkill(ctx context.Context, name string) error

	// File management — store and retrieve raw scripts/data for cache extraction
	StoreSkillFiles(ctx context.Context, skillName string, files []SkillFile) (int, error)
	GetSkillFiles(ctx context.Context, skillName string) ([]SkillFile, error)

	Close() error
}
