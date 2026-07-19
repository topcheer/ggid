package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Test 1: Extremely weak password (dictionary word) scores 0.
func TestPasswordStrength_DictionaryWord(t *testing.T) {
	result := EstimateStrength("password")
	if result.Score > 1 {
		t.Errorf("expected score 0-1 for 'password', got %d", result.Score)
	}
	if !sliceContains(result.Patterns, "dictionary") {
		t.Error("expected dictionary pattern detection")
	}
}

// Test 2: Short all-digit password scores 0-1.
func TestPasswordStrength_AllDigits(t *testing.T) {
	result := EstimateStrength("123456")
	if result.Score > 1 {
		t.Errorf("expected score 0-1 for '123456', got %d", result.Score)
	}
}

// Test 3: Strong random password scores 3-4.
func TestPasswordStrength_StrongPassword(t *testing.T) {
	result := EstimateStrength("K7$mQ9!xR2vLp#Wn")
	if result.Score < 3 {
		t.Errorf("expected score 3-4 for strong password, got %d", result.Score)
	}
}

// Test 4: Keyboard sequence detection.
func TestPasswordStrength_KeyboardSequence(t *testing.T) {
	result := EstimateStrength("qwerty123")
	if !sliceContains(result.Patterns, "keyboard_sequence") {
		t.Error("expected keyboard_sequence pattern")
	}
}

// Test 5: Repeated characters detection.
func TestPasswordStrength_Repeats(t *testing.T) {
	result := EstimateStrength("aaabbbccc")
	if !sliceContains(result.Patterns, "repeats") {
		t.Error("expected repeats pattern")
	}
}

// Test 6: L33t substitution detection.
func TestPasswordStrength_L33t(t *testing.T) {
	result := EstimateStrength("p@ssw0rd")
	if !sliceContains(result.Patterns, "l33t") {
		t.Error("expected l33t pattern")
	}
}

// Test 7: Score range is always 0-4.
func TestPasswordStrength_ScoreRange(t *testing.T) {
	tests := []string{"", "a", "password", "123456", "MyStr0ng#Pass2024!", "K7$mQ9!xR2vLp#Wn4Bc"}
	for _, pw := range tests {
		result := EstimateStrength(pw)
		if result.Score < 0 || result.Score > 4 {
			t.Errorf("score %d out of range for password len %d", result.Score, len(pw))
		}
	}
}

// Test 8: POST /password/strength returns 200 with score.
func TestPasswordStrength_Endpoint(t *testing.T) {
	h := &Handler{}

	body := `{"password":"TestStr0ng#KxRvLpWnBc!"}`
	req := httptest.NewRequest("POST", "/api/v1/auth/password/strength", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	h.handlePasswordStrength(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var result StrengthResult
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if result.Score < 2 {
		t.Errorf("expected score >=2 for strong password, got %d", result.Score)
	}
}

// Test 9: POST /password/strength with breached password → score 0.
func TestPasswordStrength_BreachedPassword(t *testing.T) {
	h := &Handler{}

	body := `{"password":"password"}`
	req := httptest.NewRequest("POST", "/api/v1/auth/password/strength", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	h.handlePasswordStrength(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var result StrengthResult
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if result.Score != 0 {
		t.Errorf("expected score 0 for breached password, got %d", result.Score)
	}
}

// Test 10: POST /password/strength with empty password → 400.
func TestPasswordStrength_EmptyPassword(t *testing.T) {
	h := &Handler{}

	body := `{"password":""}`
	req := httptest.NewRequest("POST", "/api/v1/auth/password/strength", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	h.handlePasswordStrength(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// Test 11: Wrong method → 405.
func TestPasswordStrength_WrongMethod(t *testing.T) {
	h := &Handler{}

	req := httptest.NewRequest("DELETE", "/api/v1/auth/password/strength", nil)
	w := httptest.NewRecorder()

	h.handlePasswordStrength(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

// Test 12: Strength gate rejects weak passwords.
func TestPasswordStrength_GateRejectsWeak(t *testing.T) {
	ok, msg := checkPasswordStrengthGate("123")
	if ok {
		t.Error("expected gate to reject '123'")
	}
	if msg == "" {
		t.Error("expected non-empty rejection message")
	}
}

// Test 13: Strength gate accepts strong passwords.
func TestPasswordStrength_GateAcceptsStrong(t *testing.T) {
	ok, _ := checkPasswordStrengthGate("K7$mQ9!xR2vLp#Wn4Bc")
	if !ok {
		t.Error("expected gate to accept strong password")
	}
}

// Test 14: Crack time is human-readable.
func TestPasswordStrength_CrackTimeFormat(t *testing.T) {
	result := EstimateStrength("K7$mQ9!xR2vLp#Wn4Bc")
	if result.CrackTime == "" {
		t.Error("expected non-empty crack time")
	}
}

// Test 15: Suggestions are generated.
func TestPasswordStrength_Suggestions(t *testing.T) {
	result := EstimateStrength("abc")
	if len(result.Suggestions) == 0 {
		t.Error("expected suggestions for weak password")
	}
}
