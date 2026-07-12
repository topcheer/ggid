package service

import (
	"fmt"
	"sync"
	"time"
)

type DelegationLink struct {
	Delegator    string    `json:"delegator"`
	Delegatee    string    `json:"delegatee"`
	Scopes       []string  `json:"scopes"`
	MaxDepth     int       `json:"max_depth"`
	ExpiresAt    time.Time `json:"expires_at"`
	CreatedAt    time.Time `json:"created_at"`
}

type DelegationChain struct {
	ChainID      string           `json:"chain_id"`
	AgentID      string           `json:"agent_id"`
	Links        []DelegationLink `json:"links"`
	TotalDepth   int              `json:"total_depth"`
	IsValid      bool             `json:"is_valid"`
}

type DelegationChainManager struct {
	mu     sync.RWMutex
	chains map[string]*DelegationChain // chainID -> chain
	byAgent map[string][]string        // agentID -> []chainID
	seq    int
}

func NewDelegationChainManager() *DelegationChainManager {
	return &DelegationChainManager{
		chains:   make(map[string]*DelegationChain),
		byAgent:  make(map[string][]string),
	}
}

func (d *DelegationChainManager) CreateDelegation(delegator, delegatee string, scopes []string, maxDepth int) *DelegationChain {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.seq++
	chainID := fmt.Sprintf("del_%d", d.seq)
	link := DelegationLink{
		Delegator:  delegator,
		Delegatee:  delegatee,
		Scopes:     scopes,
		MaxDepth:   maxDepth,
		ExpiresAt:  time.Now().Add(24 * time.Hour),
		CreatedAt:  time.Now(),
	}
	chain := &DelegationChain{
		ChainID:    chainID,
		AgentID:    delegatee,
		Links:      []DelegationLink{link},
		TotalDepth: 1,
		IsValid:    true,
	}
	d.chains[chainID] = chain
	d.byAgent[delegatee] = append(d.byAgent[delegatee], chainID)
	return chain
}

func (d *DelegationChainManager) ValidateDelegation(chainID string) (*DelegationChain, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	chain, ok := d.chains[chainID]
	if !ok {
		return nil, fmt.Errorf("delegation chain %s not found", chainID)
	}
	// Check depth
	if chain.TotalDepth > 0 && chain.Links[0].MaxDepth > 0 && chain.TotalDepth > chain.Links[0].MaxDepth {
		chain.IsValid = false
		return chain, fmt.Errorf("delegation depth %d exceeds max %d", chain.TotalDepth, chain.Links[0].MaxDepth)
	}
	// Check expiry
	for _, link := range chain.Links {
		if time.Now().After(link.ExpiresAt) {
			chain.IsValid = false
			return chain, fmt.Errorf("delegation expired")
		}
	}
	// Check scopes non-empty
	for _, link := range chain.Links {
		if len(link.Scopes) == 0 {
			chain.IsValid = false
			return chain, fmt.Errorf("delegation has empty scopes")
		}
	}
	chain.IsValid = true
	return chain, nil
}

func (d *DelegationChainManager) RevokeDelegation(chainID string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	chain, ok := d.chains[chainID]
	if !ok {
		return fmt.Errorf("chain not found")
	}
	chain.IsValid = false
	return nil
}

func (d *DelegationChainManager) GetDelegationChain(agentID string) []*DelegationChain {
	d.mu.RLock()
	defer d.mu.RUnlock()
	chainIDs := d.byAgent[agentID]
	var chains []*DelegationChain
	for _, id := range chainIDs {
		if c, ok := d.chains[id]; ok {
			chains = append(chains, c)
		}
	}
	return chains
}