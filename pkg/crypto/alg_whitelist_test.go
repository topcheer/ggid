package crypto

import "testing"

func TestIsSupportedAlg(t *testing.T) {
	supported := []string{
		"RS256", "RS384", "RS512",
		"PS256", "PS384", "PS512",
		"ES256", "ES384", "ES512",
		"EdDSA", "SM2SM3",
	}
	for _, alg := range supported {
		if !IsSupportedAlg(alg) {
			t.Errorf("IsSupportedAlg(%q) = false, want true", alg)
		}
	}

	unsupported := []string{"none", "HS256", "HS384", "HS512", "", "sm2sm3", "RS1024"}
	for _, alg := range unsupported {
		if IsSupportedAlg(alg) {
			t.Errorf("IsSupportedAlg(%q) = true, want false", alg)
		}
	}
}

func TestSupportedAlgs_Immutable(t *testing.T) {
	a := SupportedAlgs()
	if len(a) != 11 {
		t.Fatalf("SupportedAlgs len = %d, want 11", len(a))
	}
	a[0] = "none"
	if IsSupportedAlg("none") {
		t.Fatal("SupportedAlgs must return a copy; mutation leaked into whitelist")
	}
}

func TestSupportedAlgs_ContainsSM2(t *testing.T) {
	found := false
	for _, alg := range SupportedAlgs() {
		if alg == SM2SM3Alg {
			found = true
		}
	}
	if !found {
		t.Fatal("SupportedAlgs must include SM2SM3")
	}
}
