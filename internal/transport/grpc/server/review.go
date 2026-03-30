package grpcserver

import (
	"context"

	"github.com/trungtran/coder/api/grpc/reviewpb"
	reviewdomain "github.com/trungtran/coder/internal/domain/review"
	ucreview "github.com/trungtran/coder/internal/usecase/review"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ReviewServer implements reviewpb.ReviewServiceServer.
type ReviewServer struct {
	reviewpb.UnimplementedReviewServiceServer
	mgr *ucreview.Manager
}

func NewReviewServer(mgr *ucreview.Manager) *ReviewServer {
	return &ReviewServer{mgr: mgr}
}

// Review performs a structured code review.
func (s *ReviewServer) Review(ctx context.Context, req *reviewpb.ReviewRequest) (*reviewpb.ReviewResponse, error) {
	if req.Content == "" {
		return nil, status.Error(codes.InvalidArgument, "content is required")
	}

	reviewType := req.Type
	if reviewType == "" {
		reviewType = "diff"
	}

	injectMemory := true
	injectSkills := true
	if req.Context != nil {
		injectMemory = req.Context.InjectMemory
		injectSkills = req.Context.InjectSkills
	}

	result, err := s.mgr.Review(ctx, reviewdomain.ReviewRequest{
		Type:    reviewType,
		Content: req.Content,
		Focus:   req.Focus,
		Context: reviewdomain.ReviewContext{
			InjectMemory: injectMemory,
			InjectSkills: injectSkills,
		},
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}

	return toProtoReviewResponse(result), nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func toProtoReviewResponse(r *reviewdomain.ReviewResult) *reviewpb.ReviewResponse {
	var concerns []*reviewpb.ReviewConcern
	for _, c := range r.Concerns {
		concerns = append(concerns, &reviewpb.ReviewConcern{
			Severity:    c.Severity,
			Description: c.Description,
			Location:    c.Location,
			Suggestion:  c.Suggestion,
		})
	}

	return &reviewpb.ReviewResponse{
		Summary:     r.Summary,
		Strengths:   r.Strengths,
		Concerns:    concerns,
		Suggestions: r.Suggestions,
		Stats: &reviewpb.ReviewStats{
			FilesReviewed:  int32(r.Stats.FilesReviewed),
			ConcernsHigh:   int32(r.Stats.ConcernsHigh),
			ConcernsMedium: int32(r.Stats.ConcernsMedium),
			ConcernsLow:    int32(r.Stats.ConcernsLow),
		},
		ContextUsed: &reviewpb.ContextUsed{
			MemoryHits: r.ContextUsed.MemoryHits,
			SkillHits:  r.ContextUsed.SkillHits,
		},
		Model: r.Model,
	}
}
