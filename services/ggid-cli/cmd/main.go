// Package main implements the ggid-cli command line tool.
package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

var (
	gatewayURL = os.Getenv("GGID_GATEWAY_URL")
	token      = os.Getenv("GGID_TOKEN")
	tenantID   = os.Getenv("GGID_TENANT_ID")
)

func init() {
	if gatewayURL == "" {
		gatewayURL = "http://localhost:8080"
	}
	if tenantID == "" {
		tenantID = "00000000-0000-0000-0000-000000000001"
	}
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "login":
		cmdLogin(os.Args[2:])
	case "users":
		cmdUsers(os.Args[2:])
	case "roles":
		cmdRoles(os.Args[2:])
	case "audit":
		cmdAudit(os.Args[2:])
	case "whoami":
		cmdWhoami()
	case "version":
		fmt.Println("ggid-cli v0.1.0")
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Println(`ggid-cli — GGID Identity & Access Management CLI

Commands:
  login    --username X --password Y [--tenant default]
  users    [--page 1] [--size 20]
  roles
  audit    [--page 1] [--size 20]
  whoami
  version

Env: GGID_GATEWAY_URL, GGID_TOKEN, GGID_TENANT_ID`)
}

func cmdLogin(args []string) {
	fs := flag.NewFlagSet("login", flag.ExitOnError)
	user := fs.String("username", "", "username")
	pass := fs.String("password", "", "password")
	slug := fs.String("tenant", "default", "tenant slug")
	fs.Parse(args)

	if *user == "" || *pass == "" {
		fmt.Fprintln(os.Stderr, "error: --username and --password required")
		os.Exit(1)
	}

	body, _ := json.Marshal(map[string]string{"username": *user, "password": *pass, "tenant_slug": *slug})
	resp, err := http.Post(gatewayURL+"/api/v1/auth/login", "application/json", bytes.NewReader(body))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)
	if t, ok := result["access_token"].(string); ok {
		fmt.Println(t)
	} else {
		fmt.Fprintf(os.Stderr, "login failed: %v\n", result)
		os.Exit(1)
	}
}

func cmdUsers(args []string) {
	fs := flag.NewFlagSet("users", flag.ExitOnError)
	page := fs.Int("page", 1, "")
	size := fs.Int("size", 20, "")
	fs.Parse(args)
	pretty(apiGet(fmt.Sprintf("/api/v1/users?page=%d&page_size=%d", *page, *size)))
}

func cmdRoles(args []string) {
	pretty(apiGet("/api/v1/roles"))
}

func cmdAudit(args []string) {
	fs := flag.NewFlagSet("audit", flag.ExitOnError)
	page := fs.Int("page", 1, "")
	size := fs.Int("size", 20, "")
	fs.Parse(args)
	pretty(apiGet(fmt.Sprintf("/api/v1/audit/events?page=%d&page_size=%d", *page, *size)))
}

func cmdWhoami() {
	if token == "" {
		fmt.Fprintln(os.Stderr, "not logged in")
		os.Exit(1)
	}
	parts := strings.Split(token, ".")
	if len(parts) >= 2 {
		for len(parts[1])%4 != 0 {
			parts[1] += "="
		}
		if decoded, err := base64.URLEncoding.DecodeString(parts[1]); err == nil {
			var claims map[string]any
			json.Unmarshal(decoded, &claims)
			fmt.Printf("User:  %v\n", claims["sub"])
			fmt.Printf("Scope: %v\n", claims["scopes"])
		}
	}
	fmt.Printf("Gateway: %s\nTenant:  %s\n", gatewayURL, tenantID)
}

func apiGet(path string) any {
	req, _ := http.NewRequest("GET", gatewayURL+path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Tenant-ID", tenantID)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var data any
	json.Unmarshal(body, &data)
	return data
}

func pretty(data any) {
	b, _ := json.MarshalIndent(data, "", "  ")
	fmt.Println(string(b))
}
