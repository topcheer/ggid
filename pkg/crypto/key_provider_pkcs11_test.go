//go:build pkcs11

package crypto

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/miekg/pkcs11"
)

// findSoftHSM2Lib returns the path to the SoftHSM2 PKCS#11 library if available.
func findSoftHSM2Lib(t *testing.T) string {
	if v := os.Getenv("GGID_PKCS11_LIB"); v != "" {
		if _, err := os.Stat(v); err == nil {
			return v
		}
	}
	candidates := []string{
		"/usr/lib/softhsm/libsofthsm2.so",
		"/usr/lib/x86_64-linux-gnu/softhsm/libsofthsm2.so",
		"/usr/local/lib/softhsm/libsofthsm2.so",
		"/opt/homebrew/lib/softhsm/libsofthsm2.so",
		"/opt/local/lib/softhsm/libsofthsm2.so",
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}

// setupSoftHSM2Token initializes a temporary SoftHSM2 token and returns its config directory.
func setupSoftHSM2Token(t *testing.T) (libPath, configPath, pin string, slotID uint) {
	libPath = findSoftHSM2Lib(t)
	if libPath == "" {
		t.Skip("SoftHSM2 library not found; set GGID_PKCS11_LIB to run PKCS#11 tests")
	}

	dir := t.TempDir()
	configPath = filepath.Join(dir, "softhsm2.conf")
	if err := os.WriteFile(configPath, []byte("directories.tokendir = "+dir+"/tokens\nobjectstore.backend = file\nlog.level = INFO\n"), 0600); err != nil {
		t.Fatalf("write softhsm2 config: %v", err)
	}
	t.Setenv("SOFTHSM2_CONF", configPath)
	if err := os.MkdirAll(filepath.Join(dir, "tokens"), 0700); err != nil {
		t.Fatalf("create tokens dir: %v", err)
	}

	pin = "1234"
	soPIN := "1234"
	label := "ggid-test"

	cmd := exec.Command("softhsm2-util", "--init-token", "--slot", "0", "--label", label, "--so-pin", soPIN, "--pin", pin)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Skipf("softhsm2-util not available or failed: %v\n%s", err, string(out))
	}

	// Load module, find the initialized slot, and generate an RSA key pair.
	ctx := pkcs11.New(libPath)
	if ctx == nil {
		t.Fatal("failed to load PKCS#11 library")
	}
	if err := ctx.Initialize(); err != nil {
		t.Fatalf("initialize: %v", err)
	}
	defer ctx.Finalize()

	slots, err := ctx.GetSlotList(true)
	if err != nil {
		t.Fatalf("get slot list: %v", err)
	}
	var slot uint
	for _, s := range slots {
		info, err := ctx.GetSlotInfo(s)
		if err != nil {
			continue
		}
		if info.SlotDescription == label {
			slot = s
			break
		}
	}
	if slot == 0 {
		t.Fatal("no SoftHSM2 slot found after initialization")
	}

	session, err := ctx.OpenSession(slot, pkcs11.CKF_SERIAL_SESSION|pkcs11.CKF_RW_SESSION)
	if err != nil {
		t.Fatalf("open session: %v", err)
	}
	defer ctx.CloseSession(session)
	if err := ctx.Login(session, pkcs11.CKU_USER, pin); err != nil {
		t.Fatalf("login: %v", err)
	}
	defer ctx.Logout(session)

	labelAttr := "ggid-test-rsa"
	pubTemplate := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_CLASS, pkcs11.CKO_PUBLIC_KEY),
		pkcs11.NewAttribute(pkcs11.CKA_KEY_TYPE, pkcs11.CKK_RSA),
		pkcs11.NewAttribute(pkcs11.CKA_TOKEN, true),
		pkcs11.NewAttribute(pkcs11.CKA_VERIFY, true),
		pkcs11.NewAttribute(pkcs11.CKA_MODULUS_BITS, 2048),
		pkcs11.NewAttribute(pkcs11.CKA_LABEL, labelAttr),
		pkcs11.NewAttribute(pkcs11.CKA_ID, []byte("rsa-key")),
	}
	privTemplate := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_CLASS, pkcs11.CKO_PRIVATE_KEY),
		pkcs11.NewAttribute(pkcs11.CKA_KEY_TYPE, pkcs11.CKK_RSA),
		pkcs11.NewAttribute(pkcs11.CKA_TOKEN, true),
		pkcs11.NewAttribute(pkcs11.CKA_SIGN, true),
		pkcs11.NewAttribute(pkcs11.CKA_LABEL, labelAttr),
		pkcs11.NewAttribute(pkcs11.CKA_ID, []byte("rsa-key")),
		pkcs11.NewAttribute(pkcs11.CKA_SENSITIVE, true),
		pkcs11.NewAttribute(pkcs11.CKA_EXTRACTABLE, false),
	}
	mech := []*pkcs11.Mechanism{pkcs11.NewMechanism(pkcs11.CKM_RSA_PKCS_KEY_PAIR_GEN, nil)}
	if _, _, err := ctx.GenerateKeyPair(session, mech, pubTemplate, privTemplate); err != nil {
		t.Fatalf("generate key pair: %v", err)
	}

	return libPath, configPath, pin, slot
}

func TestPKCS11KeyProvider_SignAndVerify(t *testing.T) {
	libPath, _, pin, slot := setupSoftHSM2Token(t)

	os.Setenv("GGID_PKCS11_LIB", libPath)
	os.Setenv("GGID_PKCS11_PIN", pin)
	os.Setenv("GGID_PKCS11_SLOT", fmt.Sprintf("%d", slot))
	os.Setenv("GGID_PKCS11_KEY_LABEL", "ggid-test-rsa")

	provider, err := newPKCS11KeyProvider(context.Background(), PKCS11KeyProviderConfig{})
	if err != nil {
		t.Fatalf("newPKCS11KeyProvider: %v", err)
	}
	defer provider.Close()

	meta := provider.Metadata()
	if meta.KeyID != "ggid-test-rsa" {
		t.Errorf("expected key ID ggid-test-rsa, got %s", meta.KeyID)
	}
	if meta.Algorithm != RS256 {
		t.Errorf("expected algorithm RS256, got %s", meta.Algorithm)
	}

	pub, ok := provider.Public().(*rsa.PublicKey)
	if !ok {
		t.Fatalf("expected *rsa.PublicKey, got %T", provider.Public())
	}

	digest := sha256.Sum256([]byte("hello pkcs11"))
	signer := provider.Signer()
	sig, err := signer.Sign(rand.Reader, digest[:], crypto.SHA256)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	if err := rsa.VerifyPKCS1v15(pub, crypto.SHA256, digest[:], sig); err != nil {
		t.Fatalf("signature verification failed: %v", err)
	}
}

func TestPKCS11KeyProvider_MissingLibrary(t *testing.T) {
	os.Setenv("GGID_PKCS11_LIB", "/nonexistent/libsofthsm2.so")
	os.Unsetenv("GGID_PKCS11_PIN")
	os.Unsetenv("GGID_PKCS11_SLOT")
	os.Setenv("GGID_PKCS11_KEY_LABEL", "test")

	_, err := newPKCS11KeyProvider(context.Background(), PKCS11KeyProviderConfig{})
	if err == nil {
		t.Fatal("expected error for missing library")
	}
}
