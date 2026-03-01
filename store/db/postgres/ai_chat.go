package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/usememos/memos/store"
)

func (d *DB) EnsureAIChatTables(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS ai_chat_session (
			id         SERIAL PRIMARY KEY,
			uid        TEXT    NOT NULL UNIQUE,
			creator_id INTEGER NOT NULL,
			title      TEXT    NOT NULL DEFAULT 'New Chat',
			summary    TEXT    NOT NULL DEFAULT '',
			created_ts BIGINT  NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
			updated_ts BIGINT  NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())
		)`,
		`CREATE TABLE IF NOT EXISTS ai_chat_message (
			id          SERIAL PRIMARY KEY,
			session_id  INTEGER NOT NULL REFERENCES ai_chat_session(id) ON DELETE CASCADE,
			role        TEXT    NOT NULL,
			content     TEXT    NOT NULL,
			tool_name   TEXT    NOT NULL DEFAULT '',
			token_count INTEGER NOT NULL DEFAULT 0,
			created_ts  BIGINT  NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())
		)`,
		`CREATE INDEX IF NOT EXISTS idx_ai_chat_message_session ON ai_chat_message(session_id)`,
	}
	for _, s := range stmts {
		if _, err := d.db.ExecContext(ctx, s); err != nil {
			return err
		}
	}
	return nil
}

func (d *DB) CreateAIChatSession(ctx context.Context, create *store.AIChatSession) (*store.AIChatSession, error) {
	stmt := `INSERT INTO ai_chat_session (uid, creator_id, title)
	         VALUES ($1, $2, $3)
	         RETURNING id, created_ts, updated_ts`
	if err := d.db.QueryRowContext(ctx, stmt, create.UID, create.CreatorID, create.Title).
		Scan(&create.ID, &create.CreatedTs, &create.UpdatedTs); err != nil {
		return nil, err
	}
	return create, nil
}

func (d *DB) ListAIChatSessions(ctx context.Context, find *store.FindAIChatSession) ([]*store.AIChatSession, error) {
	where, args := []string{"1 = 1"}, []any{}
	if v := find.CreatorID; v != nil {
		where, args = append(where, "creator_id = "+placeholder(len(args)+1)), append(args, *v)
	}
	if v := find.UID; v != nil {
		where, args = append(where, "uid = "+placeholder(len(args)+1)), append(args, *v)
	}
	query := fmt.Sprintf(
		`SELECT id, uid, creator_id, title, summary, created_ts, updated_ts
		 FROM ai_chat_session WHERE %s ORDER BY updated_ts DESC`,
		strings.Join(where, " AND "),
	)
	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*store.AIChatSession
	for rows.Next() {
		s := &store.AIChatSession{}
		if err := rows.Scan(&s.ID, &s.UID, &s.CreatorID, &s.Title, &s.Summary, &s.CreatedTs, &s.UpdatedTs); err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

func (d *DB) GetAIChatSession(ctx context.Context, find *store.FindAIChatSession) (*store.AIChatSession, error) {
	list, err := d.ListAIChatSessions(ctx, find)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, nil
	}
	return list[0], nil
}

func (d *DB) UpdateAIChatSession(ctx context.Context, update *store.UpdateAIChatSession) (*store.AIChatSession, error) {
	set, args := []string{}, []any{}
	if v := update.Title; v != nil {
		set, args = append(set, "title = "+placeholder(len(args)+1)), append(args, *v)
	}
	if v := update.Summary; v != nil {
		set, args = append(set, "summary = "+placeholder(len(args)+1)), append(args, *v)
	}
	if len(set) == 0 {
		return d.GetAIChatSession(ctx, &store.FindAIChatSession{UID: &update.UID})
	}
	set = append(set, "updated_ts = EXTRACT(EPOCH FROM NOW())")
	args = append(args, update.UID)
	stmt := fmt.Sprintf(
		`UPDATE ai_chat_session SET %s WHERE uid = %s
		 RETURNING id, uid, creator_id, title, summary, created_ts, updated_ts`,
		strings.Join(set, ", "), placeholder(len(args)),
	)
	s := &store.AIChatSession{}
	if err := d.db.QueryRowContext(ctx, stmt, args...).
		Scan(&s.ID, &s.UID, &s.CreatorID, &s.Title, &s.Summary, &s.CreatedTs, &s.UpdatedTs); err != nil {
		return nil, err
	}
	return s, nil
}

func (d *DB) DeleteAIChatSession(ctx context.Context, uid string) error {
	_, err := d.db.ExecContext(ctx, `DELETE FROM ai_chat_session WHERE uid = $1`, uid)
	return err
}

func (d *DB) CreateAIChatMessage(ctx context.Context, create *store.CreateAIChatMessage) (*store.AIChatMessage, error) {
	stmt := `INSERT INTO ai_chat_message (session_id, role, content, tool_name, token_count)
	         VALUES ($1, $2, $3, $4, $5)
	         RETURNING id, created_ts`
	m := &store.AIChatMessage{
		SessionID:  create.SessionID,
		Role:       create.Role,
		Content:    create.Content,
		ToolName:   create.ToolName,
		TokenCount: create.TokenCount,
	}
	if err := d.db.QueryRowContext(ctx, stmt,
		create.SessionID, create.Role, create.Content, create.ToolName, create.TokenCount,
	).Scan(&m.ID, &m.CreatedTs); err != nil {
		return nil, err
	}
	return m, nil
}

func (d *DB) ListAIChatMessages(ctx context.Context, find *store.FindAIChatMessage) ([]*store.AIChatMessage, error) {
	query := `SELECT id, session_id, role, content, tool_name, token_count, created_ts
	          FROM ai_chat_message WHERE session_id = $1 ORDER BY id ASC`
	rows, err := d.db.QueryContext(ctx, query, find.SessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*store.AIChatMessage
	for rows.Next() {
		m := &store.AIChatMessage{}
		if err := rows.Scan(&m.ID, &m.SessionID, &m.Role, &m.Content, &m.ToolName, &m.TokenCount, &m.CreatedTs); err != nil {
			return nil, err
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

func (d *DB) DeleteAIChatMessages(ctx context.Context, sessionID int32) error {
	_, err := d.db.ExecContext(ctx, `DELETE FROM ai_chat_message WHERE session_id = $1`, sessionID)
	return err
}
