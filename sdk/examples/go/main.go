//go:build ignore
// +build ignore

// GGID Go SDK Quickstart — JWT login + verify in <20 lines.
//
// Run:  GGID_URL=https://ggid.iot2.win go run main.go
// Local: GGID_URL=http://localhost:8080 go run main.go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	ggid "github.com/ggid/ggid/sdk/go/ggid"
)

func main() {
	url := os.Getenv("GGID_URL")
	if url == "" {
		url = "https://ggid.iot2.win"
	}
	tenantID := "00000000-0000-0000-0000-000000000001"

	ctx := context.Background()

	// 1. Create client
	client := ggid.NewClient(url, ggid.WithTenantID(tenantID))

	// 2. Login to get JWT
	tokens, err := client.Login(ctx, "admin", "Admin@123456")
	if err != nil {
		log.Fatalf("Login failed: %v", err)
	}
	fmt.Printf("Login OK — access token: %d chars\n", len(tokens.AccessToken))

	// 3. Verify the token
	claims, err := client.VerifyToken(ctx, tokens.AccessToken)
	if err != nil {
		log.Fatalf("Verify failed: %v", err)
	}
	fmt.Printf("Verified — user: %s, subject: %s\n", claims.Username, claims.Subject)
	fmt.Println("Quickstart complete!")
}
