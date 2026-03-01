package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/usememos/memos/store"
)

func (d *DB) EnsureAIChatTables(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS ai_chat_session (
			id         INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
			uid        VARCHAR(256) NOT NULL UNIQUE,
			creator_id INT NOT NULL,
			title      TEXT NOT NULL,
			summary    TEXT NOT NULL,
			created_ts TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_ts TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS ai_chat_message (
			id          INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
			session_id  INT NOT NULL,
			role        VARCHAR(256) NOT NULL,
			content     TEXT NOT NULL,
			tool_name   VARCHAR(256) NOT NULL DEFAULT '',
			token_count INT NOT NULL DEFAULT 0,
			created_ts  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			CONSTRAINT fk_ai_chat_message_session FOREIGN KEY (session_id) REFERENCES ai_chat_session(id) ON DELETE CASCADE
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
	stmt := "INSERT INTO `ai_chat_session` (`uid`, `creator_id`, `title`) VALUES (?, ?, ?)"
	result, err := d.db.ExecContext(ctx, stmt, create.UID, create.CreatorID, create.Title)
	if err != nil {
		return nil, err
	}
	rawID, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	_ = rawID
	// Fetch it back to populate timestamps
	return d.GetAIChatSession(ctx, &store.FindAIChatSession{CreatorID: &create.CreatorID, UID: &create.UID})
}

func (d *DB) ListAIChatSessions(ctx context.Context, find *store.FindAIChatSession) ([]*store.AIChatSession, error) {
	where, args := []string{"1 = 1"}, []any{}
	if v := find.CreatorID; v != nil {
		where, args = append(where, "`creator_id` = ?"), append(args, *v)
	}
	if v := find.UID; v != nil {
		where, args = append(where, "`uid` = ?"), append(args, *v)
	}
	query := fmt.Sprintf(
		`SELECT id, uid, creator_id, title, summary, UNIX_TIMESTAMP(created_ts), UNIX_TIMESTAMP(updated_ts)
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
		set, args = append(set, "`title` = ?"), append(args, *v)
	}
	if v := update.Summary; v != nil {
		set, args = append(set, "`summary` = ?"), append(args, *v)
	}
	if len(set) == 0 {
		return d.GetAIChatSession(ctx, &store.FindAIChatSession{UID: &update.UID})
	}
	set = append(set, "`updated_ts` = CURRENT_TIMESTAMP")
	args = append(args, update.UID)
	stmt := fmt.Sprintf("UPDATE `ai_chat_session` SET %s WHERE `uid` = ?", strings.Join(set, ", "))
	
	if _, err := d.db.ExecContext(ctx, stmt, args...); err != nil {
		return nil, err
	}
	return d.GetAIChatSession(ctx, &store.FindAIChatSession{UID: &update.UID})
}

func (d *DB) DeleteAIChatSession(ctx context.Context, uid string) error {
	_, err := d.db.ExecContext(ctx, "DELETE FROM `ai_chat_session` WHERE `uid` = ?", uid)
	return err
}

func (d *DB) CreateAIChatMessage(ctx context.Context, create *store.CreateAIChatMessage) (*store.AIChatMessage, error) {
	stmt := "INSERT INTO `ai_chat_message` (`session_id`, `role`, `content`, `tool_name`, `token_count`) VALUES (?, ?, ?, ?, ?)"
	result, err := d.db.ExecContext(ctx, stmt, create.SessionID, create.Role, create.Content, create.ToolName, create.TokenCount)
	if err != nil {
		return nil, err
	}
	rawID, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	
	m := &store.AIChatMessage{
		ID:         int32(rawID),
		SessionID:  create.SessionID,
		Role:       create.Role,
		Content:    create.Content,
		ToolName:   create.ToolName,
		TokenCount: create.TokenCount,
	}
	// Fetch created_ts
	_ = d.db.QueryRowContext(ctx, "SELECT UNIX_TIMESTAMP(created_ts) FROM ai_chat_message WHERE id = ?", m.ID).Scan(&m.CreatedTs)
	
	return m, nil
}

func (d *DB) ListAIChatMessages(ctx context.Context, find *store.FindAIChatMessage) ([]*store.AIChatMessage, error) {
	query := `SELECT id, session_id, role, content, tool_name, token_count, UNIX_TIMESTAMP(created_ts)
	          FROM ai_chat_message WHERE session_id = ? ORDER BY id ASC`
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
	_, err := d.db.ExecContext(ctx, "DELETE FROM `ai_chat_message` WHERE `session_id` = ?", sessionID)
	return err
}
