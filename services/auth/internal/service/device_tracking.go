package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// DeviceInfo holds device fingerprint information for a session.
type DeviceInfo struct {
	SessionID   string `json:"session_id"`
	Fingerprint string `json:"fingerprint"`
	UserAgent   string `json:"user_agent"`
	IPAddress   string `json:"ip_address"`
	LastSeen    string `json:"last_seen"`
}

// TrackDevice records a device fingerprint for a user session in Redis.
// The fingerprint is a SHA-256 hash of the User-Agent + IP for privacy.
func (s *SessionService) TrackDevice(ctx context.Context, rdb *redis.Client, tenantID, userID, sessionID uuid.UUID, userAgent, ip string) error {
	fp := deviceFingerprint(userAgent, ip)
	key := fmt.Sprintf("ggid:devices:%s:%s", tenantID, userID)

	device := DeviceInfo{
		SessionID:   sessionID.String(),
		Fingerprint: fp,
		UserAgent:   userAgent,
		IPAddress:   ip,
		LastSeen:    time.Now().Format(time.RFC3339),
	}

	// Store in a Redis hash keyed by session ID.
	return rdb.HSet(ctx, key, sessionID.String(), device.Fingerprint+":"+device.IPAddress+":"+device.LastSeen).Err()
}

// ListDevices returns all tracked devices for a user.
func (s *SessionService) ListDevices(ctx context.Context, rdb *redis.Client, tenantID, userID uuid.UUID) ([]DeviceInfo, error) {
	key := fmt.Sprintf("ggid:devices:%s:%s", tenantID, userID)
	sessions, err := rdb.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var devices []DeviceInfo
	for sessionID, data := range sessions {
		parts := splitDeviceData(data)
		d := DeviceInfo{
			SessionID:   sessionID,
			Fingerprint: parts[0],
			IPAddress:   parts[1],
			LastSeen:    parts[2],
		}
		if len(parts) > 3 {
			d.UserAgent = parts[3]
		}
		devices = append(devices, d)
	}
	return devices, nil
}

// RemoveDevice removes a device from tracking (on logout).
func (s *SessionService) RemoveDevice(ctx context.Context, rdb *redis.Client, tenantID, userID, sessionID uuid.UUID) error {
	key := fmt.Sprintf("ggid:devices:%s:%s", tenantID, userID)
	return rdb.HDel(ctx, key, sessionID.String()).Err()
}

func deviceFingerprint(userAgent, ip string) string {
	h := sha256.Sum256([]byte(userAgent + ip))
	return hex.EncodeToString(h[:16])
}

func splitDeviceData(data string) []string {
	var parts []string
	current := ""
	for _, c := range data {
		if c == ':' {
			parts = append(parts, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	parts = append(parts, current)
	return parts
}
