// GGID Go SDK Quickstart — 5 minute integration
//
// Prerequisites:
//   1. GGID running on localhost:8080 (docker run -p 8080:8080 ggid/ggid-all-in-one:latest)
//   2. go get github.com/ggid/ggid/sdk/go
//
// Run: go run main.go
package main

import (
	"context"
	"fmt"
	"log"

	ggid "github.com/ggid/ggid/sdk/go"
)

func main() {
	// 1. Create client
	client := ggid.NewClient("http://localhost:8080")

	// 2. Login as admin
	ctx := context.Background()
	token, err := client.Login(ctx, &ggid.LoginRequest{
		Username: "admin", Password: "Admin@123456",
	})
	if err != nil { log.Fatalf("login failed: %v", err) }
	fmt.Println("✓ Logged in, token acquired")

	// 3. Get current user
	me, err := client.GetUser(ctx, token.UserID, token.AccessToken)
	if err != nil { log.Fatalf("get user failed: %v", err) }
	fmt.Printf("✓ User: %s (%s)\n", me.DisplayName, me.Email)

	// 4. Create an OAuth client
	oauth, err := client.CreateOAuthClient(ctx, token.AccessToken, &ggid.OAuthClient{
		Name: "my-app", RedirectURIs: []string{"http://localhost:3000/callback"},
	})
	if err != nil { log.Fatalf("create oauth failed: %v", err) }
	fmt.Printf("✓ OAuth client created: %s (secret: %s)\n", oauth.ClientID, oauth.ClientSecret)
}
