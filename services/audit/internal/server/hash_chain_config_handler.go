package httpserver

import (
	"encoding/json"
	"net/http"
)

type AuditHashChainConfig struct {
	ChainAlgorithm       string   `json:"chain_algorithm"`
	AnchorIntervalBlocks int      `json:"anchor_interval_blocks"`
	CheckpointFrequency  int      `json:"checkpoint_frequency"`
	TamperDetectionMode  string   `json:"tamper_detection_mode"`
	AlertOnTamper        []string `json:"alert_on_tamper"`
	RetentionProofCount  int      `json:"retention_proof_count"`
}

var globalAuditHashChainConfig = &AuditHashChainConfig{
	ChainAlgorithm:       "sha256",
	AnchorIntervalBlocks: 1000,
	CheckpointFrequency:  100,
	TamperDetectionMode:  "continuous",
	AlertOnTamper:        []string{"webhook", "siem"},
	RetentionProofCount:  10000,
}

func (s *HTTPServer) handleAuditHashChainConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(globalAuditHashChainConfig)
	case http.MethodPut:
		var cfg AuditHashChainConfig
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
			return
		}
		if cfg.AnchorIntervalBlocks < 1 {
			http.Error(w, `{"error":"anchor_interval_blocks must be at least 1"}`, http.StatusBadRequest)
			return
		}
		globalAuditHashChainConfig = &cfg
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(cfg)
	default:
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	}
}
