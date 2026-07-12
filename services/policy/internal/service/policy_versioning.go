package service

import (
	"sync"
	"time"
)

type PolicyVersionStatus string

const (
	PolicyVersionDraft     PolicyVersionStatus = "draft"
	PolicyVersionActive    PolicyVersionStatus = "active"
	PolicyVersionArchived  PolicyVersionStatus = "archived"
	PolicyVersionRolledBack PolicyVersionStatus = "rolled_back"
)

type PolicyVersion struct {
	VersionID     string              `json:"version_id"`
	PolicyID      string              `json:"policy_id"`
	VersionNumber int                 `json:"version_number"`
	Status        PolicyVersionStatus `json:"status"`
	Diff          string              `json:"diff"`
	CreatedBy     string              `json:"created_by"`
	CreatedAt     time.Time           `json:"created_at"`
}

type VersionDiff struct {
	FromVersion int      `json:"from_version"`
	ToVersion   int      `json:"to_version"`
	Added       []string `json:"added"`
	Removed     []string `json:"removed"`
	Modified    []string `json:"modified"`
}

type PolicyVersioningService struct {
	mu       sync.RWMutex
	versions map[string][]PolicyVersion // policy_id -> versions (sorted by version_number)
	seq      int
}

func NewPolicyVersioningService() *PolicyVersioningService {
	return &PolicyVersioningService{versions: make(map[string][]PolicyVersion)}
}

func (s *PolicyVersioningService) CreateVersion(policyID, createdBy, diff string) *PolicyVersion {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	v := PolicyVersion{
		VersionID:     pvID(s.seq),
		PolicyID:      policyID,
		VersionNumber: len(s.versions[policyID]) + 1,
		Status:        PolicyVersionDraft,
		Diff:          diff,
		CreatedBy:     createdBy,
		CreatedAt:     time.Now(),
	}
	s.versions[policyID] = append(s.versions[policyID], v)
	return &v
}

func (s *PolicyVersioningService) GetVersion(policyID, versionID string) *PolicyVersion {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, v := range s.versions[policyID] {
		if v.VersionID == versionID {
			return &v
		}
	}
	return nil
}

func (s *PolicyVersioningService) ListVersions(policyID string) []PolicyVersion {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.versions[policyID]
}

func (s *PolicyVersioningService) RollbackVersion(policyID, versionID string) (*PolicyVersion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	versions := s.versions[policyID]
	var target *PolicyVersion
	for i := range versions {
		if versions[i].VersionID == versionID {
			target = &versions[i]
			break
		}
	}
	if target == nil {
		return nil, nil
	}
	// Archive current active
	for i := range versions {
		if versions[i].Status == PolicyVersionActive {
			versions[i].Status = PolicyVersionArchived
		}
	}
	target.Status = PolicyVersionActive
	s.seq++
	rollback := PolicyVersion{
		VersionID:     pvID(s.seq),
		PolicyID:      policyID,
		VersionNumber: len(versions) + 1,
		Status:        PolicyVersionActive,
		Diff:          "rollback to v" + versionID,
		CreatedBy:     "system",
		CreatedAt:     time.Now(),
	}
	s.versions[policyID] = append(versions, rollback)
	return &rollback, nil
}

func (s *PolicyVersioningService) CompareVersions(policyID string, v1, v2 int) *VersionDiff {
	s.mu.RLock()
	defer s.mu.RUnlock()
	versions := s.versions[policyID]
	var v1Diff, v2Diff string
	for _, v := range versions {
		if v.VersionNumber == v1 {
			v1Diff = v.Diff
		}
		if v.VersionNumber == v2 {
			v2Diff = v.Diff
		}
	}
	return &VersionDiff{
		FromVersion: v1,
		ToVersion:   v2,
		Added:       []string{},
		Removed:     []string{},
		Modified:    []string{v1Diff, v2Diff},
	}
}

func pvID(n int) string {
	const hex = "0123456789abcdef"
	if n == 0 {
		return "pv_0"
	}
	var buf []byte
	for n > 0 {
		buf = append([]byte{hex[n%16]}, buf...)
		n /= 16
	}
	return "pv_" + string(buf)
}