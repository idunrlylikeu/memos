package v1

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	"github.com/lithammer/shortuuid/v4"
	"github.com/tmc/langchaingo/tools"

	"github.com/usememos/memos/plugin/vectorstore"
	"github.com/usememos/memos/server/auth"
	"github.com/usememos/memos/store"
)

// ─────────────────────────────────────────────────────────────────────────────
// Constants
// ─────────────────────────────────────────────────────────────────────────────

const (
	// compactThreshold is the total character count of messages that triggers compaction.
	// Roughly 80% of a 128k-token context window (4 chars ≈ 1 token).
	compactThreshold = 400_000

	// keepRecentMessages is the number of recent messages to keep verbatim after compaction.
	keepRecentMessages = 10

	// maxAgentRounds caps the number of tool-use iterations per request.
	maxAgentRounds = 6
)

// ─────────────────────────────────────────────────────────────────────────────
// Request / Response types
// ─────────────────────────────────────────────────────────────────────────────

type chatRequest struct {
	Content   string `json:"content"`   // user message text
	TagFilter string `json:"tagFilter"` // optional "#golang" etc.
}

type sessionRequest struct {
	Title string `json:"title"`
}

type sessionResponse struct {
	UID       string `json:"uid"`
	Title     string `json:"title"`
	CreatedTs int64  `json:"createdTs"`
	UpdatedTs int64  `json:"updatedTs"`
}

type messageResponse struct {
	ID        int32  `json:"id"`
	Role      string `json:"role"`
	Content   string `json:"content"`
	ToolName  string `json:"toolName,omitempty"`
	CreatedTs int64  `json:"createdTs"`
}

// ─────────────────────────────────────────────────────────────────────────────
// Route registration (called from v1.go)
// ─────────────────────────────────────────────────────────────────────────────

func (s *APIV1Service) registerAIChatRoutes(e *echo.Echo) {
	g := e.Group("/api/v1/ai")
	g.GET("/sessions", s.listAIChatSessions)
	g.POST("/sessions", s.createAIChatSession)
	g.PATCH("/sessions/:uid", s.updateAIChatSession)
	g.DELETE("/sessions/:uid", s.deleteAIChatSession)
	g.GET("/sessions/:uid/messages", s.listAIChatMessages)
	g.POST("/sessions/:uid/chat", s.handleAIChat)
}

// ─────────────────────────────────────────────────────────────────────────────
// Session CRUD
// ─────────────────────────────────────────────────────────────────────────────

func (s *APIV1Service) listAIChatSessions(c *echo.Context) error {
	user, err := s.requireAuth(c)
	if err != nil {
		return err
	}
	sessions, err := s.Store.ListAIChatSessions(c.Request().Context(), &store.FindAIChatSession{
		CreatorID: &user.ID,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	resp := make([]sessionResponse, 0, len(sessions))
	for _, sess := range sessions {
		resp = append(resp, sessionResponse{
			UID:       sess.UID,
			Title:     sess.Title,
			CreatedTs: sess.CreatedTs,
			UpdatedTs: sess.UpdatedTs,
		})
	}
	return c.JSON(http.StatusOK, resp)
}

func (s *APIV1Service) createAIChatSession(c *echo.Context) error {
	user, err := s.requireAuth(c)
	if err != nil {
		return err
	}
	var req sessionRequest
	if err := c.Bind(&req); err != nil {
		req.Title = "New Chat"
	}
	if req.Title == "" {
		req.Title = "New Chat"
	}
	sess, err := s.Store.CreateAIChatSession(c.Request().Context(), &store.AIChatSession{
		UID:       uuid.New().String()[:8],
		CreatorID: user.ID,
		Title:     req.Title,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusCreated, sessionResponse{
		UID:       sess.UID,
		Title:     sess.Title,
		CreatedTs: sess.CreatedTs,
		UpdatedTs: sess.UpdatedTs,
	})
}

func (s *APIV1Service) updateAIChatSession(c *echo.Context) error {
	uid := c.Param("uid")
	user, err := s.requireAuth(c)
	if err != nil {
		return err
	}
	// verify ownership
	sess, err := s.Store.GetAIChatSession(c.Request().Context(), &store.FindAIChatSession{UID: &uid})
	if err != nil || sess == nil || sess.CreatorID != user.ID {
		return echo.NewHTTPError(http.StatusNotFound, "session not found")
	}

	var req sessionRequest
	if err := c.Bind(&req); err != nil || req.Title == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "title required")
	}
	updated, err := s.Store.UpdateAIChatSession(c.Request().Context(), &store.UpdateAIChatSession{
		UID:   uid,
		Title: &req.Title,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, sessionResponse{
		UID:       updated.UID,
		Title:     updated.Title,
		UpdatedTs: updated.UpdatedTs,
	})
}

func (s *APIV1Service) deleteAIChatSession(c *echo.Context) error {
	uid := c.Param("uid")
	user, err := s.requireAuth(c)
	if err != nil {
		return err
	}
	sess, err := s.Store.GetAIChatSession(c.Request().Context(), &store.FindAIChatSession{UID: &uid})
	if err != nil || sess == nil || sess.CreatorID != user.ID {
		return echo.NewHTTPError(http.StatusNotFound, "session not found")
	}
	if err := s.Store.DeleteAIChatSession(c.Request().Context(), uid); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.NoContent(http.StatusNoContent)
}

func (s *APIV1Service) listAIChatMessages(c *echo.Context) error {
	uid := c.Param("uid")
	user, err := s.requireAuth(c)
	if err != nil {
		return err
	}
	sess, err := s.Store.GetAIChatSession(c.Request().Context(), &store.FindAIChatSession{UID: &uid})
	if err != nil || sess == nil || sess.CreatorID != user.ID {
		return echo.NewHTTPError(http.StatusNotFound, "session not found")
	}
	msgs, err := s.Store.ListAIChatMessages(c.Request().Context(), &store.FindAIChatMessage{
		SessionID: sess.ID,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	resp := make([]messageResponse, 0, len(msgs))
	for _, m := range msgs {
		resp = append(resp, messageResponse{
			ID:        m.ID,
			Role:      m.Role,
			Content:   m.Content,
			ToolName:  m.ToolName,
			CreatedTs: m.CreatedTs,
		})
	}
	return c.JSON(http.StatusOK, resp)
}

// ─────────────────────────────────────────────────────────────────────────────
// Main chat handler (SSE)
// ─────────────────────────────────────────────────────────────────────────────

func (s *APIV1Service) handleAIChat(c *echo.Context) error {
	if s.Profile.OpenRouterAPIKey == "" {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "AI chat is not configured (missing OPENROUTER_API_KEY)")
	}

	uid := c.Param("uid")
	user, err := s.requireAuth(c)
	if err != nil {
		return err
	}

	var req chatRequest
	if err := c.Bind(&req); err != nil || strings.TrimSpace(req.Content) == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "content required")
	}

	ctx := c.Request().Context()

	// ── 1. Load session ──────────────────────────────────────────────────────
	sess, err := s.Store.GetAIChatSession(ctx, &store.FindAIChatSession{UID: &uid})
	if err != nil || sess == nil || sess.CreatorID != user.ID {
		return echo.NewHTTPError(http.StatusNotFound, "session not found")
	}

	// ── 2. Load history from DB ───────────────────────────────────────────────
	dbMsgs, err := s.Store.ListAIChatMessages(ctx, &store.FindAIChatMessage{SessionID: sess.ID})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// ── 3. Context compaction ─────────────────────────────────────────────────
	dbMsgs, sess, err = s.maybeCompact(ctx, sess, dbMsgs, user.ID)
	if err != nil {
		slog.Warn("context compaction failed", "err", err)
	}

	// ── 4. Set up SSE ─────────────────────────────────────────────────────────
	rw := c.Response()
	rw.Header().Set("Content-Type", "text/event-stream")
	rw.Header().Set("Cache-Control", "no-cache")
	rw.Header().Set("Connection", "keep-alive")
	rw.Header().Set("X-Accel-Buffering", "no")
	rw.WriteHeader(http.StatusOK)

	emit := func(eventType, payload string) {
		data, _ := json.Marshal(map[string]string{"type": eventType, "content": payload})
		fmt.Fprintf(rw, "data: %s\n\n", data)
		if f, ok := rw.(http.Flusher); ok {
			f.Flush()
		}
	}
	emitJSON := func(eventType string, obj any) {
		inner, _ := json.Marshal(obj)
		data, _ := json.Marshal(map[string]json.RawMessage{
			"type":    json.RawMessage(`"` + eventType + `"`),
			"payload": inner,
		})
		fmt.Fprintf(rw, "data: %s\n\n", data)
		if f, ok := rw.(http.Flusher); ok {
			f.Flush()
		}
	}

	// ── 5. Persist user message ───────────────────────────────────────────────
	if _, err := s.Store.CreateAIChatMessage(ctx, &store.CreateAIChatMessage{
		SessionID:  sess.ID,
		Role:       "user",
		Content:    req.Content,
		TokenCount: int32(len(req.Content) / 4),
	}); err != nil {
		slog.Warn("failed to persist user message", "err", err)
	}

	// ── 6. Auto-title on first message ───────────────────────────────────────
	if len(dbMsgs) == 0 && sess.Title == "New Chat" {
		go s.autoTitleSession(context.Background(), sess.UID, req.Content)
	}

	// ── 7-11. Native OpenRouter function-calling agent loop ───────────────────
	// We bypass langchaingo's brittle text-based ReAct agent and call OpenRouter
	// directly using the OpenAI-compatible `tools` API, which is reliable on any
	// function-capable model.

	// Build our tool registry (same tools as before, but now dispatched natively)
	toolRegistry := map[string]tools.Tool{
		"search_memos":       newSearchMemosTool(s.VectorStore, user.ID, req.TagFilter),
		"query_memos":        newQueryMemosTool(s.Store, user.ID),
		"create_memo":        newCreateMemoTool(s.Store, user.ID),
		"append_to_memo":     newAppendToMemoTool(s.Store, user.ID),
		"update_memo":        newUpdateMemoTool(s.Store, user.ID),
		"update_memo_tags":   newUpdateMemoTagsTool(s.Store, user.ID),
		"delete_memo":        newDeleteMemoTool(s.Store, user.ID),
		"get_user_stats":     newGetUserStatsTool(s.Store, user.ID),
		"list_memos_by_tag":  newListMemosByTagTool(s.Store, user.ID),
	}

	// Tool schema definitions sent to the LLM
	toolDefs := []map[string]any{
		buildToolDef("search_memos", "Search the user's notes semantically for a concept or topic. Use for general/conceptual questions.", map[string]any{
			"query": map[string]any{"type": "string", "description": "The search query"},
		}, []string{"query"}),
		buildToolDef("query_memos", "Search the user's notes by exact date range or keyword. ALWAYS use this for date-specific questions like 'what did I post on Jan 26'.", map[string]any{
			"text_search": map[string]any{"type": "string", "description": "Exact keyword to search (optional)"},
			"date_start":  map[string]any{"type": "string", "description": "Start date in YYYY-MM-DD (optional)"},
			"date_end":    map[string]any{"type": "string", "description": "End date in YYYY-MM-DD (optional)"},
		}, []string{}),
		buildToolDef("create_memo", "Create a new note for the user.", map[string]any{
			"content": map[string]any{"type": "string", "description": "The content of the new note"},
		}, []string{"content"}),
		buildToolDef("append_to_memo", "Append text to an existing note without overwriting it.", map[string]any{
			"uid":     map[string]any{"type": "string", "description": "Note UID"},
			"content": map[string]any{"type": "string", "description": "Text to append"},
		}, []string{"uid", "content"}),
		buildToolDef("update_memo", "Fully rewrite the content of an existing note.", map[string]any{
			"uid":     map[string]any{"type": "string", "description": "Note UID"},
			"content": map[string]any{"type": "string", "description": "New content"},
		}, []string{"uid", "content"}),
		buildToolDef("update_memo_tags", "Add hashtags to an existing note.", map[string]any{
			"uid":      map[string]any{"type": "string", "description": "Note UID"},
			"new_tags": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Tags to add, e.g. ['#dev','#work']"},
		}, []string{"uid", "new_tags"}),
		buildToolDef("delete_memo", "Permanently delete a note.", map[string]any{
			"uid": map[string]any{"type": "string", "description": "Note UID"},
		}, []string{"uid"}),
		buildToolDef("get_user_stats", "Get note statistics (total count, etc). No parameters needed.", map[string]any{}, []string{}),
		buildToolDef("list_memos_by_tag", "List all notes tagged with a specific hashtag.", map[string]any{
			"tag": map[string]any{"type": "string", "description": "Tag including hash, e.g. '#work'"},
		}, []string{"tag"}),
	}

	// Build message history
	systemText := buildSystemPrompt(sess.Summary, time.Now())
	messages := []map[string]any{
		{"role": "system", "content": systemText},
	}
	for _, m := range dbMsgs {
		if m.Role == "user" || m.Role == "assistant" {
			messages = append(messages, map[string]any{"role": m.Role, "content": m.Content})
		}
	}
	messages = append(messages, map[string]any{"role": "user", "content": req.Content})

	slog.Info("[AGENT INIT]", "model", s.Profile.AIModel, "tools", len(toolDefs))
	slog.Info("[AGENT PROMPT]", "input", req.Content)

	var finalAnswer string

	for round := 0; round < maxAgentRounds; round++ {
		// Call OpenRouter
		reqBody := map[string]any{
			"model":    s.Profile.AIModel,
			"messages": messages,
			"tools":    toolDefs,
		}
		bodyBytes, _ := json.Marshal(reqBody)

		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
			"https://openrouter.ai/api/v1/chat/completions",
			bytes.NewReader(bodyBytes))
		if err != nil {
			emit("error", "failed to build request: "+err.Error())
			break
		}
		httpReq.Header.Set("Authorization", "Bearer "+s.Profile.OpenRouterAPIKey)
		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(httpReq)
		if err != nil {
			emit("error", "LLM request failed: "+err.Error())
			break
		}
		var apiResp struct {
			Choices []struct {
				Message struct {
					Role      string          `json:"role"`
					Content   string          `json:"content"`
					ToolCalls []struct {
						ID       string `json:"id"`
						Type     string `json:"type"`
						Function struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						} `json:"function"`
					} `json:"tool_calls"`
				} `json:"message"`
			} `json:"choices"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil || len(apiResp.Choices) == 0 {
			resp.Body.Close()
			emit("error", "failed to decode LLM response")
			break
		}
		resp.Body.Close()

		msg := apiResp.Choices[0].Message

		// No tool calls → final text answer
		if len(msg.ToolCalls) == 0 {
			finalAnswer = msg.Content
			slog.Info("[AGENT FINISH]", "answer", finalAnswer)
			break
		}

		// Append assistant's tool-call message to context
		assistantMsg := map[string]any{
			"role":       "assistant",
			"content":    msg.Content,
			"tool_calls": msg.ToolCalls,
		}
		messages = append(messages, assistantMsg)

		// Execute each tool call and append results
		// Deduplicate calls — some models repeat the same tool_call_id in one response
		seenCallIDs := make(map[string]bool)
		for _, tc := range msg.ToolCalls {
			if seenCallIDs[tc.ID] {
				continue
			}
			seenCallIDs[tc.ID] = true
			toolName := tc.Function.Name
			toolInput := tc.Function.Arguments

			slog.Info("[AGENT TOOL CALL]", "tool", toolName, "input", toolInput)
			emitJSON("tool_call", map[string]string{"name": toolName, "input": toolInput})

			var toolResult string
			if t, ok := toolRegistry[toolName]; ok {
				toolResult, err = t.Call(ctx, toolInput)
				if err != nil {
					toolResult = "Error: " + err.Error()
				}
			} else {
				toolResult = "Unknown tool: " + toolName
			}
			slog.Info("[AGENT TOOL RESULT]", "tool", toolName, "result", toolResult)

			messages = append(messages, map[string]any{
				"role":         "tool",
				"tool_call_id": tc.ID,
				"content":      toolResult,
			})
		}
	}

	slog.Info("[AGENT RAW RESULT]", "answer", finalAnswer)

	if finalAnswer != "" {
		for _, word := range strings.Fields(finalAnswer) {
			emit("token", word+" ")
			time.Sleep(8 * time.Millisecond)
		}
	}

	// ── 11. Persist assistant answer ──────────────────────────────────────────
	if finalAnswer != "" {
		if _, err := s.Store.CreateAIChatMessage(ctx, &store.CreateAIChatMessage{
			SessionID:  sess.ID,
			Role:       "assistant",
			Content:    finalAnswer,
			TokenCount: int32(len(finalAnswer) / 4),
		}); err != nil {
			slog.Warn("failed to persist assistant message", "err", err)
		}
	}

	// ── 12. Emit source citations from vector search results ──────────────────
	if s.VectorStore != nil {
		sources, _ := s.VectorStore.SearchSimilar(ctx, user.ID, req.Content, 3)
		for _, src := range sources {
			emitJSON("source", map[string]any{
				"memo_uid": src.MemoUID,
				"snippet":  src.Content[:min(200, len(src.Content))],
			})
		}
	}

	// ── 13. Update session timestamp ──────────────────────────────────────────
	empty := ""
	_, _ = s.Store.UpdateAIChatSession(ctx, &store.UpdateAIChatSession{
		UID:     uid,
		Title:   nil,
		Summary: &empty,
	})

	emit("done", uid)

	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Context compaction
// ─────────────────────────────────────────────────────────────────────────────

// maybeCompact summarises older messages when the total character count exceeds
// compactThreshold, keeping only the most recent keepRecentMessages verbatim.
func (s *APIV1Service) maybeCompact(
	ctx context.Context,
	sess *store.AIChatSession,
	msgs []*store.AIChatMessage,
	userID int32,
) ([]*store.AIChatMessage, *store.AIChatSession, error) {
	if s.Profile.OpenRouterAPIKey == "" {
		return msgs, sess, nil
	}

	total := 0
	for _, m := range msgs {
		total += len(m.Content)
	}
	if total <= compactThreshold {
		return msgs, sess, nil
	}

	// Split: old = everything except last keepRecentMessages
	cutAt := len(msgs) - keepRecentMessages
	if cutAt <= 0 {
		return msgs, sess, nil
	}
	old := msgs[:cutAt]
	recent := msgs[cutAt:]

	// Build a prompt for the summarisation model
	var sb strings.Builder
	sb.WriteString("Summarise this conversation concisely, preserving key facts and decisions:\n\n")
	for _, m := range old {
		sb.WriteString(m.Role + ": " + m.Content + "\n")
	}

	// Call OpenRouter directly to summarize the old messages
	summary, err := s.callLLM(ctx, sb.String())
	if err != nil {
		return msgs, sess, err
	}

	// Persist summary & delete old messages
	// Add existing summary as prefix
	existingSummary := sess.Summary
	fullSummary := summary
	if existingSummary != "" {
		fullSummary = existingSummary + "\n\n" + summary
	}

	updatedSess, err := s.Store.UpdateAIChatSession(ctx, &store.UpdateAIChatSession{
		UID:     sess.UID,
		Summary: &fullSummary,
	})
	if err != nil {
		return msgs, sess, err
	}

	// Delete only the compacted messages (the old ones) by deleting all and re-inserting recent
	if err := s.Store.DeleteAIChatMessages(ctx, sess.ID); err != nil {
		return msgs, sess, err
	}
	for _, m := range recent {
		_, _ = s.Store.CreateAIChatMessage(ctx, &store.CreateAIChatMessage{
			SessionID:  sess.ID,
			Role:       m.Role,
			Content:    m.Content,
			ToolName:   m.ToolName,
			TokenCount: m.TokenCount,
		})
	}

	slog.Info("context compacted", "session", sess.UID, "summary_len", len(fullSummary), "kept_messages", len(recent))
	return recent, updatedSess, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Auto-title
// ─────────────────────────────────────────────────────────────────────────────

func (s *APIV1Service) autoTitleSession(ctx context.Context, uid, firstMessage string) {
	if s.Profile.OpenRouterAPIKey == "" {
		return
	}
	prompt := fmt.Sprintf(
		"Generate a short (5-7 word) title for a chat that starts with:\n\"%s\"\nReturn only the title, no quotes.",
		firstMessage,
	)
	title, err := s.callLLM(ctx, prompt)
	if err != nil || strings.TrimSpace(title) == "" {
		return
	}
	title = strings.TrimSpace(title)
	_, _ = s.Store.UpdateAIChatSession(ctx, &store.UpdateAIChatSession{UID: uid, Title: &title})
}

// ─────────────────────────────────────────────────────────────────────────────
// Helper: requireAuth (convenience wrapper)
// ─────────────────────────────────────────────────────────────────────────────

func (s *APIV1Service) requireAuth(c *echo.Context) (*store.User, error) {
	authHeader := c.Request().Header.Get("Authorization")
	cookieHeader := c.Request().Header.Get("Cookie")
	user, err := auth.NewAuthenticator(s.Store, s.Secret).AuthenticateToUser(
		c.Request().Context(), authHeader, cookieHeader,
	)
	if err != nil || user == nil {
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}
	return user, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Helper: SearchMemos tool
// ─────────────────────────────────────────────────────────────────────────────

type searchMemosTool struct {
	vs        *vectorstore.Store
	userID    int32
	tagFilter string
}

func newSearchMemosTool(vs *vectorstore.Store, userID int32, tagFilter string) tools.Tool {
	return &searchMemosTool{vs: vs, userID: userID, tagFilter: tagFilter}
}

func (t *searchMemosTool) Name() string { return "search_memos" }
func (t *searchMemosTool) Description() string {
	return "Search through the user's personal notes (memos) for relevant information. Input should be a search query."
}
func (t *searchMemosTool) Call(ctx context.Context, input string) (string, error) {
	slog.Info("[AGENT TOOL CALL]", "tool", t.Name(), "input", input)
	if t.vs == nil {
		return "Vector store not available.", nil
	}
	results, err := t.vs.SearchSimilar(ctx, t.userID, input, 5)
	if err != nil {
		return "", err
	}
	if len(results) == 0 {
		return "No relevant notes found.", nil
	}
	var sb strings.Builder
	for i, r := range results {
		preview := r.Content
		if len(preview) > 400 {
			preview = preview[:400] + "..."
		}
		sb.WriteString(fmt.Sprintf("[%d] Note %s (score %.2f):\n%s\n\n", i+1, r.MemoUID, r.Score, preview))
	}
	return sb.String(), nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Helper: UpdateMemo tool
// ─────────────────────────────────────────────────────────────────────────────

type updateMemoTool struct {
	store  *store.Store
	userID int32
}

func newUpdateMemoTool(store *store.Store, userID int32) tools.Tool {
	return &updateMemoTool{store: store, userID: userID}
}

func (t *updateMemoTool) Name() string { return "update_memo" }
func (t *updateMemoTool) Description() string {
	return "Update an existing note (memo). Input should be JSON string with keys `uid` (string) and `content` (string)."
}
func (t *updateMemoTool) Call(ctx context.Context, input string) (string, error) {
	slog.Info("[AGENT TOOL CALL]", "tool", t.Name(), "input", input)
	var payload struct {
		UID     string `json:"uid"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal([]byte(input), &payload); err != nil {
		return "Error: failed to parse input JSON.", nil
	}
	
	m, err := t.store.GetMemo(ctx, &store.FindMemo{UID: &payload.UID})
	if err != nil || m == nil {
		return "Error: note not found.", nil
	}
	if m.CreatorID != t.userID {
		return "Error: unauthorized to update this note.", nil
	}

	err = t.store.UpdateMemo(ctx, &store.UpdateMemo{
		ID:      m.ID,
		Content: &payload.Content,
	})
	if err != nil {
		return "Error: " + err.Error(), nil
	}
	return "Note successfully updated.", nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Helper: CreateMemo tool
// ─────────────────────────────────────────────────────────────────────────────

type createMemoTool struct {
	store  *store.Store
	userID int32
}

func newCreateMemoTool(store *store.Store, userID int32) tools.Tool {
	return &createMemoTool{store: store, userID: userID}
}

func (t *createMemoTool) Name() string { return "create_memo" }
func (t *createMemoTool) Description() string {
	return "Draft and save a brand new note for the user. Input must be a JSON string with key `content` (string)."
}
func (t *createMemoTool) Call(ctx context.Context, input string) (string, error) {
	slog.Info("[AGENT TOOL CALL]", "tool", t.Name(), "input", input)
	var payload struct {
		Content string `json:"content"`
	}
	if err := json.Unmarshal([]byte(input), &payload); err != nil {
		return "Error: failed to parse input JSON.", nil
	}
	
	// Use the same shortuuid format that Memos uses for all memo UIDs
	uid := shortuuid.New()
	_, err := t.store.CreateMemo(ctx, &store.Memo{
		UID:        uid,
		CreatorID:  t.userID,
		Content:    payload.Content,
		Visibility: store.Private,
	})
	if err != nil {
		return "Error creating note: " + err.Error(), nil
	}
	return fmt.Sprintf("Note successfully created with UID: %s", uid), nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Helper: AppendToMemo tool
// ─────────────────────────────────────────────────────────────────────────────

type appendToMemoTool struct {
	store  *store.Store
	userID int32
}

func newAppendToMemoTool(store *store.Store, userID int32) tools.Tool {
	return &appendToMemoTool{store: store, userID: userID}
}

func (t *appendToMemoTool) Name() string { return "append_to_memo" }
func (t *appendToMemoTool) Description() string {
	return "Add a new thought or bullet point to the bottom of an existing note instead of overwriting it. Input must be a JSON string with keys `uid` (string) and `content` (string)."
}
func (t *appendToMemoTool) Call(ctx context.Context, input string) (string, error) {
	slog.Info("[AGENT TOOL CALL]", "tool", t.Name(), "input", input)
	var payload struct {
		UID     string `json:"uid"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal([]byte(input), &payload); err != nil {
		return "Error: failed to parse input JSON.", nil
	}
	
	m, err := t.store.GetMemo(ctx, &store.FindMemo{UID: &payload.UID})
	if err != nil || m == nil {
		return "Error: note not found.", nil
	}
	if m.CreatorID != t.userID {
		return "Error: unauthorized to modify this note.", nil
	}

	newContent := m.Content + "\n\n" + payload.Content
	err = t.store.UpdateMemo(ctx, &store.UpdateMemo{
		ID:      m.ID,
		Content: &newContent,
	})
	if err != nil {
		return "Error appending to note: " + err.Error(), nil
	}
	return "Content successfully appended to note.", nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Helper: UpdateMemoTags tool
// ─────────────────────────────────────────────────────────────────────────────

type updateMemoTagsTool struct {
	store  *store.Store
	userID int32
}

func newUpdateMemoTagsTool(store *store.Store, userID int32) tools.Tool {
	return &updateMemoTagsTool{store: store, userID: userID}
}

func (t *updateMemoTagsTool) Name() string { return "update_memo_tags" }
func (t *updateMemoTagsTool) Description() string {
	return "Adds or modifies hashtag properties dynamically within an existing note's markdown body. Input must be a JSON string with keys `uid` (string) and `new_tags` (string array like [\"#dev\", \"#journal\"])."
}
func (t *updateMemoTagsTool) Call(ctx context.Context, input string) (string, error) {
	slog.Info("[AGENT TOOL CALL]", "tool", t.Name(), "input", input)
	var payload struct {
		UID     string   `json:"uid"`
		NewTags []string `json:"new_tags"`
	}
	if err := json.Unmarshal([]byte(input), &payload); err != nil {
		return "Error: failed to parse input JSON.", nil
	}
	
	m, err := t.store.GetMemo(ctx, &store.FindMemo{UID: &payload.UID})
	if err != nil || m == nil {
		return "Error: note not found.", nil
	}
	if m.CreatorID != t.userID {
		return "Error: unauthorized to modify this note.", nil
	}

	newContent := m.Content + "\n\n" + strings.Join(payload.NewTags, " ")
	err = t.store.UpdateMemo(ctx, &store.UpdateMemo{
		ID:      m.ID,
		Content: &newContent,
	})
	if err != nil {
		return "Error appending tags: " + err.Error(), nil
	}
	return "Tags successfully added to the note body.", nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Helper: DeleteMemo tool
// ─────────────────────────────────────────────────────────────────────────────

type deleteMemoTool struct {
	store  *store.Store
	userID int32
}

func newDeleteMemoTool(store *store.Store, userID int32) tools.Tool {
	return &deleteMemoTool{store: store, userID: userID}
}

func (t *deleteMemoTool) Name() string { return "delete_memo" }
func (t *deleteMemoTool) Description() string {
	return "Permanently deletes a specific note. Input must be a JSON string with key `uid` (string)."
}
func (t *deleteMemoTool) Call(ctx context.Context, input string) (string, error) {
	slog.Info("[AGENT TOOL CALL]", "tool", t.Name(), "input", input)
	var payload struct {
		UID string `json:"uid"`
	}
	if err := json.Unmarshal([]byte(input), &payload); err != nil {
		return "Error: failed to parse input JSON.", nil
	}
	
	m, err := t.store.GetMemo(ctx, &store.FindMemo{UID: &payload.UID})
	if err != nil || m == nil {
		return "Error: note not found.", nil
	}
	if m.CreatorID != t.userID {
		return "Error: unauthorized to access this note.", nil
	}

	err = t.store.DeleteMemo(ctx, &store.DeleteMemo{ID: m.ID})
	if err != nil {
		return "Error deleting note: " + err.Error(), nil
	}
	return "Note successfully and permanently deleted.", nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Helper: GetUserStats tool
// ─────────────────────────────────────────────────────────────────────────────

type getUserStatsTool struct {
	store  *store.Store
	userID int32
}

func newGetUserStatsTool(store *store.Store, userID int32) tools.Tool {
	return &getUserStatsTool{store: store, userID: userID}
}

func (t *getUserStatsTool) Name() string { return "get_user_stats" }
func (t *getUserStatsTool) Description() string {
	return "Gets general statistics about the user's Memos account, like total active notes created. Input should be an empty JSON object: {}"
}
func (t *getUserStatsTool) Call(ctx context.Context, input string) (string, error) {
	slog.Info("[AGENT TOOL CALL]", "tool", t.Name(), "input", input)
	state := store.Normal
	memos, err := t.store.ListMemos(ctx, &store.FindMemo{
		CreatorID: &t.userID,
		RowStatus: &state,
		ExcludeContent: true,
	})
	if err != nil {
		return "Error retrieving stats: " + err.Error(), nil
	}
	
	return fmt.Sprintf("User Statistics:\nTotal Active Memos: %d", len(memos)), nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Helper: ListMemosByTag tool
// ─────────────────────────────────────────────────────────────────────────────

type listMemosByTagTool struct {
	store  *store.Store
	userID int32
}

func newListMemosByTagTool(store *store.Store, userID int32) tools.Tool {
	return &listMemosByTagTool{store: store, userID: userID}
}

func (t *listMemosByTagTool) Name() string { return "list_memos_by_tag" }
func (t *listMemosByTagTool) Description() string {
	return "Retrieves a strict list of notes that contain a specific tag. Input must be a JSON string with key `tag` (string, including the hash, e.g., '#ideas'). Useful when semantic vector search isn't explicit enough."
}
func (t *listMemosByTagTool) Call(ctx context.Context, input string) (string, error) {
	slog.Info("[AGENT TOOL CALL]", "tool", t.Name(), "input", input)
	var payload struct {
		Tag string `json:"tag"`
	}
	if err := json.Unmarshal([]byte(input), &payload); err != nil {
		return "Error: failed to parse input JSON.", nil
	}
	
	find := &store.FindMemo{
		CreatorID: &t.userID,
		ExcludeComments: true,
	}
	find.Filters = append(find.Filters, fmt.Sprintf("content.contains('%s')", strings.ReplaceAll(payload.Tag, "'", "\\'")))
	
	memos, err := t.store.ListMemos(ctx, find)
	if err != nil {
		return "Error searching tags: " + err.Error(), nil
	}
	
	if len(memos) == 0 {
		return fmt.Sprintf("No notes found with the tag %s.", payload.Tag), nil
	}
	
	var sb strings.Builder
	for i, r := range memos {
		if i >= 10 {
			sb.WriteString(fmt.Sprintf("... and %d more tagged notes.", len(memos)-10))
			break
		}
		preview := r.Content
		if len(preview) > 300 {
			preview = preview[:300] + "..."
		}
		sb.WriteString(fmt.Sprintf("[%d] Note %s:\n%s\n\n", i+1, r.UID, preview))
	}
	return sb.String(), nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Helper: QueryMemos tool (Exact Database Search)
// ─────────────────────────────────────────────────────────────────────────────

type queryMemosTool struct {
	store  *store.Store
	userID int32
}

func newQueryMemosTool(store *store.Store, userID int32) tools.Tool {
	return &queryMemosTool{store: store, userID: userID}
}

func (t *queryMemosTool) Name() string { return "query_memos" }
func (t *queryMemosTool) Description() string {
	return `Search the user's notes using exact text matches and specific date ranges. 
Input must be a JSON string with optional keys: 
- "text_search": exact keyword/phrase to search
- "date_start": start date (YYYY-MM-DD or YYYY-MM-DDTHH:MM:SSZ)
- "date_end": end date (YYYY-MM-DD or YYYY-MM-DDTHH:MM:SSZ)
Use this tool over search_memos when the user asks for notes on specific dates (e.g., "what did I post on Jan 26").`
}
func (t *queryMemosTool) Call(ctx context.Context, input string) (string, error) {
	slog.Info("[AGENT TOOL CALL]", "tool", t.Name(), "input", input)
	var payload struct {
		TextSearch string `json:"text_search"`
		DateStart  string `json:"date_start"`
		DateEnd    string `json:"date_end"`
	}
	if err := json.Unmarshal([]byte(input), &payload); err != nil {
		return "Error: failed to parse input JSON.", nil
	}

	find := &store.FindMemo{
		CreatorID:       &t.userID,
		ExcludeComments: true,
	}

	if payload.TextSearch != "" {
		// CEL engine wrapper for standard text matching
		find.Filters = append(find.Filters, fmt.Sprintf("content.contains('%s')", strings.ReplaceAll(payload.TextSearch, "'", "\\'")))
	}
	
	if payload.DateStart != "" {
		parsed, err := time.Parse("2006-01-02", payload.DateStart)
		if err == nil {
			find.Filters = append(find.Filters, fmt.Sprintf("created_ts >= %d", parsed.Unix()))
		}
	}
	if payload.DateEnd != "" {
		parsed, err := time.Parse("2006-01-02", payload.DateEnd)
		if err == nil {
			// Add 24 hours to include the whole end day
			find.Filters = append(find.Filters, fmt.Sprintf("created_ts <= %d", parsed.Add(24*time.Hour).Unix()))
		}
	}

	memos, err := t.store.ListMemos(ctx, find)
	if err != nil {
		return "Error searching database: " + err.Error(), nil
	}

	if len(memos) == 0 {
		return "No notes found matching those criteria.", nil
	}
	
	var sb strings.Builder
	for i, r := range memos {
		if i >= 5 {
			sb.WriteString(fmt.Sprintf("... and %d more notes skipped.", len(memos)-5))
			break
		}
		preview := r.Content
		if len(preview) > 400 {
			preview = preview[:400] + "..."
		}
		sb.WriteString(fmt.Sprintf("[%d] Note %s (Created: %s):\n%s\n\n", 
			i+1, 
			r.UID, 
			time.Unix(r.CreatedTs, 0).Format("2006-01-02 15:04"), 
			preview,
		))
	}
	return sb.String(), nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

func buildSystemPrompt(summary string, now time.Time) string {
	base := fmt.Sprintf(
		`You are an AI assistant for the user's personal knowledge base (Memos app).
Today's local date: %s.

You have access to tools that let you read the user's database. YOU CURRENTLY HAVE ZERO KNOWLEDGE OF THE USER'S NOTES.
CRITICAL INSTRUCTIONS:
1. YOU MUST ALWAYS USE A TOOL to look up notes. NEVER attempt to answer questions about the user's notes from your own memory.
2. For questions about a SPECIFIC DATE or exact keyword, YOU MUST use "query_memos". This is mandatory.
3. For general conceptual questions, use "search_memos".
4. To create, append, tag, or delete notes, use the respective tools.
5. NEVER hallucinate note content. If a tool returns no results, tell the user exactly that.`,
		now.Format("2006-01-02 15:04:05"),
	)
	if summary != "" {
		base += "\n\nSummary of earlier conversation:\n" + summary
	}
	return base
}


// buildToolDef constructs an OpenAI-compatible tool definition map.
func buildToolDef(name, description string, properties map[string]any, required []string) map[string]any {
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        name,
			"description": description,
			"parameters": map[string]any{
				"type":       "object",
				"properties": properties,
				"required":   required,
			},
		},
	}
}


// callLLM makes a simple single-turn chat completion request to OpenRouter.
func (s *APIV1Service) callLLM(ctx context.Context, prompt string) (string, error) {
	reqBody := map[string]any{
		"model":    s.Profile.AIModel,
		"messages": []map[string]any{{"role": "user", "content": prompt}},
	}
	bodyBytes, _ := json.Marshal(reqBody)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://openrouter.ai/api/v1/chat/completions",
		bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Authorization", "Bearer "+s.Profile.OpenRouterAPIKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var apiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return "", err
	}
	if len(apiResp.Choices) == 0 {
		return "", fmt.Errorf("empty response from LLM")
	}
	return apiResp.Choices[0].Message.Content, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// proxySSEFromOpenRouter is a helper for future true streaming from OpenRouter.
// Currently unused — kept for when langchaingo streaming is wired.
func proxySSEFromOpenRouter(dst io.Writer, resp *http.Response) {
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := strings.TrimPrefix(line, "data: ")
		if payload == "[DONE]" {
			break
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			continue
		}
		for _, ch := range chunk.Choices {
			if ch.Delta.Content != "" {
				data, _ := json.Marshal(map[string]string{"type": "token", "content": ch.Delta.Content})
				fmt.Fprintf(dst, "data: %s\n\n", data)
				if f, ok := dst.(http.Flusher); ok {
					f.Flush()
				}
			}
		}
	}
	_ = bytes.NewBuffer(nil) // suppress unused import
}
