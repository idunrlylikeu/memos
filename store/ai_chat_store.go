package store

import "context"

// CreateAIChatSession creates a new AI chat session.
func (s *Store) CreateAIChatSession(ctx context.Context, create *AIChatSession) (*AIChatSession, error) {
	return s.driver.CreateAIChatSession(ctx, create)
}

// ListAIChatSessions lists AI chat sessions matching the given filter.
func (s *Store) ListAIChatSessions(ctx context.Context, find *FindAIChatSession) ([]*AIChatSession, error) {
	return s.driver.ListAIChatSessions(ctx, find)
}

// GetAIChatSession returns the first session matching the given filter.
func (s *Store) GetAIChatSession(ctx context.Context, find *FindAIChatSession) (*AIChatSession, error) {
	return s.driver.GetAIChatSession(ctx, find)
}

// UpdateAIChatSession updates a session's mutable fields.
func (s *Store) UpdateAIChatSession(ctx context.Context, update *UpdateAIChatSession) (*AIChatSession, error) {
	return s.driver.UpdateAIChatSession(ctx, update)
}

// DeleteAIChatSession deletes a session and all its messages (cascade).
func (s *Store) DeleteAIChatSession(ctx context.Context, uid string) error {
	return s.driver.DeleteAIChatSession(ctx, uid)
}

// CreateAIChatMessage persists a new message to a session.
func (s *Store) CreateAIChatMessage(ctx context.Context, create *CreateAIChatMessage) (*AIChatMessage, error) {
	return s.driver.CreateAIChatMessage(ctx, create)
}

// ListAIChatMessages returns all messages for a given session, ordered oldest first.
func (s *Store) ListAIChatMessages(ctx context.Context, find *FindAIChatMessage) ([]*AIChatMessage, error) {
	return s.driver.ListAIChatMessages(ctx, find)
}

// DeleteAIChatMessages deletes all messages for the given session (used during compaction).
func (s *Store) DeleteAIChatMessages(ctx context.Context, sessionID int32) error {
	return s.driver.DeleteAIChatMessages(ctx, sessionID)
}
