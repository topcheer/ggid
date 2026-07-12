package service

import (
	"fmt"
	"sync"
	"time"
)

type DeviceBinding struct {
	DeviceID    string    `json:"device_id"`
	UserID      string    `json:"user_id"`
	DeviceName  string    `json:"device_name"`
	Fingerprint string    `json:"fingerprint"`
	Platform    string    `json:"platform"`
	BoundAt     time.Time `json:"bound_at"`
	LastSeen    time.Time `json:"last_seen"`
	TrustScore  int       `json:"trust_score"`
}

type DeviceBindingService struct {
	mu       sync.RWMutex
	devices  map[string]*DeviceBinding // deviceID -> binding
	byUser   map[string][]string       // userID -> []deviceID
	seq      int
}

func NewDeviceBindingService() *DeviceBindingService {
	return &DeviceBindingService{
		devices: make(map[string]*DeviceBinding),
		byUser:  make(map[string][]string),
	}
}

func (s *DeviceBindingService) BindDevice(userID, deviceName, fingerprint, platform string) (*DeviceBinding, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Check duplicate fingerprint
	for _, d := range s.devices {
		if d.Fingerprint == fingerprint && d.UserID == userID {
			return nil, fmt.Errorf("device already bound to user")
		}
	}
	s.seq++
	d := &DeviceBinding{
		DeviceID:    fmt.Sprintf("dev_%d", s.seq),
		UserID:      userID,
		DeviceName:  deviceName,
		Fingerprint: fingerprint,
		Platform:    platform,
		BoundAt:     time.Now(),
		LastSeen:    time.Now(),
		TrustScore:  50, // initial trust
	}
	s.devices[d.DeviceID] = d
	s.byUser[userID] = append(s.byUser[userID], d.DeviceID)
	return d, nil
}

func (s *DeviceBindingService) UnbindDevice(deviceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.devices[deviceID]
	if !ok {
		return fmt.Errorf("device not found")
	}
	delete(s.devices, deviceID)
	var filtered []string
	for _, id := range s.byUser[d.UserID] {
		if id != deviceID {
			filtered = append(filtered, id)
		}
	}
	s.byUser[d.UserID] = filtered
	return nil
}

func (s *DeviceBindingService) ListBoundDevices(userID string) []*DeviceBinding {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var list []*DeviceBinding
	for _, did := range s.byUser[userID] {
		if d, ok := s.devices[did]; ok {
			list = append(list, d)
		}
	}
	return list
}

func (s *DeviceBindingService) VerifyDeviceBinding(userID, fingerprint string) (*DeviceBinding, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, did := range s.byUser[userID] {
		if d, ok := s.devices[did]; ok {
			if d.Fingerprint == fingerprint {
				// Update last seen and trust score
				d.LastSeen = time.Now()
				if d.TrustScore < 100 {
					d.TrustScore += 5
				}
				return d, true
			}
		}
	}
	return nil, false
}