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
