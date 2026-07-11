// GGID Go SDK Quickstart — JWT login + verify in <20 lines.
//
// Usage:
//   GGID_URL=https://ggid.iot2.win go run sdk/examples/go-quickstart/quickstart.go
//   GGID_URL=http://localhost:8080 go run sdk/examples/go-quickstart/quickstart.go
//
//go:build ignore

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

	// 1. Create client — WithJWKS enables token verification
	client := ggid.NewClient(url,
		ggid.WithTenantID(tenantID),
		ggid.WithJWKS(url+"/.well-known/jwks.json"))

	// 2. Login to get JWT
	tokens, err := client.Login(ctx, "sdk_test_user", "Xk9#Zm2!vQ7nRp")
	if err != nil {
		log.Fatalf("Login failed: %v", err)
	}
	fmt.Printf("Login OK — access token: %d chars\n", len(tokens.AccessToken))

	// 3. Verify the token (checks signature via JWKS)
	claims, err := client.VerifyToken(ctx, tokens.AccessToken)
	if err != nil {
		log.Fatalf("Verify failed: %v", err)
	}
	fmt.Printf("Verified — subject: %v\n", claims["sub"])
	fmt.Println("Quickstart complete!")
}
