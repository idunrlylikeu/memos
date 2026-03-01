package store

// AIChatSession represents a single conversation thread.
type AIChatSession struct {
	ID        int32
	UID       string
	CreatorID int32
	Title     string
	Summary   string // compacted/summarized older history
	CreatedTs int64
	UpdatedTs int64
}

// AIChatMessage is a single message within a session.
type AIChatMessage struct {
	ID         int32
	SessionID  int32
	Role       string // "user" | "assistant" | "tool"
	Content    string
	ToolName   string // non-empty when Role == "tool"
	TokenCount int32
	CreatedTs  int64
}

// FindAIChatSession filters for ListAIChatSessions.
type FindAIChatSession struct {
	UID       *string
	CreatorID *int32
}

// UpdateAIChatSession carries fields accepted by UpdateAIChatSession.
type UpdateAIChatSession struct {
	UID     string
	Title   *string
	Summary *string
}

// FindAIChatMessage filters for ListAIChatMessages.
type FindAIChatMessage struct {
	SessionID int32
}

// CreateAIChatTemplate is the payload for CreateAIChatMessage.
type CreateAIChatMessage struct {
	SessionID  int32
	Role       string
	Content    string
	ToolName   string
	TokenCount int32
}
