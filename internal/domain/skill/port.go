package skill

import (
	"context"
	"time"
)

// SkillRepository is the persistence port.
type SkillRepository interface {
	UpsertSkill(ctx context.Context, s *Skill) error
	StoreChunk(ctx context.Context, c *SkillChunk) error
	GetSkill(ctx context.Context, name string) (*Skill, []SkillChunk, error)
	ListSkills(ctx context.Context, source string, category string, limit int, offset int) ([]Skill, error)
	DeleteSkill(ctx context.Context, name string) error
	SearchChunks(ctx context.Context, queryVector []float32, queryText string, limit int) ([]SkillChunkResult, error)
	GetSkillByID(ctx context.Context, id string) (*Skill, error)
	GetAllChunksBySkillID(ctx context.Context, skillID string) ([]SkillChunk, error)
	GetChunksBySectionID(ctx context.Context, sectionID string) ([]SkillChunk, error)
	GetAdjacentChunks(ctx context.Context, skillID string, chunkIndex int, radius int) ([]SkillChunk, error)
	GetChunkHashes(ctx context.Context, skillID string) (map[string]bool, error)
	DeleteChunksBySkillID(ctx context.Context, skillID string) error
	StoreFile(ctx context.Context, f *SkillFile) error
	GetFiles(ctx context.Context, skillID string) ([]SkillFile, error)
	DeleteFilesBySkillID(ctx context.Context, skillID string) error
	Close() error
}

// SkillClient is the interface for remote skill operations (used by CacheManager + CLI).
type SkillClient interface {
	IngestSkill(ctx context.Context, name, skillMD string, rules []RuleFile, source, sourceRepo, category string) (*IngestResult, error)
	SearchSkills(ctx context.Context, query string, limit int) ([]SkillSearchResult, error)
	ListSkills(ctx context.Context, source, category string, limit, offset int) ([]Skill, error)
	GetSkill(ctx context.Context, name string) (*Skill, []SkillChunk, error)
	DeleteSkill(ctx context.Context, name string) error
	StoreSkillFiles(ctx context.Context, skillName string, files []SkillFile) (int, error)
	GetSkillFiles(ctx context.Context, skillName string) ([]SkillFile, error)
	Close() error
}

// SkillUseCase is the combined application service interface for transport servers.
// Implemented by a facade in usecase/skill.
type SkillUseCase interface {
	IngestSkill(ctx context.Context, name, skillMD string, rules []RuleFile, source, sourceRepo, category string) (*IngestResult, error)
	IngestFiles(ctx context.Context, skillName string, files []SkillFile) (int, error)
	SearchSkills(ctx context.Context, query string, limit int) ([]SkillSearchResult, error)
	ListSkills(ctx context.Context, source, category string, limit, offset int) ([]Skill, error)
	GetSkill(ctx context.Context, name string) (*Skill, []SkillChunk, error)
	DeleteSkill(ctx context.Context, name string) error
	GetFiles(ctx context.Context, skillID string) ([]SkillFile, error)
	StoreFiles(ctx context.Context, skillID string, files []SkillFile, now time.Time) (int, error)
}
