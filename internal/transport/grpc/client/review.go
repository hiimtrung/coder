package grpcclient

import (
	"context"
	"fmt"

	"github.com/trungtran/coder/api/grpc/reviewpb"
	reviewdomain "github.com/trungtran/coder/internal/domain/review"
	httpclient "github.com/trungtran/coder/internal/transport/http/client"
	"github.com/trungtran/coder/internal/transport/grpc/credential"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ReviewClient is a gRPC implementation of httpclient.ReviewClientIface.
type ReviewClient struct {
	conn   *grpc.ClientConn
	client reviewpb.ReviewServiceClient
}

// NewReviewClient dials the coder-node gRPC endpoint and returns a ReviewClientIface.
func NewReviewClient(addr, accessToken string) (httpclient.ReviewClientIface, error) {
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithPerRPCCredentials(credential.BearerToken{Token: accessToken}),
	}
	conn, err := grpc.NewClient(addr, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to review service: %w", err)
	}
	return &ReviewClient{
		conn:   conn,
		client: reviewpb.NewReviewServiceClient(conn),
	}, nil
}

// Review performs a structured code review over gRPC.
func (c *ReviewClient) Review(ctx context.Context, reviewType, content, focus string, injectMemory, injectSkills bool) (*reviewdomain.ReviewResult, error) {
	req := &reviewpb.ReviewRequest{
		Type:    reviewType,
		Content: content,
		Focus:   focus,
		Context: &reviewpb.ReviewContext{
			InjectMemory: injectMemory,
			InjectSkills: injectSkills,
		},
	}

	resp, err := c.client.Review(ctx, req)
	if err != nil {
		return nil, grpcConnError(err)
	}

	return protoToReviewResult(resp), nil
}

// Close releases the underlying gRPC connection.
func (c *ReviewClient) Close() error {
	return c.conn.Close()
}

// ── helpers ───────────────────────────────────────────────────────────────────

func protoToReviewResult(resp *reviewpb.ReviewResponse) *reviewdomain.ReviewResult {
	if resp == nil {
		return &reviewdomain.ReviewResult{}
	}

	var concerns []reviewdomain.ReviewConcern
	for _, c := range resp.Concerns {
		concerns = append(concerns, reviewdomain.ReviewConcern{
			Severity:    c.Severity,
			Description: c.Description,
			Location:    c.Location,
			Suggestion:  c.Suggestion,
		})
	}

	result := &reviewdomain.ReviewResult{
		Summary:     resp.Summary,
		Strengths:   resp.Strengths,
		Concerns:    concerns,
		Suggestions: resp.Suggestions,
		Model:       resp.Model,
	}
	if resp.Stats != nil {
		result.Stats = reviewdomain.ReviewStats{
			FilesReviewed:  int(resp.Stats.FilesReviewed),
			ConcernsHigh:   int(resp.Stats.ConcernsHigh),
			ConcernsMedium: int(resp.Stats.ConcernsMedium),
			ConcernsLow:    int(resp.Stats.ConcernsLow),
		}
	}
	if resp.ContextUsed != nil {
		result.ContextUsed = reviewdomain.ContextUsed{
			MemoryHits: resp.ContextUsed.MemoryHits,
			SkillHits:  resp.ContextUsed.SkillHits,
		}
	}
	return result
}

// Ensure ReviewClient implements the interface at compile time.
var _ httpclient.ReviewClientIface = (*ReviewClient)(nil)
