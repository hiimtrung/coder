package ucskill

import (
	"context"
	"time"

	skilldomain "github.com/trungtran/coder/internal/domain/skill"
)

// SkillFacade implements skilldomain.SkillUseCase by wrapping Ingestor + SkillRepository.
type SkillFacade struct {
	ingestor *Ingestor
	repo     skilldomain.SkillRepository
}

// NewSkillFacade creates a new SkillFacade.
func NewSkillFacade(ingestor *Ingestor, repo skilldomain.SkillRepository) *SkillFacade {
	return &SkillFacade{
		ingestor: ingestor,
		repo:     repo,
	}
}

func (f *SkillFacade) IngestSkill(ctx context.Context, name, skillMD string, rules []skilldomain.RuleFile, source, sourceRepo, category string) (*skilldomain.IngestResult, error) {
	return f.ingestor.IngestSkill(ctx, name, skillMD, rules, source, sourceRepo, category)
}

func (f *SkillFacade) IngestFiles(ctx context.Context, skillName string, files []skilldomain.SkillFile) (int, error) {
	return f.ingestor.IngestFiles(ctx, skillName, files)
}

func (f *SkillFacade) SearchSkills(ctx context.Context, query string, limit int) ([]skilldomain.SkillSearchResult, error) {
	return f.ingestor.SearchSkills(ctx, query, limit)
}

func (f *SkillFacade) ListSkills(ctx context.Context, source, category string, limit, offset int) ([]skilldomain.Skill, error) {
	return f.repo.ListSkills(ctx, source, category, limit, offset)
}

func (f *SkillFacade) GetSkill(ctx context.Context, name string) (*skilldomain.Skill, []skilldomain.SkillChunk, error) {
	return f.repo.GetSkill(ctx, name)
}

func (f *SkillFacade) DeleteSkill(ctx context.Context, name string) error {
	return f.repo.DeleteSkill(ctx, name)
}

func (f *SkillFacade) GetFiles(ctx context.Context, skillID string) ([]skilldomain.SkillFile, error) {
	return f.repo.GetFiles(ctx, skillID)
}

// StoreFiles replaces all files for a skill: deletes old files then stores the new ones.
// This replicates the delete-then-store logic from the HTTP server's POST /v1/skill/files handler.
func (f *SkillFacade) StoreFiles(ctx context.Context, skillID string, files []skilldomain.SkillFile, now time.Time) (int, error) {
	if err := f.repo.DeleteFilesBySkillID(ctx, skillID); err != nil {
		return 0, err
	}

	stored := 0
	for i := range files {
		files[i].SkillID = skillID
		files[i].CreatedAt = now
		if err := f.repo.StoreFile(ctx, &files[i]); err != nil {
			return stored, err
		}
		stored++
	}
	return stored, nil
}
