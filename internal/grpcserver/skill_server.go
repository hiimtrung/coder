package grpcserver

import (
	"context"

	"github.com/trungtran/coder/api/grpc/skillpb"
	"github.com/trungtran/coder/internal/skill"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type SkillServer struct {
	skillpb.UnimplementedSkillServiceServer
	ingestor *skill.Ingestor
	store    skill.SkillService
}

func NewSkillServer(ingestor *skill.Ingestor, store skill.SkillService) *SkillServer {
	return &SkillServer{
		ingestor: ingestor,
		store:    store,
	}
}

func (s *SkillServer) IngestSkill(ctx context.Context, req *skillpb.IngestSkillRequest) (*skillpb.IngestSkillResponse, error) {
	var rules []skill.RuleFile
	for _, r := range req.Rules {
		rules = append(rules, skill.RuleFile{
			Path:    r.Path,
			Content: r.Content,
		})
	}

	result, err := s.ingestor.IngestSkill(ctx, req.Name, req.SkillMdContent, rules, req.Source, req.SourceRepo, req.Category)
	if err != nil {
		return nil, err
	}

	return &skillpb.IngestSkillResponse{
		SkillId:     result.SkillName,
		ChunksTotal: int32(result.ChunksTotal),
		ChunksNew:   int32(result.ChunksNew),
	}, nil
}

func (s *SkillServer) SearchSkills(ctx context.Context, req *skillpb.SearchSkillsRequest) (*skillpb.SearchSkillsResponse, error) {
	limit := int(req.Limit)
	if limit <= 0 {
		limit = 5
	}

	results, err := s.ingestor.SearchSkills(ctx, req.Query, limit)
	if err != nil {
		return nil, err
	}

	var pbResults []*skillpb.SkillSearchResult
	for _, r := range results {
		pbSkill := skillToProto(&r.Skill)
		var pbChunks []*skillpb.SkillChunkResult
		for _, c := range r.Chunks {
			pbChunks = append(pbChunks, &skillpb.SkillChunkResult{
				Id:         c.ID,
				SkillId:    c.SkillID,
				ChunkType:  c.ChunkType,
				Title:      c.Title,
				Content:    c.Content,
				ChunkIndex: int32(c.ChunkIndex),
				Score:      c.Score,
			})
		}

		pbResults = append(pbResults, &skillpb.SkillSearchResult{
			Skill:  pbSkill,
			Chunks: pbChunks,
			Score:  r.Score,
		})
	}

	return &skillpb.SearchSkillsResponse{Results: pbResults}, nil
}

func (s *SkillServer) ListSkills(ctx context.Context, req *skillpb.ListSkillsRequest) (*skillpb.ListSkillsResponse, error) {
	skills, err := s.store.ListSkills(ctx, req.Source, req.Category, int(req.Limit), int(req.Offset))
	if err != nil {
		return nil, err
	}

	var pbSkills []*skillpb.SkillInfo
	for _, sk := range skills {
		pbSkills = append(pbSkills, skillToProto(&sk))
	}

	return &skillpb.ListSkillsResponse{Skills: pbSkills}, nil
}

func (s *SkillServer) GetSkill(ctx context.Context, req *skillpb.GetSkillRequest) (*skillpb.GetSkillResponse, error) {
	sk, chunks, err := s.store.GetSkill(ctx, req.Name)
	if err != nil {
		return nil, err
	}

	pbSkill := skillToProto(sk)

	var pbChunks []*skillpb.SkillChunkResult
	for _, c := range chunks {
		pbChunks = append(pbChunks, &skillpb.SkillChunkResult{
			Id:         c.ID,
			SkillId:    c.SkillID,
			ChunkType:  c.ChunkType,
			Title:      c.Title,
			Content:    c.Content,
			ChunkIndex: int32(c.ChunkIndex),
		})
	}

	return &skillpb.GetSkillResponse{
		Skill:  pbSkill,
		Chunks: pbChunks,
	}, nil
}

func (s *SkillServer) DeleteSkill(ctx context.Context, req *skillpb.DeleteSkillRequest) (*skillpb.DeleteSkillResponse, error) {
	err := s.store.DeleteSkill(ctx, req.Name)
	if err != nil {
		return nil, err
	}
	return &skillpb.DeleteSkillResponse{Success: true}, nil
}

func skillToProto(sk *skill.Skill) *skillpb.SkillInfo {
	return &skillpb.SkillInfo{
		Id:          sk.ID,
		Name:        sk.Name,
		Description: sk.Description,
		Category:    sk.Category,
		Source:      sk.Source,
		SourceRepo:  sk.SourceRepo,
		Risk:        sk.Risk,
		Version:     sk.Version,
		Tags:        sk.Tags,
		ChunkCount:  int32(sk.ChunkCount),
		CreatedAt:   timestamppb.New(sk.CreatedAt),
		UpdatedAt:   timestamppb.New(sk.UpdatedAt),
	}
}
