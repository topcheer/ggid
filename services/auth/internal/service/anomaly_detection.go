package service

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	anomalyLockoutThreshold = 5
	anomalyLockoutDuration  = 15 * time.Minute
	anomalyWindow           = 15 * time.Minute
	anomalyKnownIPThreshold = 500.0 // km
)

// AnomalyResult holds the outcome of anomaly detection checks.
type AnomalyResult struct {
	Locked       bool   `json:"locked"`
	LockReason   string `json:"lock_reason,omitempty"`
	GeoAnomaly   bool   `json:"geo_anomaly"`
	NewDevice    bool   `json:"new_device"`
	RequireNotify bool  `json:"require_notify"`
	NotifyReason string `json:"notify_reason,omitempty"`
	Remaining    int    `json:"remaining_attempts"`
}

// RecordFailedLoginAnomaly increments the failed-login counter for a user using a
// Redis sorted set. After anomalyLockoutThreshold failures within anomalyWindow,
// the user is temporarily locked out.
func (s *AuthService) RecordFailedLoginAnomaly(ctx context.Context, username string) (*AnomalyResult, error) {
	result := &AnomalyResult{}
	now := time.Now()
	key := fmt.Sprintf("ggid:anomaly:fail:%s", username)

	// Remove entries older than the window.
	cutoff := now.Add(-anomalyWindow).UnixNano()
	pipe := s.rateLimiter.rdb.Pipeline()
	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", cutoff))
	pipe.ZAdd(ctx, key, redis.Z{Score: float64(now.UnixNano()), Member: fmt.Sprintf("%d", now.UnixNano())})
	pipe.ZCard(ctx, key)
	pipe.Expire(ctx, key, anomalyWindow+time.Minute)
	cmdrs, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return result, err
	}

	count := cmdrs[2].(*redis.IntCmd).Val()
	remaining := int(anomalyLockoutThreshold) - int(count)
	if remaining < 0 {
		remaining = 0
	}
	result.Remaining = remaining

	if count >= int64(anomalyLockoutThreshold) {
		result.Locked = true
		result.LockReason = fmt.Sprintf("Account locked after %d failed attempts. Try again in %v.", count, anomalyLockoutDuration)

		// Set a lock key so IsLoginLocked can check quickly.
		lockKey := fmt.Sprintf("ggid:anomaly:lock:%s", username)
		_ = s.rateLimiter.rdb.Set(ctx, lockKey, fmt.Sprintf("%d", now.Unix()), anomalyLockoutDuration).Err()

		// Clear the failure counter.
		_ = s.rateLimiter.rdb.Del(ctx, key).Err()
	}

	return result, nil
}

// IsLoginLocked checks if a user is currently locked out due to anomaly detection.
func (s *AuthService) IsLoginLocked(ctx context.Context, username string) bool {
	lockKey := fmt.Sprintf("ggid:anomaly:lock:%s", username)
	_, err := s.rateLimiter.rdb.Get(ctx, lockKey).Result()
	return err == nil
}

// ClearFailedLogins removes the failure counter and lock for a user (called on successful login).
func (s *AuthService) ClearFailedLogins(ctx context.Context, username string) {
	failKey := fmt.Sprintf("ggid:anomaly:fail:%s", username)
	lockKey := fmt.Sprintf("ggid:anomaly:lock:%s", username)
	_ = s.rateLimiter.rdb.Del(ctx, failKey, lockKey).Err()
}

// CheckGeoAnomaly compares the current IP's geo location with known IPs.
// If the distance exceeds the threshold for ALL known IPs, it returns true.
// lat, lon are the current login coordinates. knownIPs maps IP -> "lat,lon".
func CheckGeoAnomaly(lat, lon float64, knownIPs map[string]string) bool {
	if len(knownIPs) == 0 {
		return false // no history yet
	}

	validCount := 0
	for _, coords := range knownIPs {
		var knownLat, knownLon float64
		if _, err := fmt.Sscanf(coords, "%f,%f", &knownLat, &knownLon); err != nil {
			continue
		}
		validCount++
		dist := haversineDistance(lat, lon, knownLat, knownLon)
		if dist <= anomalyKnownIPThreshold {
			return false // within known range
		}
	}
	// If no valid coords were found, we can't determine anomaly.
	if validCount == 0 {
		return false
	}
	return true // all known IPs are far away
}

// CheckNewDevice determines whether the given device fingerprint has been seen before.
func CheckNewDevice(fingerprint string, knownDevices []string) bool {
	for _, d := range knownDevices {
		if d == fingerprint {
			return false
		}
	}
	return true
}

// RecordKnownDevice stores a device fingerprint for a user in Redis.
func (s *AuthService) RecordKnownDevice(ctx context.Context, userID, fingerprint string) error {
	key := fmt.Sprintf("ggid:anomaly:devices:%s", userID)
	return s.rateLimiter.rdb.SAdd(ctx, key, fingerprint).Err()
}

// GetKnownDevices retrieves the set of known device fingerprints for a user.
func (s *AuthService) GetKnownDevices(ctx context.Context, userID string) ([]string, error) {
	key := fmt.Sprintf("ggid:anomaly:devices:%s", userID)
	return s.rateLimiter.rdb.SMembers(ctx, key).Result()
}

// AssessLoginAnomaly runs all anomaly checks and returns the combined result.
func (s *AuthService) AssessLoginAnomaly(ctx context.Context, username, userID, ip, deviceFingerprint string, lat, lon float64) (*AnomalyResult, error) {
	result := &AnomalyResult{}

	// Check lockout.
	if s.IsLoginLocked(ctx, username) {
		result.Locked = true
		result.LockReason = "Account is temporarily locked due to suspicious activity."
		return result, nil
	}

	// Check geo anomaly.
	knownIPsKey := fmt.Sprintf("ggid:anomaly:ips:%s", userID)
	knownIPsStr, err := s.rateLimiter.rdb.HGetAll(ctx, knownIPsKey).Result()
	if err == nil && len(knownIPsStr) > 0 {
		if CheckGeoAnomaly(lat, lon, knownIPsStr) {
			result.GeoAnomaly = true
			result.RequireNotify = true
			result.NotifyReason = "Login from an unusual location."
		}
	}

	// Check device anomaly.
	knownDevices, err := s.GetKnownDevices(ctx, userID)
	if err == nil && len(knownDevices) > 0 {
		if CheckNewDevice(deviceFingerprint, knownDevices) {
			result.NewDevice = true
			if !result.RequireNotify {
				result.RequireNotify = true
				result.NotifyReason = "Login from a new device."
			}
		}
	}

	// Record current IP if we have coordinates.
	if lat != 0 && lon != 0 {
		_ = s.rateLimiter.rdb.HSet(ctx, knownIPsKey, ip, fmt.Sprintf("%f,%f", lat, lon)).Err()
		_ = s.rateLimiter.rdb.Expire(ctx, knownIPsKey, 30*24*time.Hour).Err()
	}

	return result, nil
}

// haversineDistance computes the great-circle distance between two points (in km).
func haversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusKm = 6371.0

	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180

	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Sin(dLon/2)*math.Sin(dLon/2)*math.Cos(lat1Rad)*math.Cos(lat2Rad)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusKm * c
}
