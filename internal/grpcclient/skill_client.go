package grpcclient

import (
	"context"
	"fmt"

	"github.com/trungtran/coder/api/grpc/skillpb"
	"github.com/trungtran/coder/internal/skill"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type skillClient struct {
	conn   *grpc.ClientConn
	client skillpb.SkillServiceClient
}

// NewSkillClient creates a new gRPC client for the skill service.
func NewSkillClient(addr string) (skill.Client, error) {
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to skill service: %w", err)
	}

	return &skillClient{
		conn:   conn,
		client: skillpb.NewSkillServiceClient(conn),
	}, nil
}

func (c *skillClient) IngestSkill(ctx context.Context, name, skillMD string, rules []skill.RuleFile, source, sourceRepo, category string) (*skill.IngestResult, error) {
	var pbRules []*skillpb.RuleFileEntry
	for _, r := range rules {
		pbRules = append(pbRules, &skillpb.RuleFileEntry{
			Path:    r.Path,
			Content: r.Content,
		})
	}

	resp, err := c.client.IngestSkill(ctx, &skillpb.IngestSkillRequest{
		Name:           name,
		SkillMdContent: skillMD,
		Rules:          pbRules,
		Source:         source,
		SourceRepo:     sourceRepo,
		Category:       category,
	})
	if err != nil {
		return nil, err
	}

	return &skill.IngestResult{
		SkillName:   resp.SkillId,
		ChunksTotal: int(resp.ChunksTotal),
		ChunksNew:   int(resp.ChunksNew),
	}, nil
}

func (c *skillClient) SearchSkills(ctx context.Context, query string, limit int) ([]skill.SkillSearchResult, error) {
	resp, err := c.client.SearchSkills(ctx, &skillpb.SearchSkillsRequest{
		Query: query,
		Limit: int32(limit),
	})
	if err != nil {
		return nil, err
	}

	var results []skill.SkillSearchResult
	for _, r := range resp.Results {
		sr := skill.SkillSearchResult{
			Skill: skill.Skill{
				ID:          r.Skill.Id,
				Name:        r.Skill.Name,
				Description: r.Skill.Description,
				Category:    r.Skill.Category,
				Source:      r.Skill.Source,
				SourceRepo:  r.Skill.SourceRepo,
				Risk:        r.Skill.Risk,
				Version:     r.Skill.Version,
				Tags:        r.Skill.Tags,
				ChunkCount:  int(r.Skill.ChunkCount),
				CreatedAt:   r.Skill.CreatedAt.AsTime(),
				UpdatedAt:   r.Skill.UpdatedAt.AsTime(),
			},
			Score: r.Score,
		}

		for _, ch := range r.Chunks {
			sr.Chunks = append(sr.Chunks, skill.SkillChunkResult{
				SkillChunk: skill.SkillChunk{
					ID:         ch.Id,
					SkillID:    ch.SkillId,
					ChunkType:  ch.ChunkType,
					Title:      ch.Title,
					Content:    ch.Content,
					ChunkIndex: int(ch.ChunkIndex),
				},
				Score: ch.Score,
			})
		}
		results = append(results, sr)
	}

	return results, nil
}

func (c *skillClient) ListSkills(ctx context.Context, source, category string, limit, offset int) ([]skill.Skill, error) {
	resp, err := c.client.ListSkills(ctx, &skillpb.ListSkillsRequest{
		Source:   source,
		Category: category,
		Limit:    int32(limit),
		Offset:   int32(offset),
	})
	if err != nil {
		return nil, err
	}

	var skills []skill.Skill
	for _, sk := range resp.Skills {
		skills = append(skills, skill.Skill{
			ID:          sk.Id,
			Name:        sk.Name,
			Description: sk.Description,
			Category:    sk.Category,
			Source:      sk.Source,
			SourceRepo:  sk.SourceRepo,
			Risk:        sk.Risk,
			Version:     sk.Version,
			Tags:        sk.Tags,
			ChunkCount:  int(sk.ChunkCount),
			CreatedAt:   sk.CreatedAt.AsTime(),
			UpdatedAt:   sk.UpdatedAt.AsTime(),
		})
	}

	return skills, nil
}

func (c *skillClient) GetSkill(ctx context.Context, name string) (*skill.Skill, []skill.SkillChunk, error) {
	resp, err := c.client.GetSkill(ctx, &skillpb.GetSkillRequest{Name: name})
	if err != nil {
		return nil, nil, err
	}

	sk := &skill.Skill{
		ID:          resp.Skill.Id,
		Name:        resp.Skill.Name,
		Description: resp.Skill.Description,
		Category:    resp.Skill.Category,
		Source:      resp.Skill.Source,
		SourceRepo:  resp.Skill.SourceRepo,
		Risk:        resp.Skill.Risk,
		Version:     resp.Skill.Version,
		Tags:        resp.Skill.Tags,
		ChunkCount:  int(resp.Skill.ChunkCount),
		CreatedAt:   resp.Skill.CreatedAt.AsTime(),
		UpdatedAt:   resp.Skill.UpdatedAt.AsTime(),
	}

	var chunks []skill.SkillChunk
	for _, ch := range resp.Chunks {
		chunks = append(chunks, skill.SkillChunk{
			ID:         ch.Id,
			SkillID:    ch.SkillId,
			ChunkType:  ch.ChunkType,
			Title:      ch.Title,
			Content:    ch.Content,
			ChunkIndex: int(ch.ChunkIndex),
		})
	}

	return sk, chunks, nil
}

func (c *skillClient) DeleteSkill(ctx context.Context, name string) error {
	_, err := c.client.DeleteSkill(ctx, &skillpb.DeleteSkillRequest{Name: name})
	return err
}

func (c *skillClient) Close() error {
	return c.conn.Close()
}
