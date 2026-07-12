package service

import (
	"fmt"
	"sync"
	"time"
)

type PushedAuthRequest struct {
	RequestURI string    `json:"request_uri"`
	ClientID   string    `json:"client_id"`
	AuthParams map[string]any `json:"auth_params"`
	ExpiresIn  int       `json:"expires_in"`
	CreatedAt  time.Time `json:"created_at"`
}

type PARHandler struct {
	mu      sync.RWMutex
	requests map[string]*PushedAuthRequest
	ttl     time.Duration
	seq     int
}

func NewPARHandler() *PARHandler {
	return &PARHandler{
		requests: make(map[string]*PushedAuthRequest),
		ttl:      60 * time.Second,
	}
}

func (p *PARHandler) HandlePAR(clientID string, authRequest map[string]any) (*PushedAuthRequest, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.seq++
	uri := fmt.Sprintf("urn:ietf:params:oauth:request_uri:%d", p.seq)
	par := &PushedAuthRequest{
		RequestURI: uri,
		ClientID:   clientID,
		AuthParams: authRequest,
		ExpiresIn:  int(p.ttl.Seconds()),
		CreatedAt:  time.Now(),
	}
	p.requests[uri] = par
	return par, nil
}

func (p *PARHandler) GetPAR(requestURI string) (*PushedAuthRequest, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	par, ok := p.requests[requestURI]
	if !ok {
		return nil, fmt.Errorf("request_uri not found")
	}
	if time.Since(par.CreatedAt) > p.ttl {
		return nil, fmt.Errorf("request_uri expired")
	}
	return par, nil
}

func (p *PARHandler) ValidatePAR(requestURI, clientID string) error {
	par, err := p.GetPAR(requestURI)
	if err != nil {
		return err
	}
	if par.ClientID != clientID {
		return fmt.Errorf("client_id mismatch: request does not belong to client")
	}
	return nil
}

func (p *PARHandler) CleanupExpired() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	count := 0
	now := time.Now()
	for uri, par := range p.requests {
		if now.Sub(par.CreatedAt) > p.ttl {
			delete(p.requests, uri)
			count++
		}
	}
	return count
}

func (p *PARHandler) SetTTL(ttl time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.ttl = ttl
}