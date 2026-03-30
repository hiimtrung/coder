package grpcserver

import (
	"context"

	"github.com/trungtran/coder/api/grpc/debugpb"
	debugdomain "github.com/trungtran/coder/internal/domain/debug"
	ucdebug "github.com/trungtran/coder/internal/usecase/debug"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// DebugServer implements debugpb.DebugServiceServer.
type DebugServer struct {
	debugpb.UnimplementedDebugServiceServer
	mgr *ucdebug.Manager
}

func NewDebugServer(mgr *ucdebug.Manager) *DebugServer {
	return &DebugServer{mgr: mgr}
}

// Debug analyses an error and returns a structured diagnosis.
func (s *DebugServer) Debug(ctx context.Context, req *debugpb.DebugRequest) (*debugpb.DebugResponse, error) {
	if req.ErrorMessage == "" {
		return nil, status.Error(codes.InvalidArgument, "error_message is required")
	}

	injectMemory := true
	injectSkills := true
	if req.Context != nil {
		injectMemory = req.Context.InjectMemory
		injectSkills = req.Context.InjectSkills
	}

	result, err := s.mgr.Debug(ctx, debugdomain.DebugRequest{
		ErrorMessage: req.ErrorMessage,
		FileContext:  req.FileContext,
		DiffContext:  req.DiffContext,
		Context: debugdomain.DebugContext{
			InjectMemory: injectMemory,
			InjectSkills: injectSkills,
		},
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}

	return toProtoDebugResponse(result), nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func toProtoDebugResponse(r *debugdomain.DebugResult) *debugpb.DebugResponse {
	return &debugpb.DebugResponse{
		RootCause:     r.RootCause,
		Location:      r.Location,
		Confidence:    r.Confidence,
		SuggestedFix:  r.SuggestedFix,
		SimilarIssues: r.SimilarIssues,
		FollowUp:      r.FollowUp,
		ContextUsed: &debugpb.ContextUsed{
			MemoryHits: r.ContextUsed.MemoryHits,
			SkillHits:  r.ContextUsed.SkillHits,
		},
		Model: r.Model,
	}
}
