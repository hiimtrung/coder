package ucskill

import (
	"context"
	"testing"
	"time"

	memdomain "github.com/trungtran/coder/internal/domain/memory"
	skilldomain "github.com/trungtran/coder/internal/domain/skill"
)

type fakeSkillRepo struct {
	skill  *skilldomain.Skill
	chunks []*skilldomain.SkillChunk
}

func (r *fakeSkillRepo) UpsertSkill(_ context.Context, s *skilldomain.Skill) error {
	copySkill := *s
	r.skill = &copySkill
	return nil
}

func (r *fakeSkillRepo) StoreChunk(_ context.Context, c *skilldomain.SkillChunk) error {
	copyChunk := *c
	r.chunks = append(r.chunks, &copyChunk)
	return nil
}

func (r *fakeSkillRepo) GetSkill(_ context.Context, _ string) (*skilldomain.Skill, []skilldomain.SkillChunk, error) {
	return nil, nil, nil
}

func (r *fakeSkillRepo) ListSkills(_ context.Context, _ string, _ string, _ int, _ int) ([]skilldomain.Skill, error) {
	return nil, nil
}

func (r *fakeSkillRepo) DeleteSkill(_ context.Context, _ string) error { return nil }

func (r *fakeSkillRepo) SearchChunks(_ context.Context, _ []float32, _ string, _ int) ([]skilldomain.SkillChunkResult, error) {
	return nil, nil
}

func (r *fakeSkillRepo) GetSkillByID(_ context.Context, _ string) (*skilldomain.Skill, error) {
	return nil, nil
}

func (r *fakeSkillRepo) GetAllChunksBySkillID(_ context.Context, _ string) ([]skilldomain.SkillChunk, error) {
	return nil, nil
}

func (r *fakeSkillRepo) GetChunksBySectionID(_ context.Context, _ string) ([]skilldomain.SkillChunk, error) {
	return nil, nil
}

func (r *fakeSkillRepo) GetAdjacentChunks(_ context.Context, _ string, _ int, _ int) ([]skilldomain.SkillChunk, error) {
	return nil, nil
}

func (r *fakeSkillRepo) GetChunkHashes(_ context.Context, _ string) (map[string]bool, error) {
	return map[string]bool{}, nil
}

func (r *fakeSkillRepo) DeleteChunksBySkillID(_ context.Context, _ string) error { return nil }

func (r *fakeSkillRepo) StoreFile(_ context.Context, _ *skilldomain.SkillFile) error { return nil }

func (r *fakeSkillRepo) GetFiles(_ context.Context, _ string) ([]skilldomain.SkillFile, error) { return nil, nil }

func (r *fakeSkillRepo) DeleteFilesBySkillID(_ context.Context, _ string) error { return nil }

func (r *fakeSkillRepo) Close() error { return nil }

type emptyEmbeddingProvider struct{}

func (emptyEmbeddingProvider) GenerateEmbedding(context.Context, string) ([]float32, error) {
	return []float32{}, nil
}

var _ skilldomain.SkillRepository = (*fakeSkillRepo)(nil)
var _ memdomain.EmbeddingProvider = (*emptyEmbeddingProvider)(nil)

func TestIngestSkillFallsBackWhenEmbeddingIsEmpty(t *testing.T) {
	repo := &fakeSkillRepo{}
	ing := NewIngestor(repo, emptyEmbeddingProvider{})

	skillMD := `---
name: architecture
description: Architecture rules
---

# Overview
Use clean architecture.

## Rules
Keep boundaries explicit.
`

	result, err := ing.IngestSkill(context.Background(), "architecture", skillMD, nil, "github", "hiimtrung/coder", "core")
	if err != nil {
		t.Fatalf("IngestSkill() returned error: %v", err)
	}
	if result.ChunksNew == 0 {
		t.Fatalf("expected chunks to be stored")
	}
	if len(repo.chunks) == 0 {
		t.Fatalf("expected stored chunks")
	}
	for _, chunk := range repo.chunks {
		if len(chunk.Vector) != 0 {
			t.Fatalf("expected empty vector fallback, got %d dimensions", len(chunk.Vector))
		}
		if chunk.CreatedAt.IsZero() || chunk.CreatedAt.After(time.Now().Add(time.Second)) {
			t.Fatalf("unexpected chunk timestamp: %v", chunk.CreatedAt)
		}
	}
}
