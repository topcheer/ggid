package service

import (
	"os"
	"testing"

	"github.com/ggid/ggid/pkg/crypto"
)

// TestMain enables fast password hashing for all auth service tests.
// This prevents Argon2id (64MB memory per hash) from causing timeouts
// when hundreds of tests call HashPassword under the race detector.
func TestMain(m *testing.M) {
	crypto.EnableTestFastHash()
	os.Exit(m.Run())
}
