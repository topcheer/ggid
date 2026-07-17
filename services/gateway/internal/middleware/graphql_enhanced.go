package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// GraphQL complexity limits.
const (
	MaxQueryDepth     = 10
	MaxQueryComplexity = 1000
)

// fieldCosts assigns complexity costs per field.
var fieldCosts = map[string]int{
	"users":      10,
	"groups":      5,
	"roles":       5,
	"sessions":    3,
	"auditEvents": 10,
	"policies":    5,
	"riskScore":   2,
	"device":      2,
	"members":     3,
	"permissions": 2,
}

// AnalyzeComplexity estimates query depth + complexity cost.
func AnalyzeComplexity(query string) (depth, complexity int, err error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return 0, 0, fmt.Errorf("empty query")
	}

	// Count nested braces for depth.
	currentDepth, maxDepth := 0, 0
	for _, ch := range query {
		if ch == '{' {
			currentDepth++
			if currentDepth > maxDepth {
				maxDepth = currentDepth
			}
		}
		if ch == '}' {
			currentDepth--
		}
	}
	depth = maxDepth

	// Estimate complexity by counting known field names.
	for field, cost := range fieldCosts {
		count := strings.Count(query, field)
		complexity += count * cost
	}

	return depth, complexity, nil
}

// ValidateQuery checks depth + complexity limits.
func ValidateQuery(query string) error {
	depth, complexity, err := AnalyzeComplexity(query)
	if err != nil {
		return err
	}
	if depth > MaxQueryDepth {
		return fmt.Errorf("query depth %d exceeds maximum %d", depth, MaxQueryDepth)
	}
	if complexity > MaxQueryComplexity {
		return fmt.Errorf("query complexity %d exceeds maximum %d", complexity, MaxQueryComplexity)
	}
	return nil
}

// --- Persisted Queries ---

var (
	persistedQueries   sync.Map // hash → query string
	persistedQueriesMu sync.RWMutex
	persistedMode      bool // when true, only persisted queries accepted
)

// RegisterPersistedQuery stores an approved query by its hash.
func RegisterPersistedQuery(query string) string {
	h := sha256.Sum256([]byte(query))
	hash := hex.EncodeToString(h[:16])
	persistedQueries.Store(hash, query)
	return hash
}

// LookupPersistedQuery retrieves a query by hash.
func LookupPersistedQuery(hash string) (string, bool) {
	if q, ok := persistedQueries.Load(hash); ok {
		return q.(string), true
	}
	return "", false
}

// SetPersistedOnlyMode enables/disables persisted-only mode.
func SetPersistedOnlyMode(enabled bool) {
	persistedQueriesMu.Lock()
	persistedMode = enabled
	persistedQueriesMu.Unlock()
}

// IsPersistedOnly returns whether only persisted queries are accepted.
func IsPersistedOnly() bool {
	persistedQueriesMu.RLock()
	defer persistedQueriesMu.RUnlock()
	return persistedMode
}

// --- Query Audit Log (PG-backed) ---

type graphQLQueryLog struct {
	pool *pgxpool.Pool
}

var gqlLog *graphQLQueryLog

// InitGraphQLQueryLog initializes the PG-backed query log.
func InitGraphQLQueryLog(pool *pgxpool.Pool) {
	gqlLog = &graphQLQueryLog{pool: pool}
	if pool != nil {
		pool.Exec(nil, `CREATE TABLE IF NOT EXISTS graphql_query_log (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			operation TEXT NOT NULL,
			query_hash TEXT,
			complexity INT DEFAULT 0,
			depth INT DEFAULT 0,
			duration_ms INT DEFAULT 0,
			error TEXT,
			user_id TEXT,
			created_at TIMESTAMPTZ DEFAULT now()
		)`)
	}
}

// LogGraphQLQuery records a GraphQL operation to the audit log.
func LogGraphQLQuery(operation, queryHash string, complexity, depth, durationMs int, errMsg, userID string) {
	if gqlLog == nil || gqlLog.pool == nil {
		return
	}
	gqlLog.pool.Exec(nil, `INSERT INTO graphql_query_log (operation,query_hash,complexity,depth,duration_ms,error,user_id) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		operation, queryHash, complexity, depth, durationMs, errMsg, userID)
}

// --- Enhanced Handler Wrapper ---

// GraphQLMiddleware wraps the GraphQL handler with complexity validation + persisted query check.
func GraphQLMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			next(w, r)
			return
		}

		var req GraphQLRequest
		body, err := io.ReadAll(r.Body)
		if err != nil {
			writeGraphQLResponse(w, nil, []GraphQLError{{Message: "failed to read body"}})
			return
		}
		r.Body.Close()

		if err := json.Unmarshal(body, &req); err != nil {
			writeGraphQLResponse(w, nil, []GraphQLError{{Message: "invalid JSON"}})
			return
		}

		// Check persisted query mode.
		if IsPersistedOnly() {
			hash := r.URL.Query().Get("hash")
			if hash != "" {
				if q, ok := LookupPersistedQuery(hash); ok {
					req.Query = q
				}
			}
			if req.Query == "" || strings.HasPrefix(req.Query, "__") {
				writeGraphQLResponse(w, nil, []GraphQLError{{Message: "only persisted queries allowed in production mode"}})
				return
			}
		}

		// Validate complexity.
		depth, complexity, qErr := AnalyzeComplexity(req.Query)
		if qErr != nil {
			writeGraphQLResponse(w, nil, []GraphQLError{{Message: qErr.Error()}})
			return
		}
		if depth > MaxQueryDepth {
			writeGraphQLResponse(w, nil, []GraphQLError{{Message: fmt.Sprintf("query depth %d exceeds max %d", depth, MaxQueryDepth)}})
			return
		}
		if complexity > MaxQueryComplexity {
			writeGraphQLResponse(w, nil, []GraphQLError{{Message: fmt.Sprintf("query complexity %d exceeds max %d", complexity, MaxQueryComplexity)}})
			return
		}

		// Reconstruct body for the actual handler.
		r.Body = io.NopCloser(strings.NewReader(string(body)))
		next(w, r)

		// Log the query (best-effort).
		h := sha256.Sum256([]byte(req.Query))
		hash := hex.EncodeToString(h[:16])
		operation := "query"
		if strings.Contains(req.Query, "mutation") {
			operation = "mutation"
		}
		LogGraphQLQuery(operation, hash, complexity, depth, 0, "", "")
	}
}

func writeGraphQLResponse(w http.ResponseWriter, data map[string]any, errors []GraphQLError) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(GraphQLResponse{Data: data, Errors: errors})
}

var _ = time.Now
