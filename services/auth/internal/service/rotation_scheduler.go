package service

import (
	"sync"
	"time"
)

type RotationPolicy struct {
	IntervalDays      int  `json:"interval_days"`
	AutoRotate        bool `json:"auto_rotate"`
	NotifyBeforeDays  int  `json:"notify_before_days"`
}

type RotationSchedule struct {
	CredentialID  string         `json:"credential_id"`
	Policy        RotationPolicy `json:"policy"`
	LastRotated   time.Time      `json:"last_rotated"`
	NextDue       time.Time      `json:"next_due"`
}

type DueRotation struct {
	CredentialID  string `json:"credential_id"`
	DaysOverdue   int    `json:"days_overdue"`
}

type RotationResult struct {
	CredentialID string    `json:"credential_id"`
	RotatedAt    time.Time `json:"rotated_at"`
	Success      bool      `json:"success"`
}

type RotationScheduler struct {
	mu        sync.RWMutex
	schedules map[string]*RotationSchedule
}

func NewRotationScheduler() *RotationScheduler {
	return &RotationScheduler{schedules: make(map[string]*RotationSchedule)}
}

func (rs *RotationScheduler) ScheduleRotation(credentialID string, policy RotationPolicy) *RotationSchedule {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	now := time.Now()
	sched := &RotationSchedule{
		CredentialID: credentialID,
		Policy:       policy,
		LastRotated:  now,
		NextDue:      now.AddDate(0, 0, policy.IntervalDays),
	}
	rs.schedules[credentialID] = sched
	return sched
}

func (rs *RotationScheduler) CheckDueRotations() []DueRotation {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	var due []DueRotation
	now := time.Now()
	for _, sched := range rs.schedules {
		if now.After(sched.NextDue) {
			days := int(now.Sub(sched.NextDue).Hours() / 24)
			due = append(due, DueRotation{
				CredentialID: sched.CredentialID,
				DaysOverdue:  days,
			})
		}
	}
	return due
}

func (rs *RotationScheduler) ExecuteRotation(credentialID string) *RotationResult {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	sched, ok := rs.schedules[credentialID]
	if !ok {
		return &RotationResult{CredentialID: credentialID, RotatedAt: time.Now(), Success: false}
	}
	now := time.Now()
	sched.LastRotated = now
	sched.NextDue = now.AddDate(0, 0, sched.Policy.IntervalDays)
	return &RotationResult{CredentialID: credentialID, RotatedAt: now, Success: true}
}