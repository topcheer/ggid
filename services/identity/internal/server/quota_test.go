package server

import (
	"testing"
)

func TestQuotaForPlan(t *testing.T) {
	free := QuotaForPlan("free")
	if free.MaxUsers != 100 { t.Error("free should have 100 users") }
	pro := QuotaForPlan("pro")
	if pro.MaxUsers != 1000 { t.Error("pro should have 1000 users") }
	ent := QuotaForPlan("enterprise")
	if ent.MaxUsers != 999999 { t.Error("enterprise should be unlimited") }
	unknown := QuotaForPlan("unknown")
	if unknown.Plan != "free" { t.Error("unknown plan should default to free") }
}

func TestQuotaRepo_NilPool(t *testing.T) {
	repo := newQuotaRepo(nil)
	quota, err := repo.GetQuota(nil, "tenant-1")
	if err != nil { t.Fatalf("nil pool should not error: %v", err) }
	if quota.MaxUsers != 100 { t.Error("nil pool should return default quota") }
	usage, err := repo.GetUsage(nil, "tenant-1")
	if err != nil { t.Fatalf("nil pool should not error: %v", err) }
	if usage.UserCount != 0 { t.Error("nil pool should return zero usage") }
}

func TestQuotaRepo_CheckQuota_NilPool(t *testing.T) {
	repo := newQuotaRepo(nil)
	allowed, remaining, err := repo.CheckQuota(nil, "tenant-1", "user_count")
	if err != nil { t.Fatalf("nil pool should not error: %v", err) }
	if !allowed { t.Error("should be allowed with default quota + zero usage") }
	if remaining != 100 { t.Errorf("expected 100 remaining, got %d", remaining) }
}

func TestDefaultQuota(t *testing.T) {
	q := defaultQuota("test")
	if q.Plan != "free" || q.MaxUsers != 100 {
		t.Error("default quota mismatch")
	}
}
