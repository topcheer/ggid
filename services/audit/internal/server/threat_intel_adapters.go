package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ggid/ggid/services/audit/internal/repository"
	"github.com/google/uuid"
)

// IntelAdapter fetches threat indicators from an external source.
type IntelAdapter interface {
	Fetch(ctx context.Context, client *http.Client) ([]repository.ThreatIndicator, error)
	SourceType() string
}

// getAdapter returns the appropriate adapter for a source type.
func getAdapter(sourceType string) IntelAdapter {
	switch strings.ToLower(sourceType) {
	case "ip":
		return &AbuseIPDBAdapter{}
	case "domain", "url":
		return &OTXAdapter{}
	default:
		return &OTXAdapter{}
	}
}

// AbuseIPDBAdapter fetches IP reputation from AbuseIPDB.
type AbuseIPDBAdapter struct {
	APIKey     string
	SourceID   uuid.UUID
	TenantID   uuid.UUID
	Endpoint   string
}

func (a *AbuseIPDBAdapter) SourceType() string { return "ip" }

func (a *AbuseIPDBAdapter) Fetch(ctx context.Context, client *http.Client) ([]repository.ThreatIndicator, error) {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	endp := a.Endpoint
	if endp == "" {
		endp = "https://api.abuseipdb.com/api/v2/blacklist"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endp, nil)
	if err != nil {
		return nil, fmt.Errorf("abuseipdb request: %w", err)
	}
	if a.APIKey != "" {
		req.Header.Set("Key", a.APIKey)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("abuseipdb fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("abuseipdb status %d", resp.StatusCode)
	}

	var body struct {
		Data []struct {
			IPAddress          string `json:"ipAddress"`
			AbuseConfidenceScore int  `json:"abuseConfidenceScore"`
			CountryCode        string `json:"countryCode"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("abuseipdb decode: %w", err)
	}

	var indicators []repository.ThreatIndicator
	for _, entry := range body.Data {
		severity := "low"
		if entry.AbuseConfidenceScore >= 90 {
			severity = "critical"
		} else if entry.AbuseConfidenceScore >= 75 {
			severity = "high"
		} else if entry.AbuseConfidenceScore >= 50 {
			severity = "medium"
		}
		indicators = append(indicators, repository.ThreatIndicator{
			ID:             uuid.New(),
			TenantID:       a.TenantID,
			SourceID:       a.SourceID,
			IndicatorType:  "ip",
			IndicatorValue: entry.IPAddress,
			Severity:       severity,
			Confidence:     entry.AbuseConfidenceScore,
			ExpiresAt:      expiryTime(24 * time.Hour),
			Metadata:       map[string]any{"country": entry.CountryCode, "source": "abuseipdb"},
		})
	}
	return indicators, nil
}

// OTXAdapter fetches indicators from AlienVault OTX.
type OTXAdapter struct {
	APIKey    string
	SourceID  uuid.UUID
	TenantID  uuid.UUID
	Endpoint  string
}

func (o *OTXAdapter) SourceType() string { return "domain" }

func (o *OTXAdapter) Fetch(ctx context.Context, client *http.Client) ([]repository.ThreatIndicator, error) {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	endp := o.Endpoint
	if endp == "" {
		endp = "https://otx.alienvault.com/api/v1/indicators/export"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endp, nil)
	if err != nil {
		return nil, fmt.Errorf("otx request: %w", err)
	}
	if o.APIKey != "" {
		req.Header.Set("X-OTX-API-KEY", o.APIKey)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("otx fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("otx status %d", resp.StatusCode)
	}

	var body struct {
		Results []struct {
			Indicator string `json:"indicator"`
			Type      string `json:"type"`
			Title     string `json:"title"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("otx decode: %w", err)
	}

	var indicators []repository.ThreatIndicator
	for _, r := range body.Results {
		indType := r.Type
		if indType == "" {
			indType = "domain"
		}
		indicators = append(indicators, repository.ThreatIndicator{
			ID:             uuid.New(),
			TenantID:       o.TenantID,
			SourceID:       o.SourceID,
			IndicatorType:  indType,
			IndicatorValue: r.Indicator,
			Severity:       "medium",
			Confidence:     60,
			ExpiresAt:      expiryTime(48 * time.Hour),
			Metadata:       map[string]any{"title": r.Title, "source": "otx"},
		})
	}
	return indicators, nil
}

// expiryTime returns a pointer to now + ttl.
func expiryTime(ttl time.Duration) *time.Time {
	t := time.Now().Add(ttl)
	return &t
}
