package service

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

type VerifiableCredential struct {
	Context        []string         `json:"@context"`
	ID             string           `json:"id"`
	Type           []string         `json:"type"`
	Issuer         string           `json:"issuer"`
	Subject        string           `json:"credentialSubject"`
	IssuanceDate   time.Time        `json:"issuanceDate"`
	ExpirationDate time.Time        `json:"expirationDate"`
	Claims         map[string]any   `json:"claims"`
	Proof          *VCProof         `json:"proof,omitempty"`
	Revoked        bool             `json:"revoked,omitempty"`
	RevokedReason  string           `json:"revokedReason,omitempty"`
}

type VCProof struct {
	Type         string    `json:"type"`
	Created      time.Time `json:"created"`
	VerificationMethod string `json:"verificationMethod"`
	ProofValue   string    `json:"proofValue"`
	ProofPurpose string    `json:"proofPurpose"`
}

type VCIssuer struct {
	mu      sync.RWMutex
	issued  map[string]*VerifiableCredential
	revoked map[string]bool
	keys    map[string]ed25519.PrivateKey // issuerDID -> private key
	pubKeys map[string]ed25519.PublicKey
	seq     int
}

func NewVCIssuer() *VCIssuer {
	return &VCIssuer{
		issued:  make(map[string]*VerifiableCredential),
		revoked: make(map[string]bool),
		keys:    make(map[string]ed25519.PrivateKey),
		pubKeys: make(map[string]ed25519.PublicKey),
	}
}

func (vi *VCIssuer) EnsureKey(issuerDID string) error {
	vi.mu.Lock()
	defer vi.mu.Unlock()
	if _, ok := vi.keys[issuerDID]; ok {
		return nil
	}
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("key generation failed: %w", err)
	}
	vi.keys[issuerDID] = priv
	vi.pubKeys[issuerDID] = pub
	return nil
}

func (vi *VCIssuer) IssueVC(issuerDID, subjectDID, credentialType string, claims map[string]any) (*VerifiableCredential, error) {
	if err := vi.EnsureKey(issuerDID); err != nil {
		return nil, err
	}
	vi.mu.Lock()
	defer vi.mu.Unlock()
	vi.seq++
	vc := &VerifiableCredential{
		Context:        []string{"https://www.w3.org/ns/credentials/v2"},
		ID:             fmt.Sprintf("urn:vc:ggid:%d", vi.seq),
		Type:           []string{"VerifiableCredential", credentialType},
		Issuer:         issuerDID,
		Subject:        subjectDID,
		IssuanceDate:   time.Now(),
		ExpirationDate: time.Now().AddDate(1, 0, 0),
		Claims:         claims,
	}
	vi.issued[vc.ID] = vc
	return vc, nil
}

func (vi *VCIssuer) SignVC(vc *VerifiableCredential, issuerDID string) error {
	vi.mu.RLock()
	priv, ok := vi.keys[issuerDID]
	vi.mu.RUnlock()
	if !ok {
		return fmt.Errorf("no key for issuer %s", issuerDID)
	}
	payload, err := json.Marshal(struct {
		Context        []string       `json:"@context"`
		ID             string         `json:"id"`
		Type           []string       `json:"type"`
		Issuer         string         `json:"issuer"`
		Subject        string         `json:"credentialSubject"`
		IssuanceDate   time.Time      `json:"issuanceDate"`
		ExpirationDate time.Time      `json:"expirationDate"`
		Claims         map[string]any `json:"claims"`
	}{vc.Context, vc.ID, vc.Type, vc.Issuer, vc.Subject, vc.IssuanceDate, vc.ExpirationDate, vc.Claims})
	if err != nil {
		return err
	}
	sig := ed25519.Sign(priv, payload)
	vc.Proof = &VCProof{
		Type:               "Ed25519Signature2020",
		Created:            time.Now(),
		VerificationMethod: issuerDID + "#key-1",
		ProofValue:         fmt.Sprintf("%x", sig),
		ProofPurpose:       "assertionMethod",
	}
	return nil
}

func (vi *VCIssuer) VerifyVC(vc *VerifiableCredential, issuerDID string) error {
	vi.mu.RLock()
	defer vi.mu.RUnlock()
	if vi.revoked[vc.ID] {
		return fmt.Errorf("credential %s is revoked", vc.ID)
	}
	if time.Now().After(vc.ExpirationDate) {
		return fmt.Errorf("credential expired")
	}
	if vc.Proof == nil {
		return fmt.Errorf("no proof on credential")
	}
	if _, ok := vi.pubKeys[issuerDID]; !ok {
		return fmt.Errorf("unknown issuer key")
	}
	if !strings.HasPrefix(vc.Proof.VerificationMethod, issuerDID) {
		return fmt.Errorf("proof verification method does not match issuer")
	}
	return nil
}

func (vi *VCIssuer) RevokeVC(vcID, reason string) {
	vi.mu.Lock()
	defer vi.mu.Unlock()
	vi.revoked[vcID] = true
	if vc, ok := vi.issued[vcID]; ok {
		vc.Revoked = true
		vc.RevokedReason = reason
	}
}

func (vi *VCIssuer) ListIssuedVCs(issuerDID string) []*VerifiableCredential {
	vi.mu.RLock()
	defer vi.mu.RUnlock()
	var list []*VerifiableCredential
	for _, vc := range vi.issued {
		if vc.Issuer == issuerDID {
			list = append(list, vc)
		}
	}
	return list
}