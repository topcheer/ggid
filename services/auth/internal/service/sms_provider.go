package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// --- LogSMSSender (dev mode, logs to stdout) ---

type LogSMSSender struct{}

func (s *LogSMSSender) SendSMS(to, message string) error {
	log.Printf("[DEV SMS] to=%s: %s", to, message)
	return nil
}

// --- TwilioSMSSender ---

// TwilioSMSSender sends SMS via Twilio REST API.
// Config: TWILIO_ACCOUNT_SID, TWILIO_AUTH_TOKEN, TWILIO_FROM_NUMBER
type TwilioSMSSender struct {
	accountSID string
	authToken  string
	fromNumber string
}

func NewTwilioSMSSender() *TwilioSMSSender {
	return &TwilioSMSSender{
		accountSID: os.Getenv("TWILIO_ACCOUNT_SID"),
		authToken:  os.Getenv("TWILIO_AUTH_TOKEN"),
		fromNumber: os.Getenv("TWILIO_FROM_NUMBER"),
	}
}

func (s *TwilioSMSSender) SendSMS(to, message string) error {
	if s.accountSID == "" || s.authToken == "" || s.fromNumber == "" {
		return fmt.Errorf("twilio not configured: TWILIO_ACCOUNT_SID/AUTH_TOKEN/FROM_NUMBER required")
	}

	apiURL := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", s.accountSID)

	data := url.Values{}
	data.Set("To", to)
	data.Set("From", s.fromNumber)
	data.Set("Body", message)

	req, err := http.NewRequest("POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.SetBasicAuth(s.accountSID, s.authToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("twilio API error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errResp map[string]any
		json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("twilio error %d: %v", resp.StatusCode, errResp["message"])
	}

	log.Printf("SMS sent via Twilio to %s", to)
	return nil
}

// --- AWSSNSSMSSender ---

// AWSSNSSMSSender sends SMS via AWS SNS.
// Config: AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, AWS_REGION (optional AWS_SESSION_TOKEN)
type AWSSNSSMSSender struct {
	accessKey string
	secretKey string
	region    string
}

func NewAWSSNSSMSSender() *AWSSNSSMSSender {
	return &AWSSNSSMSSender{
		accessKey: os.Getenv("AWS_ACCESS_KEY_ID"),
		secretKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
		region:    os.Getenv("AWS_REGION"),
	}
}

func (s *AWSSNSSMSSender) SendSMS(to, message string) error {
	if s.accessKey == "" || s.secretKey == "" {
		return fmt.Errorf("AWS SNS not configured: AWS_ACCESS_KEY_ID/SECRET_ACCESS_KEY required")
	}

	// Use AWS SNS publish API via HTTP (simple, no SDK dependency).
	// In production, this would use the AWS SDK or signed HTTP request.
	// For now, we provide the interface and structure — actual AWS SigV4
	// signing requires the AWS SDK or a signing library.
	//
	// This implementation logs the intent and returns nil in dev mode.
	// Production deployment would use: sns.Publish(phoneNumber=to, message=message)

	log.Printf("SMS via AWS SNS to=%s region=%s (requires SDK for SigV4 signing)", to, s.region)
	return fmt.Errorf("AWS SNS SMS requires AWS SDK for SigV4 signing — configure in deployment")
}

// --- Factory ---

// NewSMSSenderFromEnv creates the configured SMS sender based on GGID_SMS_PROVIDER.
// Options: "twilio", "sns", "log" (default).
func NewSMSSenderFromEnv() SMSSender {
	provider := os.Getenv("GGID_SMS_PROVIDER")
	switch strings.ToLower(provider) {
	case "twilio":
		log.Println("SMS provider: Twilio")
		return NewTwilioSMSSender()
	case "sns":
		log.Println("SMS provider: AWS SNS")
		return NewAWSSNSSMSSender()
	default:
		if os.Getenv("GGID_ENV") == "production" && provider == "" {
			log.Println("WARNING: no GGID_SMS_PROVIDER set in production — SMS will be logged only")
		}
		return &LogSMSSender{}
	}
}

// Ensure SMSSender implementations satisfy the interface.
var (
	_ SMSSender = (*LogSMSSender)(nil)
	_ SMSSender = (*TwilioSMSSender)(nil)
	_ SMSSender = (*AWSSNSSMSSender)(nil)
)

// suppress unused import
var _ = bytes.NewBuffer
