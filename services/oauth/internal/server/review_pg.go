package server

import (
	"encoding/json"
	"context"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
)

type pgReviewStore struct{ pool *pgxpool.Pool }

type reviewAdapter struct {
	pg   *pgReviewStore
	mu   *sync.RWMutex
	store map[string]*AgentReview
}

var reviewAdapterVar *reviewAdapter

func newReviewAdapter(pool *pgxpool.Pool) *reviewAdapter {
	a := &reviewAdapter{mu: &reviewMu, store: reviewStore}
	if pool != nil {
		a.pg = &pgReviewStore{pool: pool}
		ctx := context.Background()
		a.pg.EnsureSchema(ctx)
	}
	return a
}

func (s *pgReviewStore) EnsureSchema(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS agent_reviews (id TEXT PRIMARY KEY, agent_id TEXT NOT NULL, reviewer TEXT NOT NULL, scopes_reviewed JSONB DEFAULT '[]', decision TEXT DEFAULT '', comment TEXT DEFAULT '', timestamp TIMESTAMPTZ DEFAULT NOW())`)
	return err
}

func (s *pgReviewStore) Put(ctx context.Context, rv *AgentReview) error {
	scopesJSON, _ := json.Marshal(rv.ScopesReviewed)
	_, err := s.pool.Exec(ctx, `INSERT INTO agent_reviews (id, agent_id, reviewer, scopes_reviewed, decision, comment, timestamp) VALUES ($1,$2,$3,$4,$5,$6,$7) ON CONFLICT (id) DO UPDATE SET decision=$5, comment=$6, timestamp=$7`, rv.ID, rv.AgentID, rv.Reviewer, scopesJSON, rv.Decision, rv.Comment, rv.Timestamp)
	return err
}

func (s *pgReviewStore) Get(ctx context.Context, id string) (*AgentReview, bool) {
	var rv AgentReview; var scopesBytes []byte
	err := s.pool.QueryRow(ctx, `SELECT id, agent_id, reviewer, scopes_reviewed, decision, comment, timestamp FROM agent_reviews WHERE id = $1`, id).Scan(&rv.ID, &rv.AgentID, &rv.Reviewer, &scopesBytes, &rv.Decision, &rv.Comment, &rv.Timestamp)
	if err != nil { return nil, false }
	if len(scopesBytes) > 0 { json.Unmarshal(scopesBytes, &rv.ScopesReviewed) }
	return &rv, true
}

func (s *pgReviewStore) ListByAgent(ctx context.Context, agentID string) ([]*AgentReview, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, agent_id, reviewer, scopes_reviewed, decision, comment, timestamp FROM agent_reviews WHERE agent_id = $1 ORDER BY timestamp DESC`, agentID)
	if err != nil { return nil, err }
	defer rows.Close()
	var result []*AgentReview
	for rows.Next() {
		var rv AgentReview; var scopesBytes []byte
		if err := rows.Scan(&rv.ID, &rv.AgentID, &rv.Reviewer, &scopesBytes, &rv.Decision, &rv.Comment, &rv.Timestamp); err != nil { return nil, err }
		if len(scopesBytes) > 0 { json.Unmarshal(scopesBytes, &rv.ScopesReviewed) }
		result = append(result, &rv)
	}
	return result, nil
}

func (a *reviewAdapter) Put(rv *AgentReview) {
	if a.pg != nil { a.pg.Put(context.Background(), rv); return }
	a.mu.Lock(); a.store[rv.ID] = rv; a.mu.Unlock()
}

func (a *reviewAdapter) Get(id string) (*AgentReview, bool) {
	if a.pg != nil { rv, ok := a.pg.Get(context.Background(), id); if ok { return rv, true } }
	a.mu.RLock(); rv, ok := a.store[id]; a.mu.RUnlock()
	return rv, ok
}

func (a *reviewAdapter) ListByAgent(agentID string) []*AgentReview {
	if a.pg != nil { list, _ := a.pg.ListByAgent(context.Background(), agentID); if list != nil { return list } }
	a.mu.RLock(); defer a.mu.RUnlock()
	var result []*AgentReview
	for _, rv := range a.store { if rv.AgentID == agentID { result = append(result, rv) } }
	return result
}
