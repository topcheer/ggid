package service

import (
	"testing"
)

func TestLogSMSSender_Send(t *testing.T) {
	s := &LogSMSSender{}
	if err := s.SendSMS("+1234567890", "Your OTP is 123456"); err != nil {
		t.Fatalf("LogSMSSender should not error: %v", err)
	}
}

func TestTwilioSMSSender_NotConfigured(t *testing.T) {
	s := &TwilioSMSSender{} // empty config
	err := s.SendSMS("+1234567890", "test")
	if err == nil {
		t.Fatal("should error when not configured")
	}
}

func TestNewSMSSenderFromEnv_Default(t *testing.T) {
	// No GGID_SMS_PROVIDER set → should return LogSMSSender.
	s := NewSMSSenderFromEnv()
	if _, ok := s.(*LogSMSSender); !ok {
		t.Errorf("expected LogSMSSender, got %T", s)
	}
}
