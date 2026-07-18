package server

import (
	"testing"
)

func TestBackupRepo_NilPool(t *testing.T) {
	repo := newBackupRepo(nil)
	backups, err := repo.List(nil, 10)
	if err != nil { t.Fatalf("nil pool should not error: %v", err) }
	if len(backups) != 0 { t.Error("nil pool should return empty") }
}

func TestBackupRepo_TriggerNilPool(t *testing.T) {
	repo := newBackupRepo(nil)
	backup, err := repo.TriggerBackup(nil, "full")
	if err != nil { t.Fatalf("nil pool should not error: %v", err) }
	if backup.Type != "full" { t.Error("type mismatch") }
	if backup.Status != "completed" { t.Error("should be completed") }
	if !backup.Encrypted { t.Error("should be encrypted") }
	if backup.Location == "" { t.Error("should have location") }
}

func TestBackupRecord_Struct(t *testing.T) {
	b := &BackupRecord{Type: "wal", Status: "completed", SizeBytes: 1024}
	if b.Type != "wal" { t.Error("type mismatch") }
	if b.SizeBytes != 1024 { t.Error("size mismatch") }
}

func TestBackupRepo_VerifyNilPool(t *testing.T) {
	repo := newBackupRepo(nil)
	if err := repo.MarkVerified(nil, "test-id"); err != nil {
		t.Errorf("nil pool should not error: %v", err)
	}
}

func TestBackupRepo_CompleteNilPool(t *testing.T) {
	repo := newBackupRepo(nil)
	if err := repo.MarkCompleted(nil, "test-id", 2048, "s3://test"); err != nil {
		t.Errorf("nil pool should not error: %v", err)
	}
}
