package crypto

import (
	"encoding/hex"
	"os"
	"testing"
)

func TestEncryptDecryptRoundtrip(t *testing.T) {
	key := make([]byte, 32)
	os.Setenv("GGID_ENCRYPTION_KEY", hex.EncodeToString(key))
	defer os.Unsetenv("GGID_ENCRYPTION_KEY")
	keyCache = nil // reset

	pt := "JBSWY3DPEHPK3PXP"
	ct, err := EncryptTOTPSecret(pt)
	if err != nil { t.Fatal(err) }
	if ct == pt { t.Fatal("should differ") }
	dt, err := DecryptTOTPSecret(ct)
	if err != nil { t.Fatal(err) }
	if dt != pt { t.Fatalf("got %q want %q", dt, pt) }
}

func TestPlaintextBackwardCompat(t *testing.T) {
	os.Setenv("GGID_ENCRYPTION_KEY", hex.EncodeToString(make([]byte, 32)))
	defer os.Unsetenv("GGID_ENCRYPTION_KEY")
	keyCache = nil
	pt := "JBSWY3DPEHPK3PXP"
	dt, _ := DecryptTOTPSecret(pt)
	if dt != pt { t.Fatalf("plaintext fallback: got %q", dt) }
}

func TestNoKeyReturnsPlaintext(t *testing.T) {
	os.Unsetenv("GGID_ENCRYPTION_KEY")
	keyCache = nil
	ct, err := EncryptTOTPSecret("secret")
	if err != nil { t.Fatal(err) }
	if ct != "secret" { t.Fatal("should return plaintext") }
}

func TestEmptyString(t *testing.T) {
	dt, err := DecryptTOTPSecret("")
	if err != nil || dt != "" { t.Fatalf("empty: got %q err %v", dt, err) }
}
