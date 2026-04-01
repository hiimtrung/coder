package grpcserver

import (
	"context"
	"encoding/json"

	"github.com/trungtran/coder/api/grpc/memorypb"
	memdomain "github.com/trungtran/coder/internal/domain/memory"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type MemoryServer struct {
	memorypb.UnimplementedMemoryServiceServer
	manager memdomain.MemoryManager
}

func NewMemoryServer(manager memdomain.MemoryManager) *MemoryServer {
	return &MemoryServer{
		manager: manager,
	}
}

func (s *MemoryServer) Store(ctx context.Context, req *memorypb.StoreRequest) (*memorypb.StoreResponse, error) {
	var meta map[string]any
	if req.MetadataJson != "" {
		if err := json.Unmarshal([]byte(req.MetadataJson), &meta); err != nil {
			return nil, err
		}
	}

	id, err := s.manager.Store(
		ctx,
		req.Title,
		req.Content,
		memdomain.MemoryType(req.Type),
		meta,
		req.Scope,
		req.Tags,
	)
	if err != nil {
		return nil, err
	}

	return &memorypb.StoreResponse{Id: id}, nil
}

func (s *MemoryServer) Search(ctx context.Context, req *memorypb.SearchRequest) (*memorypb.SearchResponse, error) {
	var meta map[string]any
	if req.MetaFiltersJson != "" {
		if err := json.Unmarshal([]byte(req.MetaFiltersJson), &meta); err != nil {
			return nil, err
		}
	}

	results, err := s.manager.Search(
		ctx,
		req.Query,
		req.Scope,
		req.Tags,
		memdomain.MemoryType(req.Type),
		meta,
		int(req.Limit),
	)
	if err != nil {
		return nil, err
	}

	var pbResults []*memorypb.SearchResult
	for _, r := range results {
		metaJson, _ := json.Marshal(r.Metadata)
		pbResults = append(pbResults, &memorypb.SearchResult{
			Score: r.Score,
			Knowledge: &memorypb.Knowledge{
				Id:              r.ID,
				Title:           r.Title,
				Content:         r.Content,
				Type:            string(r.Type),
				MetadataJson:    string(metaJson),
				Tags:            r.Tags,
				Scope:           r.Scope,
				ParentId:        r.ParentID,
				ChunkIndex:      int32(r.ChunkIndex),
				NormalizedTitle: r.NormalizedTitle,
				ContentHash:     r.ContentHash,
				CreatedAt:       timestamppb.New(r.CreatedAt),
				UpdatedAt:       timestamppb.New(r.UpdatedAt),
			},
		})
	}

	return &memorypb.SearchResponse{Results: pbResults}, nil
}

func (s *MemoryServer) Recall(ctx context.Context, req *memorypb.RecallRequest) (*memorypb.RecallResponse, error) {
	var meta map[string]any
	if req.MetaFiltersJson != "" {
		if err := json.Unmarshal([]byte(req.MetaFiltersJson), &meta); err != nil {
			return nil, err
		}
	}

	result, err := s.manager.Recall(ctx, memdomain.RecallOptions{
		Task:        req.Task,
		Current:     req.Current,
		Trigger:     req.Trigger,
		Budget:      int(req.Budget),
		Limit:       int(req.Limit),
		Scope:       req.Scope,
		Tags:        req.Tags,
		Type:        memdomain.MemoryType(req.Type),
		MetaFilters: meta,
	})
	if err != nil {
		return nil, err
	}

	pbMemories := make([]*memorypb.RecalledMemory, 0, len(result.Memories))
	for _, memory := range result.Memories {
		metaJSON, _ := json.Marshal(memory.Result.Metadata)
		pbMemories = append(pbMemories, &memorypb.RecalledMemory{
			Result: &memorypb.SearchResult{
				Score: memory.Result.Score,
				Knowledge: &memorypb.Knowledge{
					Id:              memory.Result.ID,
					Title:           memory.Result.Title,
					Content:         memory.Result.Content,
					Type:            string(memory.Result.Type),
					MetadataJson:    string(metaJSON),
					Tags:            memory.Result.Tags,
					Scope:           memory.Result.Scope,
					ParentId:        memory.Result.ParentID,
					ChunkIndex:      int32(memory.Result.ChunkIndex),
					NormalizedTitle: memory.Result.NormalizedTitle,
					ContentHash:     memory.Result.ContentHash,
					CreatedAt:       timestamppb.New(memory.Result.CreatedAt),
					UpdatedAt:       timestamppb.New(memory.Result.UpdatedAt),
				},
			},
			Reason: memory.Reason,
		})
	}

	return &memorypb.RecallResponse{
		Task:      result.Task,
		Trigger:   result.Trigger,
		Budget:    int32(result.Budget),
		Limit:     int32(result.Limit),
		Coverage:  result.Coverage,
		Keep:      result.Keep,
		Add:       result.Add,
		Drop:      result.Drop,
		Conflicts: result.Conflicts,
		Memories:  pbMemories,
	}, nil
}

func (s *MemoryServer) List(ctx context.Context, req *memorypb.ListRequest) (*memorypb.ListResponse, error) {
	items, err := s.manager.List(ctx, int(req.Limit), int(req.Offset))
	if err != nil {
		return nil, err
	}

	var pbItems []*memorypb.Knowledge
	for _, r := range items {
		metaJson, _ := json.Marshal(r.Metadata)
		pbItems = append(pbItems, &memorypb.Knowledge{
			Id:              r.ID,
			Title:           r.Title,
			Content:         r.Content,
			Type:            string(r.Type),
			MetadataJson:    string(metaJson),
			Tags:            r.Tags,
			Scope:           r.Scope,
			ParentId:        r.ParentID,
			ChunkIndex:      int32(r.ChunkIndex),
			NormalizedTitle: r.NormalizedTitle,
			ContentHash:     r.ContentHash,
			CreatedAt:       timestamppb.New(r.CreatedAt),
			UpdatedAt:       timestamppb.New(r.UpdatedAt),
		})
	}

	return &memorypb.ListResponse{Items: pbItems}, nil
}

func (s *MemoryServer) Delete(ctx context.Context, req *memorypb.DeleteRequest) (*memorypb.DeleteResponse, error) {
	err := s.manager.Delete(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return &memorypb.DeleteResponse{Success: true}, nil
}

func (s *MemoryServer) Verify(ctx context.Context, req *memorypb.VerifyRequest) (*memorypb.VerifyResponse, error) {
	opts := memdomain.VerifyOptions{
		VerifiedBy: req.VerifiedBy,
		SourceRef:  req.SourceRef,
	}
	if req.VerifiedAt != nil {
		opts.VerifiedAt = req.VerifiedAt.AsTime()
	}
	if req.HasConfidence {
		opts.Confidence = &req.Confidence
	}

	updated, err := s.manager.Verify(ctx, req.Id, opts)
	if err != nil {
		return nil, err
	}
	return &memorypb.VerifyResponse{UpdatedCount: int32(updated)}, nil
}

func (s *MemoryServer) Supersede(ctx context.Context, req *memorypb.SupersedeRequest) (*memorypb.SupersedeResponse, error) {
	updated, err := s.manager.Supersede(ctx, req.Id, req.ReplacementId)
	if err != nil {
		return nil, err
	}
	return &memorypb.SupersedeResponse{UpdatedCount: int32(updated)}, nil
}

func (s *MemoryServer) Audit(ctx context.Context, req *memorypb.AuditRequest) (*memorypb.AuditResponse, error) {
	report, err := s.manager.Audit(ctx, memdomain.AuditOptions{
		Scope:          req.Scope,
		UnverifiedDays: int(req.UnverifiedDays),
	})
	if err != nil {
		return nil, err
	}

	payload, err := json.Marshal(report)
	if err != nil {
		return nil, err
	}
	return &memorypb.AuditResponse{ReportJson: string(payload)}, nil
}

func (s *MemoryServer) Compact(ctx context.Context, req *memorypb.CompactRequest) (*memorypb.CompactResponse, error) {
	if req.Revector {
		if err := s.manager.Revector(ctx); err != nil {
			return nil, err
		}
	}
	removed, err := s.manager.Compact(ctx, req.Threshold)
	if err != nil {
		return nil, err
	}
	return &memorypb.CompactResponse{RemovedCount: int32(removed)}, nil
}
