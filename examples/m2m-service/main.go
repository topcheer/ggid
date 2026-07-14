package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

func main() {
	mode := flag.String("mode", "a", "Service mode: 'a' (caller) or 'b' (callee)")
	flag.Parse()

	ggidURL := os.Getenv("GGID_URL")
	if ggidURL == "" {
		ggidURL = "https://ggid.iot2.win"
	}

	tenantID := os.Getenv("GGID_TENANT_ID")
	if tenantID == "" {
		tenantID = "00000000-0000-0000-0000-000000000001"
	}

	port := os.Getenv("PORT")

	switch *mode {
	case "a":
		if port == "" {
			port = "5001"
		}
		clientID := os.Getenv("CLIENT_ID")
		clientSecret := os.Getenv("CLIENT_SECRET")
		if clientID == "" || clientSecret == "" {
			log.Fatal("CLIENT_ID and CLIENT_SECRET environment variables are required for service-a")
		}

		serviceBURL := os.Getenv("SERVICE_B_URL")
		if serviceBURL == "" {
			serviceBURL = "http://localhost:5002"
		}

		ggidClient := NewGGIDClient(ggidURL, tenantID, clientID, clientSecret)
		log.Printf("[service-a] GGID URL: %s", ggidURL)
		log.Printf("[service-a] Client ID: %s", clientID)
		startServiceA(port, ggidClient, serviceBURL)

	case "b":
		if port == "" {
			port = "5002"
		}
		jwksCache := NewJWKSKeyCache(ggidURL)
		log.Printf("[service-b] GGID URL: %s", ggidURL)
		startServiceB(port, jwksCache, tenantID)

	default:
		fmt.Fprintf(os.Stderr, "Invalid mode: %s (use 'a' or 'b')\n", *mode)
		os.Exit(1)
	}
}
