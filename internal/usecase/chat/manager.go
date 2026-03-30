package chat

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	chatdomain "github.com/trungtran/coder/internal/domain/chat"
	memdomain "github.com/trungtran/coder/internal/domain/memory"
	skilldomain "github.com/trungtran/coder/internal/domain/skill"
)

const (
	defaultMemoryLimit  = 5
	defaultSkillLimit   = 3
	defaultHistoryLimit = 20
	contextTimeout      = 1500 * time.Millisecond

	baseSystemPrompt = `You are a senior software engineer AI assistant embedded in the developer's workflow.
Answer concisely and precisely. When suggesting code, follow the project's established patterns.
If you reference specific files or line numbers, be accurate.`
)

// Manager implements chatdomain.Manager.
type Manager struct {
	repo    chatdomain.Repository
	llm     chatdomain.LLMProvider
	memory  memdomain.MemoryManager
	skills  skilldomain.SkillUseCase
	model   string
	cfg     Config
}

// Config holds runtime knobs for the chat manager.
type Config struct {
	Model        string
	MemoryLimit  int
	SkillLimit   int
	HistoryLimit int
}

func NewManager(
	repo chatdomain.Repository,
	llm chatdomain.LLMProvider,
	memory memdomain.MemoryManager,
	skills skilldomain.SkillUseCase,
	cfg Config,
) *Manager {
	if cfg.MemoryLimit <= 0 {
		cfg.MemoryLimit = defaultMemoryLimit
	}
	if cfg.SkillLimit <= 0 {
		cfg.SkillLimit = defaultSkillLimit
	}
	if cfg.HistoryLimit <= 0 {
		cfg.HistoryLimit = defaultHistoryLimit
	}
	if cfg.Model == "" {
		cfg.Model = "qwen3.5:0.8b"
	}
	return &Manager{repo: repo, llm: llm, memory: memory, skills: skills, cfg: cfg}
}

// Chat runs the full context injection pipeline and returns a complete response.
func (m *Manager) Chat(ctx context.Context, clientID string, req chatdomain.ChatRequest) (*chatdomain.ChatResponse, error) {
	session, err := m.resolveSession(ctx, clientID, req.SessionID)
	if err != nil {
		return nil, err
	}

	messages, contextUsed, err := m.buildMessages(ctx, req)
	if err != nil {
		return nil, err
	}

	llmResp, err := m.llm.Chat(ctx, m.cfg.Model, messages)
	if err != nil {
		return nil, err
	}

	if err := m.persistTurn(ctx, session.ID, req.Message, llmResp); err != nil {
		return nil, err
	}
	m.autoTitle(ctx, session)

	return &chatdomain.ChatResponse{
		Reply:       llmResp.Content,
		SessionID:   session.ID,
		ContextUsed: contextUsed,
		Model:       m.cfg.Model,
		Tokens: chatdomain.TokenUsage{
			Prompt:     llmResp.TokensIn,
			Completion: llmResp.TokensOut,
		},
	}, nil
}

// ChatStream runs context injection + streaming. onDelta receives each token chunk.
func (m *Manager) ChatStream(ctx context.Context, clientID string, req chatdomain.ChatRequest, onDelta func(string)) (*chatdomain.ChatResponse, error) {
	session, err := m.resolveSession(ctx, clientID, req.SessionID)
	if err != nil {
		return nil, err
	}

	messages, contextUsed, err := m.buildMessages(ctx, req)
	if err != nil {
		return nil, err
	}

	llmResp, err := m.llm.ChatStream(ctx, m.cfg.Model, messages, onDelta)
	if err != nil {
		return nil, err
	}

	if err := m.persistTurn(ctx, session.ID, req.Message, llmResp); err != nil {
		return nil, err
	}
	m.autoTitle(ctx, session)

	return &chatdomain.ChatResponse{
		Reply:       llmResp.Content,
		SessionID:   session.ID,
		ContextUsed: contextUsed,
		Model:       m.cfg.Model,
		Tokens: chatdomain.TokenUsage{
			Prompt:     llmResp.TokensIn,
			Completion: llmResp.TokensOut,
		},
	}, nil
}

func (m *Manager) ListSessions(ctx context.Context, clientID string, limit int) ([]chatdomain.Session, error) {
	return m.repo.ListSessions(ctx, clientID, limit)
}

func (m *Manager) GetSession(ctx context.Context, id string) (*chatdomain.Session, []chatdomain.Message, error) {
	session, err := m.repo.GetSession(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	msgs, err := m.repo.GetMessages(ctx, id, 0) // 0 = all
	return session, msgs, err
}

func (m *Manager) DeleteSession(ctx context.Context, id string) error {
	return m.repo.DeleteSession(ctx, id)
}

// --- private helpers ---

// resolveSession returns the existing session or creates a new one.
func (m *Manager) resolveSession(ctx context.Context, clientID, sessionID string) (*chatdomain.Session, error) {
	if sessionID != "" {
		s, err := m.repo.GetSession(ctx, sessionID)
		if err == nil {
			return s, nil
		}
		// Session ID provided but not found — create fresh with that ID
	}

	now := time.Now()
	id := sessionID
	if id == "" {
		id = uuid.New().String()
	}
	s := &chatdomain.Session{
		ID:        id,
		ClientID:  clientID,
		Title:     "",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := m.repo.CreateSession(ctx, s); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	return s, nil
}

// buildMessages runs the context injection pipeline and assembles the messages array.
func (m *Manager) buildMessages(ctx context.Context, req chatdomain.ChatRequest) ([]chatdomain.LLMMessage, chatdomain.ContextUsed, error) {
	memLimit := m.cfg.MemoryLimit
	skillLimit := m.cfg.SkillLimit
	injectMem := true
	injectSkill := true

	if req.Context.MemoryLimit > 0 {
		memLimit = req.Context.MemoryLimit
	}
	if req.Context.SkillLimit > 0 {
		skillLimit = req.Context.SkillLimit
	}
	if !req.Context.InjectMemory {
		injectMem = false
	}
	if !req.Context.InjectSkills {
		injectSkill = false
	}

	var contextUsed chatdomain.ContextUsed
	var memContext, skillContext string

	// Step 1+2: Parallel context search with timeout
	type memResult struct {
		results []memdomain.SearchResult
		err     error
	}
	type skillResult struct {
		results []skilldomain.SkillSearchResult
		err     error
	}

	memCh := make(chan memResult, 1)
	skillCh := make(chan skillResult, 1)

	ctxTimeout, cancel := context.WithTimeout(ctx, contextTimeout)
	defer cancel()

	if injectMem && m.memory != nil {
		go func() {
			results, err := m.memory.Search(ctxTimeout, req.Message, "", nil, "", nil, memLimit)
			memCh <- memResult{results, err}
		}()
	} else {
		memCh <- memResult{}
	}

	if injectSkill && m.skills != nil {
		go func() {
			results, err := m.skills.SearchSkills(ctxTimeout, req.Message, skillLimit)
			skillCh <- skillResult{results, err}
		}()
	} else {
		skillCh <- skillResult{}
	}

	memRes := <-memCh
	skillRes := <-skillCh

	// Step 3: Build enriched system prompt
	var systemPrompt strings.Builder
	systemPrompt.WriteString(baseSystemPrompt)

	if len(skillRes.results) > 0 {
		systemPrompt.WriteString("\n\n## Relevant patterns and rules:\n")
		for _, r := range skillRes.results {
			for _, chunk := range r.Chunks {
				systemPrompt.WriteString(chunk.Content)
				systemPrompt.WriteString("\n")
			}
			contextUsed.SkillHits = append(contextUsed.SkillHits, r.Skill.Name)
		}
		skillContext = "injected"
		_ = skillContext
	}

	if len(memRes.results) > 0 {
		systemPrompt.WriteString("\n\n## Past decisions and learnings:\n")
		for _, r := range memRes.results {
			systemPrompt.WriteString(r.Title + ": " + r.Content)
			systemPrompt.WriteString("\n")
			contextUsed.MemoryHits = append(contextUsed.MemoryHits, r.Title)
		}
		memContext = "injected"
		_ = memContext
	}

	if req.Context.ExtraSystem != "" {
		systemPrompt.WriteString("\n\n")
		systemPrompt.WriteString(req.Context.ExtraSystem)
	}

	// Step 4: Build messages array = system + history + current message
	var messages []chatdomain.LLMMessage
	messages = append(messages, chatdomain.LLMMessage{
		Role:    "system",
		Content: systemPrompt.String(),
	})

	// Append session history
	if req.SessionID != "" {
		history, err := m.repo.GetMessages(ctx, req.SessionID, m.cfg.HistoryLimit)
		if err == nil {
			for _, msg := range history {
				if msg.Role == "system" {
					continue // skip stored system messages
				}
				messages = append(messages, chatdomain.LLMMessage{
					Role:    msg.Role,
					Content: msg.Content,
				})
			}
		}
	}

	messages = append(messages, chatdomain.LLMMessage{
		Role:    "user",
		Content: req.Message,
	})

	return messages, contextUsed, nil
}

// persistTurn saves the user message and assistant reply to the session.
func (m *Manager) persistTurn(ctx context.Context, sessionID, userMsg string, llmResp *chatdomain.LLMResponse) error {
	now := time.Now()

	if err := m.repo.AppendMessage(ctx, &chatdomain.Message{
		ID:        uuid.New().String(),
		SessionID: sessionID,
		Role:      "user",
		Content:   userMsg,
		CreatedAt: now,
	}); err != nil {
		return fmt.Errorf("failed to save user message: %w", err)
	}

	if err := m.repo.AppendMessage(ctx, &chatdomain.Message{
		ID:        uuid.New().String(),
		SessionID: sessionID,
		Role:      "assistant",
		Content:   llmResp.Content,
		TokensIn:  llmResp.TokensIn,
		TokensOut: llmResp.TokensOut,
		CreatedAt: now.Add(time.Millisecond), // ensure ordering
	}); err != nil {
		return fmt.Errorf("failed to save assistant message: %w", err)
	}

	// Update session timestamp
	s := &chatdomain.Session{
		ID:        sessionID,
		UpdatedAt: now,
	}
	return m.repo.UpdateSession(ctx, s)
}

// autoTitle sets the session title from the first user message if not yet set.
func (m *Manager) autoTitle(ctx context.Context, session *chatdomain.Session) {
	if session.Title != "" {
		return
	}
	msgs, err := m.repo.GetMessages(ctx, session.ID, 1)
	if err != nil || len(msgs) == 0 {
		return
	}
	title := msgs[0].Content
	if len(title) > 60 {
		title = title[:57] + "..."
	}
	session.Title = title
	session.UpdatedAt = time.Now()
	m.repo.UpdateSession(ctx, session)
}
