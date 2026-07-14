package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// LoginAttempt records a single login attempt.
type LoginAttempt struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	IPAddress    string    `json:"ip_address"`
	UserAgent    string    `json:"user_agent"`
	Timestamp    time.Time `json:"timestamp"`
	Success      bool      `json:"success"`
	FailureReason string   `json:"failure_reason,omitempty"`
}

const (
	loginAttemptMaxRecords = 100             // keep last 100 per user
	loginAttemptTTL        = 30 * 24 * time.Hour // 30 days
)

// RecordLoginAttempt logs a login attempt to Redis (sorted set by timestamp).
// It stores the most recent N attempts per user.
func (s *AuthService) RecordLoginAttempt(ctx context.Context, username, ip, userAgent string, success bool, failureReason string) {
	key := fmt.Sprintf("ggid:login_attempts:%s", username)

	attempt := LoginAttempt{
		ID:            fmt.Sprintf("%d", time.Now().UnixNano()),
		Username:      username,
		IPAddress:     ip,
		UserAgent:     userAgent,
		Timestamp:     time.Now(),
		Success:       success,
		FailureReason: failureReason,
	}

	data, err := json.Marshal(attempt)
	if err != nil {
		return
	}

	score := float64(time.Now().UnixNano())
	pipe := s.rateLimiter.rdb.Pipeline()
	pipe.ZAdd(ctx, key, redis.Z{Score: score, Member: string(data)})
	// Keep only the most recent N records.
	pipe.ZRemRangeByRank(ctx, key, 0, int64(-loginAttemptMaxRecords-1))
	pipe.Expire(ctx, key, loginAttemptTTL)
	_, _ = pipe.Exec(ctx)
}

// GetLoginAttempts retrieves login attempts for a user (most recent first).
func (s *AuthService) GetLoginAttempts(ctx context.Context, username string, limit int) ([]LoginAttempt, error) {
	if limit <= 0 || limit > loginAttemptMaxRecords {
		limit = 50
	}

	key := fmt.Sprintf("ggid:login_attempts:%s", username)
	results, err := s.rateLimiter.rdb.ZRevRange(ctx, key, 0, int64(limit-1)).Result() //nolint:staticcheck // SA1019: ZRevRange deprecated but functional
	if err != nil {
		return nil, err
	}

	attempts := make([]LoginAttempt, 0, len(results))
	for _, data := range results {
		var a LoginAttempt
		if err := json.Unmarshal([]byte(data), &a); err == nil {
			attempts = append(attempts, a)
		}
	}
	return attempts, nil
}
