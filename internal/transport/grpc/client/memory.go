package grpcclient

import (
	"context"
	"encoding/json"

	"github.com/trungtran/coder/api/grpc/memorypb"
	memdomain "github.com/trungtran/coder/internal/domain/memory"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client is a gRPC memory client that implements memdomain.MemoryManager.
type Client struct {
	conn *grpc.ClientConn
	c    memorypb.MemoryServiceClient
}

func NewClient(target string) (*Client, error) {
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	c := memorypb.NewMemoryServiceClient(conn)
	return &Client{conn: conn, c: c}, nil
}

func (c *Client) Store(ctx context.Context, title, content string, memType memdomain.MemoryType, metadata map[string]any, scope string, tags []string) (string, error) {
	var metaBytes []byte
	if metadata != nil {
		metaBytes, _ = json.Marshal(metadata)
	}

	req := &memorypb.StoreRequest{
		Title:        title,
		Content:      content,
		Type:         string(memType),
		MetadataJson: string(metaBytes),
		Scope:        scope,
		Tags:         tags,
	}

	res, err := c.c.Store(ctx, req)
	if err != nil {
		return "", err
	}
	return res.Id, nil
}

func (c *Client) Search(ctx context.Context, query string, scope string, tags []string, memType memdomain.MemoryType, metaFilters map[string]any, limit int) ([]memdomain.SearchResult, error) {
	var metaBytes []byte
	if metaFilters != nil {
		metaBytes, _ = json.Marshal(metaFilters)
	}

	req := &memorypb.SearchRequest{
		Query:           query,
		Scope:           scope,
		Tags:            tags,
		Type:            string(memType),
		MetaFiltersJson: string(metaBytes),
		Limit:           int32(limit),
	}

	res, err := c.c.Search(ctx, req)
	if err != nil {
		return nil, err
	}

	var results []memdomain.SearchResult
	for _, r := range res.Results {
		var meta map[string]any
		if r.Knowledge.MetadataJson != "" {
			json.Unmarshal([]byte(r.Knowledge.MetadataJson), &meta)
		}

		results = append(results, memdomain.SearchResult{
			Score: r.Score,
			Knowledge: memdomain.Knowledge{
				ID:              r.Knowledge.Id,
				Title:           r.Knowledge.Title,
				Content:         r.Knowledge.Content,
				Type:            memdomain.MemoryType(r.Knowledge.Type),
				Metadata:        meta,
				Tags:            r.Knowledge.Tags,
				Scope:           r.Knowledge.Scope,
				ParentID:        r.Knowledge.ParentId,
				ChunkIndex:      int(r.Knowledge.ChunkIndex),
				NormalizedTitle: r.Knowledge.NormalizedTitle,
				ContentHash:     r.Knowledge.ContentHash,
				CreatedAt:       r.Knowledge.CreatedAt.AsTime(),
				UpdatedAt:       r.Knowledge.UpdatedAt.AsTime(),
			},
		})
	}
	return results, nil
}

func (c *Client) List(ctx context.Context, limit, offset int) ([]memdomain.Knowledge, error) {
	req := &memorypb.ListRequest{
		Limit:  int32(limit),
		Offset: int32(offset),
	}

	res, err := c.c.List(ctx, req)
	if err != nil {
		return nil, err
	}

	var items []memdomain.Knowledge
	for _, r := range res.Items {
		var meta map[string]any
		if r.MetadataJson != "" {
			json.Unmarshal([]byte(r.MetadataJson), &meta)
		}
		items = append(items, memdomain.Knowledge{
			ID:              r.Id,
			Title:           r.Title,
			Content:         r.Content,
			Type:            memdomain.MemoryType(r.Type),
			Metadata:        meta,
			Tags:            r.Tags,
			Scope:           r.Scope,
			ParentID:        r.ParentId,
			ChunkIndex:      int(r.ChunkIndex),
			NormalizedTitle: r.NormalizedTitle,
			ContentHash:     r.ContentHash,
			CreatedAt:       r.CreatedAt.AsTime(),
			UpdatedAt:       r.UpdatedAt.AsTime(),
		})
	}
	return items, nil
}

func (c *Client) Delete(ctx context.Context, id string) error {
	_, err := c.c.Delete(ctx, &memorypb.DeleteRequest{Id: id})
	return err
}

func (c *Client) Compact(ctx context.Context, threshold float32) (int, error) {
	res, err := c.c.Compact(ctx, &memorypb.CompactRequest{Threshold: threshold, Revector: false})
	if err != nil {
		return 0, err
	}
	return int(res.RemovedCount), nil
}

func (c *Client) Revector(ctx context.Context) error {
	_, err := c.c.Compact(ctx, &memorypb.CompactRequest{Threshold: 0, Revector: true})
	return err
}

func (c *Client) Close() error {
	return c.conn.Close()
}
