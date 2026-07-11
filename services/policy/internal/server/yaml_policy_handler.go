package httpserver

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

type YAMLPolicy struct {
	Name        string         `yaml:"name" json:"name"`
	Description string         `yaml:"description" json:"description"`
	Rules       []YAMLRule     `yaml:"rules" json:"rules"`
	Roles       []string       `yaml:"roles" json:"roles"`
	Bindings    []YAMLBinding  `yaml:"bindings" json:"bindings"`
}

type YAMLRule struct {
	Resource  string `yaml:"resource" json:"resource"`
	Action    string `yaml:"action" json:"action"`
	Effect    string `yaml:"effect" json:"effect"`
	Condition string `yaml:"condition,omitempty" json:"condition,omitempty"`
}

type YAMLBinding struct {
	Role     string `yaml:"role" json:"role"`
	Resource string `yaml:"resource" json:"resource"`
}

var (
	yamlPolicyMu sync.RWMutex
	yamlPolicies = make(map[string]*YAMLPolicy)
)

// POST /api/v1/policies/import-yaml — import declarative policy from YAML.
// GET /api/v1/policies/export-yaml — export all policies as YAML.
func (s *HTTPServer) handleYAMLPolicy(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var p YAMLPolicy
		body, _ := readBody(r)
		if err := yaml.Unmarshal(body, &p); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid YAML: "+err.Error())
			return
		}
		if p.Name == "" {
			p.Name = "policy-" + uuid.New().String()[:8]
		}
		id := uuid.New().String()
		yamlPolicyMu.Lock()
		yamlPolicies[id] = &p
		yamlPolicyMu.Unlock()
		writeJSON(w, http.StatusCreated, map[string]any{
			"id": id, "name": p.Name, "rules_count": len(p.Rules),
			"roles_count": len(p.Roles), "bindings_count": len(p.Bindings),
			"imported_at": time.Now().UTC().Format(time.RFC3339),
		})
	case http.MethodGet:
		yamlPolicyMu.RLock()
		all := make([]*YAMLPolicy, 0, len(yamlPolicies))
		for _, p := range yamlPolicies {
			all = append(all, p)
		}
		yamlPolicyMu.RUnlock()
		yamlData, _ := yaml.Marshal(map[string]any{"policies": all})
		w.Header().Set("Content-Type", "application/x-yaml")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, string(yamlData))
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// readBody reads the full request body.
func readBody(r *http.Request) ([]byte, error) {
	buf := make([]byte, 0, 4096)
	chunk := make([]byte, 4096)
	for {
		n, err := r.Body.Read(chunk)
		if n > 0 {
			buf = append(buf, chunk[:n]...)
		}
		if err != nil {
			break
		}
	}
	return buf, nil
}


