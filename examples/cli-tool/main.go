// GGID CLI Tool Demo — demonstrates device code flow authentication.
//
// This CLI tool shows how to authenticate a command-line application
// with GGID using the OAuth 2.0 Device Authorization Grant (RFC 8628).
//
// Usage:
//   go run main.go login    — Start device code flow
//   go run main.go whoami   — Show current user info
//   go run main.go token     — Print current access token
//   go run main.go logout    — Clear stored credentials

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	ggidURL    = "https://ggid.iot2.win"
	tenantID   = "00000000-0000-0000-0000-000000000001"
	clientID   = "gcid__sbYZX3_2aJ4eDz-Oy1qRQ"
	tokenFile  = ".ggid-cli-token.json"
)

type TokenData struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type DeviceAuthResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	Error        string `json:"error,omitempty"`
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "login":
		login()
	case "whoami":
		whoami()
	case "token":
		showToken()
	case "logout":
		logout()
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`GGID CLI Demo — Device Code Flow Authentication

Usage:
  ggid-cli login    Start device code flow authentication
  ggid-cli whoami   Show current user info
  ggid-cli token    Print current access token
  ggid-cli logout   Clear stored credentials

Environment:
  GGID_URL       GGID gateway URL (default: https://ggid.iot2.win)
  GGID_TENANT    Tenant ID (default: 00000000-0000-0000-0000-000000000001)`)
}

func login() {
	fmt.Println("Starting GGID Device Code flow...")
	fmt.Println()

	// Step 1: Request device authorization
	deviceResp, err := requestDeviceAuth()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("  Go to: %s\n", deviceResp.VerificationURI)
	fmt.Printf("  Enter code: %s\n", deviceResp.UserCode)
	fmt.Println()
	fmt.Println("Waiting for authentication...")

	// Step 2: Poll for token
	interval := deviceResp.Interval
	if interval == 0 {
		interval = 5
	}

	deadline := time.Now().Add(time.Duration(deviceResp.ExpiresIn) * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(time.Duration(interval) * time.Second)

		tokenResp, err := pollToken(deviceResp.DeviceCode)
		if err != nil {
			fmt.Printf("Error polling: %v\n", err)
			continue
		}

		if tokenResp.Error == "authorization_pending" {
			continue
		}
		if tokenResp.Error == "slow_down" {
			interval += 5
			continue
		}
		if tokenResp.Error != "" {
			fmt.Printf("Authentication failed: %s\n", tokenResp.Error)
			os.Exit(1)
		}

		// Success — save token
		tokenData := TokenData{
			AccessToken:  tokenResp.AccessToken,
			RefreshToken: tokenResp.RefreshToken,
			ExpiresAt:    time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
		}
		if err := saveToken(tokenData); err != nil {
			fmt.Printf("Error saving token: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("\nAuthentication successful!")
		fmt.Printf("Token saved to ~/.%s\n", tokenFile)
		return
	}

	fmt.Println("Authentication timed out.")
	os.Exit(1)
}

func whoami() {
	tokenData, err := loadToken()
	if err != nil {
		fmt.Println("Not logged in. Run 'ggid-cli login' first.")
		os.Exit(1)
	}

	if time.Now().After(tokenData.ExpiresAt) {
		fmt.Println("Token expired. Run 'ggid-cli login' again.")
		os.Exit(1)
	}

	// Call userinfo endpoint
	req, _ := http.NewRequest("GET", ggidURL+"/api/v1/oauth/userinfo", nil)
	req.Header.Set("Authorization", "Bearer "+tokenData.AccessToken)
	req.Header.Set("X-Tenant-ID", tenantID)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("Error (HTTP %d): %s\n", resp.StatusCode, string(body))
		os.Exit(1)
	}

	var userInfo map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&userInfo)

	fmt.Println("Current User:")
	fmt.Printf("  User ID:  %s\n", userInfo["sub"])
	if name, ok := userInfo["name"]; ok {
		fmt.Printf("  Name:     %s\n", name)
	}
	if email, ok := userInfo["email"]; ok {
		fmt.Printf("  Email:    %s\n", email)
	}
	if roles, ok := userInfo["roles"]; ok {
		fmt.Printf("  Roles:    %v\n", roles)
	}
	fmt.Printf("  Expires:  %s\n", tokenData.ExpiresAt.Format(time.RFC3339))
}

func showToken() {
	tokenData, err := loadToken()
	if err != nil {
		fmt.Println("Not logged in. Run 'ggid-cli login' first.")
		os.Exit(1)
	}
	fmt.Println(tokenData.AccessToken)
}

func logout() {
	path := getTokenPath()
	if err := os.Remove(path); err != nil {
		fmt.Println("Already logged out.")
		return
	}
	fmt.Println("Logged out successfully.")
}

func requestDeviceAuth() (*DeviceAuthResponse, error) {
	body := fmt.Sprintf("client_id=%s&scope=openid+profile+email", clientID)
	req, _ := http.NewRequest("POST", ggidURL+"/api/v1/oauth/device_authorization", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Tenant-ID", tenantID)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("device auth failed (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	var deviceResp DeviceAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&deviceResp); err != nil {
		return nil, err
	}
	return &deviceResp, nil
}

func pollToken(deviceCode string) (*TokenResponse, error) {
	body := fmt.Sprintf("grant_type=urn:ietf:params:oauth:grant-type:device_code&device_code=%s&client_id=%s", deviceCode, clientID)
	req, _ := http.NewRequest("POST", ggidURL+"/api/v1/oauth/token", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Tenant-ID", tenantID)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var tokenResp TokenResponse
	json.NewDecoder(resp.Body).Decode(&tokenResp)
	return &tokenResp, nil
}

func getTokenPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, tokenFile)
}

func saveToken(data TokenData) error {
	path := getTokenPath()
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(data)
}

func loadToken() (*TokenData, error) {
	path := getTokenPath()
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var data TokenData
	if err := json.NewDecoder(f).Decode(&data); err != nil {
		return nil, err
	}
	return &data, nil
}
