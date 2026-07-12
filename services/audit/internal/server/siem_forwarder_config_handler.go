package httpserver

import (
	"encoding/json"
	"net/http"
)

type SIEMDestination struct {
	SIEMType      string `json:"siem_type"`
	Protocol      string `json:"protocol"`
	Host          string `json:"host"`
	Auth          string `json:"auth"`
	Format        string `json:"format"`
	BatchSize     int    `json:"batch_size"`
	FlushInterval int    `json:"flush_interval_seconds"`
}

type SIEMForwarderConfig struct {
	Destinations          []SIEMDestination `json:"destinations"`
	FilterRules           []string          `json:"filter_rules"`
	RetryPolicy           struct {
		MaxRetries  int `json:"max_retries"`
		BackoffSecs int `json:"backoff_seconds"`
	} `json:"retry_policy"`
	CircuitBreaker struct {
		Enabled          bool `json:"enabled"`
		FailureThreshold int  `json:"failure_threshold"`
		ResetTimeoutSecs int  `json:"reset_timeout_seconds"`
	} `json:"circuit_breaker"`
	HealthCheckIntervalSecs int `json:"health_check_interval_seconds"`
}

func (s *HTTPServer) handleSIEMForwarderConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		result := SIEMForwarderConfig{}
		result.Destinations = []SIEMDestination{
			{SIEMType: "splunk", Protocol: "hec", Host: "https://splunk.example.com:8088", Auth: "token", Format: "json", BatchSize: 100, FlushInterval: 5},
			{SIEMType: "elastic", Protocol: "elasticsearch", Host: "https://elastic.example.com:9200", Auth: "api_key", Format: "ecs", BatchSize: 500, FlushInterval: 10},
			{SIEMType: "datadog", Protocol: "logs_api", Host: "https://http-intake.logs.datadoghq.com", Auth: "api_key", Format: "json", BatchSize: 200, FlushInterval: 5},
		}
		result.FilterRules = []string{"include:type=security", "include:type=auth", "exclude:severity=debug", "include:tenant_id!=null"}
		result.RetryPolicy.MaxRetries = 5
		result.RetryPolicy.BackoffSecs = 30
		result.CircuitBreaker.Enabled = true
		result.CircuitBreaker.FailureThreshold = 10
		result.CircuitBreaker.ResetTimeoutSecs = 60
		result.HealthCheckIntervalSecs = 30
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	case http.MethodPut:
		var req SIEMForwarderConfig
		json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": "updated"})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
