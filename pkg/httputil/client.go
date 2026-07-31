// Package httputil provides shared HTTP client utilities for connection reuse.
package httputil

import (
	"net"
	"net/http"
	"time"
)

// DefaultClient is a shared HTTP client with tuned transport for general use.
// Reusing a single client across requests enables connection pooling and
// avoids the overhead of creating a new Transport per request.
var DefaultClient = &http.Client{
	Timeout: 30 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	},
}

// ShortTimeoutClient is a shared client with a short timeout for
// external API calls (JWKS fetch, breach check, webhook delivery, etc.).
var ShortTimeoutClient = &http.Client{
	Timeout: 10 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        50,
		MaxIdleConnsPerHost: 5,
		IdleConnTimeout:     90 * time.Second,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	},
}
