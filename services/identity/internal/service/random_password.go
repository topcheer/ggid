package service

import (
	"crypto/rand"
	
	"math/big"
)

// GenerateRandomPassword generates a cryptographically random password
// that satisfies the default password policy (uppercase, lowercase, digit, special).
func GenerateRandomPassword() string {
	const (
		upper   = "ABCDEFGHJKLMNPQRSTUVWXYZ"
		lower   = "abcdefghijkmnopqrstuvwxyz"
		digit   = "23456789"
		special = "!@#$%^&*"
		all     = upper + lower + digit + special
	)
	length := 24
	result := make([]byte, length)

	// Ensure at least one of each required type
	sets := []string{upper, lower, digit, special}
	for i, set := range sets {
		idx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(set))))
		result[i] = set[idx.Int64()]
	}

	// Fill the rest randomly
	for i := len(sets); i < length; i++ {
		idx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(all))))
		result[i] = all[idx.Int64()]
	}

	// Shuffle
	for i := len(result) - 1; i > 0; i-- {
		jBig, _ := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		j := int(jBig.Int64())
		result[i], result[j] = result[j], result[i]
	}

	return string(result)
}

// avoid extra import

