package webhooks

// newTestDeliverer returns an HTTP deliverer that allows loopback addresses.
// Use this in tests where httptest.NewServer binds to 127.0.0.1.
// In production, always use NewHTTPDeliverer() which blocks loopback.
func newTestDeliverer() *HTTPDeliverer {
	return NewSSRFSafeDeliverer(&SSRFConfig{
		BlockedCIDRs: []string{
			"10.0.0.0/8",
			"172.16.0.0/12",
			"192.168.0.0/16",
			"169.254.0.0/16",
			"fc00::/7",
			"fe80::/10",
		},
		AllowRedirects: false,
		AllowLoopback:  true,
		DialTimeout:    5 * 1e9, // 5s
	})
}
