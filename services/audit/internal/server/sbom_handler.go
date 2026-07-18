package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

type SBOMComponent struct {
	BOMRef       string            `json:"bom-ref"`
	Type         string            `json:"type"`
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Licenses     []string          `json:"licenses"`
	Purl         string            `json:"purl"`
	Supplier     string            `json:"supplier"`
	Vulnerabilities []SBOMVuln     `json:"vulnerabilities"`
}

type SBOMVuln struct {
	ID         string `json:"id"`
	Severity   string `json:"severity"`
	Description string `json:"description"`
}

type CycloneDXBOM struct {
	BOMFormat   string         `json:"bomFormat"`
	SpecVersion string         `json:"specVersion"`
	SerialNumber string        `json:"serialNumber"`
	Version     int            `json:"version"`
	Metadata    map[string]any `json:"metadata"`
	Components  []SBOMComponent `json:"components"`
}

var sbomComponents = []SBOMComponent{
	{BOMRef: "pkg:generic/ggid/auth@1.0", Type: "application", Name: "ggid-auth", Version: "1.0.0", Licenses: []string{"Apache-2.0"}, Purl: "pkg:generic/ggid/auth@1.0", Supplier: "GGID", Vulnerabilities: []SBOMVuln{}},
	{BOMRef: "pkg:generic/ggid/gateway@1.0", Type: "application", Name: "ggid-gateway", Version: "1.0.0", Licenses: []string{"Apache-2.0"}, Purl: "pkg:generic/ggid/gateway@1.0", Supplier: "GGID", Vulnerabilities: []SBOMVuln{}},
	{BOMRef: "pkg:generic/ggid/policy@1.0", Type: "application", Name: "ggid-policy", Version: "1.0.0", Licenses: []string{"Apache-2.0"}, Purl: "pkg:generic/ggid/policy@1.0", Supplier: "GGID", Vulnerabilities: []SBOMVuln{}},
	{BOMRef: "pkg:golang/github.com/golang-jwt/jwt@5.2.1", Type: "library", Name: "github.com/golang-jwt/jwt", Version: "5.2.1", Licenses: []string{"MIT"}, Purl: "pkg:golang/github.com/golang-jwt/jwt@5.2.1", Supplier: "golang-jwt", Vulnerabilities: []SBOMVuln{}},
	{BOMRef: "pkg:golang/github.com/redis/go-redis@9.7.0", Type: "library", Name: "github.com/redis/go-redis", Version: "9.7.0", Licenses: []string{"BSD-2-Clause"}, Purl: "pkg:golang/github.com/redis/go-redis@9.7.0", Supplier: "redis", Vulnerabilities: []SBOMVuln{}},
}

func (s *HTTPServer) handleSBOM(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	bom := CycloneDXBOM{
		BOMFormat:    "CycloneDX",
		SpecVersion:  "1.5",
		SerialNumber: "urn:uuid:ggid-sbom-" + time.Now().Format("20060102"),
		Version:      1,
		Metadata:     map[string]any{"timestamp": time.Now().Format(time.RFC3339), "tool": "ggid-sbom-gen"},
		Components:   sbomComponents,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(bom)
}

func (s *HTTPServer) handleSBOMComponent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	parts := strings.Split(strings.TrimSuffix(r.URL.Path, "/"), "/")
	if len(parts) < 1 {
		writeJSONError(w, http.StatusBadRequest, "component name required")
		return
	}
	componentName := parts[len(parts)-1]
	for _, c := range sbomComponents {
		if c.Name == componentName {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(c)
			return
		}
	}
	writeJSONError(w, http.StatusNotFound, "component not found")
}