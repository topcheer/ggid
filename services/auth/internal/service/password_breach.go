package service

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// CheckPasswordBreach checks if a password has been found in known data breaches
// using the HIBP k-anonymity model (haveibeenpwned.com API).
// Only the first 5 characters of the SHA-1 hash are sent to the API.
// Returns an error if the password has been breached.
func (ps *PasswordService) CheckPasswordBreach(ctx context.Context, password string) error {
	// Compute SHA-1 hash of the password.
	h := sha1.Sum([]byte(password))
	hash := strings.ToUpper(hex.EncodeToString(h[:]))

	// k-anonymity: send only first 5 chars to the API.
	prefix := hash[:5]
	suffix := hash[5:]

	// Query the HIBP API.
	url := "https://api.pwnedpasswords.com/range/" + prefix
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		// Don't block registration if the request can't be created.
		return nil
	}
	req.Header.Set("User-Agent", "GGID-IAM")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// Don't block registration if the API is unreachable.
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// API error — don't block registration.
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	// Parse the response: each line is "SUFFIX:COUNT".
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 && parts[0] == suffix {
			return fmt.Errorf("password has been found in %s data breaches", strings.TrimSpace(parts[1]))
		}
	}

	return nil
}
