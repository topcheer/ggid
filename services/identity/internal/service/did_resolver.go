package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

type DIDDocument struct {
	ID              string         `json:"id"`
	AlsoKnownAs     []string       `json:"alsoKnownAs,omitempty"`
	VerificationMethod []VMMethod  `json:"verificationMethod"`
	Services        []DIDService   `json:"service,omitempty"`
	Raw             map[string]any `json:"raw,omitempty"`
	ResolvedAt      time.Time      `json:"resolvedAt"`
}

type VMMethod struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	Controller   string `json:"controller"`
	PublicKeyJwk map[string]string `json:"publicKeyJwk,omitempty"`
}

type DIDService struct {
	ID              string `json:"id"`
	Type            string `json:"type"`
	ServiceEndpoint string `json:"serviceEndpoint"`
}

type DIDResolver struct {
	mu    sync.RWMutex
	cache map[string]cachedDID
	ttl   time.Duration
}

type cachedDID struct {
	doc     *DIDDocument
	expires time.Time
}

func NewDIDResolver(ttl time.Duration) *DIDResolver {
	return &DIDResolver{cache: make(map[string]cachedDID), ttl: ttl}
}

func (r *DIDResolver) ResolveDID(did string) (*DIDDocument, error) {
	r.mu.RLock()
	if c, ok := r.cache[did]; ok && time.Now().Before(c.expires) {
		r.mu.RUnlock()
		return c.doc, nil
	}
	r.mu.RUnlock()

	parts := strings.SplitN(did, ":", 3)
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid DID format: %s", did)
	}
	method := parts[1]

	var doc *DIDDocument
	var err error
	switch method {
	case "web":
		doc, err = r.resolveDIDWeb(parts[2])
	case "key":
		doc, err = r.resolveDIDKey(parts[2])
	case "ion":
		doc, err = r.resolveDIDIon(parts[2])
	default:
		return nil, fmt.Errorf("unsupported DID method: %s", method)
	}
	if err != nil {
		return nil, err
	}

	r.mu.Lock()
	r.cache[did] = cachedDID{doc: doc, expires: time.Now().Add(r.ttl)}
	r.mu.Unlock()
	return doc, nil
}

func (r *DIDResolver) resolveDIDWeb(suffix string) (*DIDDocument, error) {
	domain := strings.ReplaceAll(suffix, "/", "/.well-known/")
	url := fmt.Sprintf("https://%s/.well-known/did.json", strings.SplitN(domain, "/", 2)[0])
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("did:web fetch failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("did:web returned %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	return &DIDDocument{
		ID:         fmt.Sprintf("did:web:%s", suffix),
		Raw:        raw,
		ResolvedAt: time.Now(),
	}, nil
}

func (r *DIDResolver) resolveDIDKey(suffix string) (*DIDDocument, error) {
	return &DIDDocument{
		ID:         fmt.Sprintf("did:key:%s", suffix),
		VerificationMethod: []VMMethod{
			{ID: fmt.Sprintf("did:key:%s#%s", suffix, suffix[:8]), Type: "Ed25519VerificationKey2020", Controller: fmt.Sprintf("did:key:%s", suffix)},
		},
		ResolvedAt: time.Now(),
	}, nil
}

func (r *DIDResolver) resolveDIDIon(suffix string) (*DIDDocument, error) {
	return &DIDDocument{
		ID:         fmt.Sprintf("did:ion:%s", suffix),
		VerificationMethod: []VMMethod{
			{ID: fmt.Sprintf("did:ion:%s#%s", suffix, suffix[:8]), Type: "JsonWebKey2020", Controller: fmt.Sprintf("did:ion:%s", suffix)},
		},
		ResolvedAt: time.Now(),
	}, nil
}