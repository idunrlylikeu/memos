package vectorstore

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	chromem "github.com/philippgille/chromem-go"
)

// SearchResult is a single semantic-search hit.
type SearchResult struct {
	MemoUID string
	Content string
	Score   float32
}

// Store wraps chromem-go with per-user collections and disk persistence.
type Store struct {
	mu       sync.RWMutex
	db       *chromem.DB
	dataDir  string
	embedFn  chromem.EmbeddingFunc
}

// New creates (or opens) the persistent vector store at dataDir/vectorstore/.
// embedFunc is the embedding function to use — pass chromem.NewEmbeddingFuncOpenAICompat
// pointed at the OpenRouter embeddings endpoint.
func New(dataDir string, embedFunc chromem.EmbeddingFunc) (*Store, error) {
	dir := filepath.Join(dataDir, "vectorstore")
	if err := os.MkdirAll(dir, 0750); err != nil {
		return nil, fmt.Errorf("create vectorstore dir: %w", err)
	}
	db, err := chromem.NewPersistentDB(dir, false)
	if err != nil {
		return nil, fmt.Errorf("open vectorstore: %w", err)
	}
	return &Store{db: db, dataDir: dir, embedFn: embedFunc}, nil
}

// collectionName returns the per-user collection name.
func collectionName(userID int32) string {
	return fmt.Sprintf("user_%d_memos", userID)
}

// getOrCreateCollection returns (or creates) the per-user collection.
func (s *Store) getOrCreateCollection(userID int32) *chromem.Collection {
	name := collectionName(userID)
	col := s.db.GetCollection(name, s.embedFn)
	if col == nil {
		var err error
		col, err = s.db.CreateCollection(name, nil, s.embedFn)
		if err != nil {
			slog.Error("failed to create vector collection", "user", userID, "err", err)
			return nil
		}
	}
	return col
}

// UpsertMemo indexes (or re-indexes) a memo for a user.
// tags is a space-separated string of hashtags like "#golang #project".
func (s *Store) UpsertMemo(ctx context.Context, userID int32, memoUID, content, tags string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	col := s.getOrCreateCollection(userID)
	if col == nil {
		return fmt.Errorf("vectorstore: nil collection for user %d", userID)
	}

	doc := chromem.Document{
		ID:      memoUID,
		Content: content,
		Metadata: map[string]string{
			"tags": tags,
		},
	}
	return col.AddDocument(ctx, doc)
}

// SearchSimilar returns the top-k memos most semantically similar to the query.
func (s *Store) SearchSimilar(ctx context.Context, userID int32, query string, k int) ([]SearchResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	col := s.getOrCreateCollection(userID)
	if col == nil {
		return nil, nil
	}
	
	count := col.Count()
	if count == 0 {
		return nil, nil
	}
	if k > count {
		k = count
	}

	var results []chromem.Result
	var err error
	
	// chromem-go sometimes throws "nResults must be <= number of documents" despite Count checks.
	// Step down k if it fails.
	for attemptK := k; attemptK > 0; attemptK-- {
		results, err = col.Query(ctx, query, attemptK, nil, nil)
		if err == nil {
			break
		}
	}
	if err != nil {
		return nil, err
	}

	out := make([]SearchResult, 0, len(results))
	for _, r := range results {
		out = append(out, SearchResult{
			MemoUID: r.ID,
			Content: r.Content,
			Score:   r.Similarity,
		})
	}
	return out, nil
}
