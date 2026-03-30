package grpcserver

import (
	"context"

	"github.com/trungtran/coder/api/grpc/chatpb"
	chatdomain "github.com/trungtran/coder/internal/domain/chat"
	authdomain "github.com/trungtran/coder/internal/domain/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ChatServer implements chatpb.ChatServiceServer using the chat.Manager use case.
type ChatServer struct {
	chatpb.UnimplementedChatServiceServer
	mgr chatdomain.Manager
}

func NewChatServer(mgr chatdomain.Manager) *ChatServer {
	return &ChatServer{mgr: mgr}
}

// Chat handles a blocking completion request.
func (s *ChatServer) Chat(ctx context.Context, req *chatpb.ChatRequest) (*chatpb.ChatResponse, error) {
	clientID := clientIDFromContext(ctx)

	domainReq := toDomainChatRequest(req)
	resp, err := s.mgr.Chat(ctx, clientID, domainReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}

	return &chatpb.ChatResponse{
		Reply:       resp.Reply,
		SessionId:   resp.SessionID,
		ContextUsed: toProtoContextUsed(resp.ContextUsed.MemoryHits, resp.ContextUsed.SkillHits),
		Model:       resp.Model,
		Tokens: &chatpb.TokenUsage{
			Prompt:     int32(resp.Tokens.Prompt),
			Completion: int32(resp.Tokens.Completion),
		},
	}, nil
}

// ChatStream handles a server-side streaming completion request.
// Token deltas are sent as they arrive; the final frame carries metadata.
func (s *ChatServer) ChatStream(req *chatpb.ChatRequest, stream chatpb.ChatService_ChatStreamServer) error {
	ctx := stream.Context()
	clientID := clientIDFromContext(ctx)

	domainReq := toDomainChatRequest(req)

	onDelta := func(delta string) {
		_ = stream.Send(&chatpb.ChatStreamResponse{Delta: delta})
	}

	resp, err := s.mgr.ChatStream(ctx, clientID, domainReq, onDelta)
	if err != nil {
		// Send error frame so the client can surface it cleanly.
		_ = stream.Send(&chatpb.ChatStreamResponse{Error: err.Error()})
		return status.Errorf(codes.Internal, "%v", err)
	}

	// Final frame: done=true with full metadata.
	return stream.Send(&chatpb.ChatStreamResponse{
		Done:      true,
		SessionId: resp.SessionID,
		ContextUsed: toProtoContextUsed(
			resp.ContextUsed.MemoryHits,
			resp.ContextUsed.SkillHits,
		),
		Tokens: &chatpb.TokenUsage{
			Prompt:     int32(resp.Tokens.Prompt),
			Completion: int32(resp.Tokens.Completion),
		},
	})
}

// ListSessions returns recent sessions for the authenticated client.
func (s *ChatServer) ListSessions(ctx context.Context, req *chatpb.ListSessionsRequest) (*chatpb.ListSessionsResponse, error) {
	clientID := clientIDFromContext(ctx)
	limit := int(req.Limit)
	if limit <= 0 {
		limit = 50
	}

	sessions, err := s.mgr.ListSessions(ctx, clientID, limit)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}

	var pbSessions []*chatpb.Session
	for _, sess := range sessions {
		pbSessions = append(pbSessions, toProtoSession(sess))
	}
	return &chatpb.ListSessionsResponse{Sessions: pbSessions}, nil
}

// GetSession returns a session and its message history.
func (s *ChatServer) GetSession(ctx context.Context, req *chatpb.GetSessionRequest) (*chatpb.GetSessionResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}

	sess, msgs, err := s.mgr.GetSession(ctx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "%v", err)
	}

	var pbMsgs []*chatpb.Message
	for _, m := range msgs {
		pbMsgs = append(pbMsgs, &chatpb.Message{
			Id:        m.ID,
			SessionId: m.SessionID,
			Role:      m.Role,
			Content:   m.Content,
			TokensIn:  int32(m.TokensIn),
			TokensOut: int32(m.TokensOut),
			CreatedAt: timestamppb.New(m.CreatedAt),
		})
	}

	return &chatpb.GetSessionResponse{
		Session:  toProtoSession(*sess),
		Messages: pbMsgs,
	}, nil
}

// DeleteSession removes a session and all its messages.
func (s *ChatServer) DeleteSession(ctx context.Context, req *chatpb.DeleteSessionRequest) (*chatpb.DeleteSessionResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}

	if err := s.mgr.DeleteSession(ctx, req.Id); err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return &chatpb.DeleteSessionResponse{Success: true}, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

// clientIDFromContext extracts the client ID injected by the auth interceptor.
// Falls back to "anonymous" in open mode.
func clientIDFromContext(ctx context.Context) string {
	client := authdomain.ClientFromContext(ctx)
	if client != nil && client.ID != "" {
		return client.ID
	}
	return "anonymous"
}

func toDomainChatRequest(req *chatpb.ChatRequest) chatdomain.ChatRequest {
	domainReq := chatdomain.ChatRequest{
		Message:   req.Message,
		SessionID: req.SessionId,
		Context: chatdomain.ChatContext{
			InjectMemory: true,
			InjectSkills: true,
		},
	}
	if req.Context != nil {
		domainReq.Context = chatdomain.ChatContext{
			InjectMemory: req.Context.InjectMemory,
			InjectSkills: req.Context.InjectSkills,
			MemoryLimit:  int(req.Context.MemoryLimit),
			SkillLimit:   int(req.Context.SkillLimit),
			ExtraSystem:  req.Context.ExtraSystem,
		}
	}
	return domainReq
}

func toProtoSession(s chatdomain.Session) *chatpb.Session {
	return &chatpb.Session{
		Id:        s.ID,
		ClientId:  s.ClientID,
		Title:     s.Title,
		CreatedAt: timestamppb.New(s.CreatedAt),
		UpdatedAt: timestamppb.New(s.UpdatedAt),
	}
}

func toProtoContextUsed(memHits, skillHits []string) *chatpb.ContextUsed {
	return &chatpb.ContextUsed{
		MemoryHits: memHits,
		SkillHits:  skillHits,
	}
}
