package middleware

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// KnownBotPatterns are User-Agent substrings that identify known crawlers.
var knownBotPatterns = []string{
	"googlebot", "bingbot", "slurp", "duckduckbot",
	"baiduspider", "yandexbot", "facebookexternalhit",
	"twitterbot", "linkedinbot", "whatsapp",
	"applebot", "petalbot", "semrush", "ahrefsbot",
}

// SuspiciousPatterns indicate potential malicious bots.
var suspiciousPatterns = []string{
	"sqlmap", "nikto", "nmap", "masscan", "dirbuster",
	"wpscan", "hydra", "metasploit", "burp",
}

// BotDetect blocks known malicious bots and tags detected crawlers.
func BotDetect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ua := strings.ToLower(r.Header.Get("User-Agent"))

		// Block suspicious bots
		for _, pattern := range suspiciousPatterns {
			if strings.Contains(ua, pattern) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte(`{"error":"access denied"}`))
				return
			}
		}

		// Tag known crawlers for downstream analytics
		for _, pattern := range knownBotPatterns {
			if strings.Contains(ua, pattern) {
				w.Header().Set("X-Bot-Detected", pattern)
				break
			}
		}

		next.ServeHTTP(w, r)
	})
}

// BehavioralBotDetect uses a sliding window to detect abnormal request rates.
type BehavioralBotDetect struct {
	window    time.Duration
	threshold int
	store     *botRateStore
}

type botRateStore struct {
	mu      sync.Mutex
	buckets map[string]*botRequestLog
}

type botRequestLog struct {
	count   int
	expires time.Time
}

// NewBehavioralBotDetect creates a behavioral bot detector.
// threshold = max requests per window per IP before challenge.
func NewBehavioralBotDetect(threshold int, window time.Duration) *BehavioralBotDetect {
	return &BehavioralBotDetect{
		window:    window,
		threshold: threshold,
		store: &botRateStore{
			buckets: make(map[string]*botRequestLog),
		},
	}
}

// Middleware tracks per-IP request rates and challenges high-volume IPs.
func (b *BehavioralBotDetect) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := extractClientIP(r)
		if ip == "" {
			next.ServeHTTP(w, r)
			return
		}

		b.store.mu.Lock()
		defer b.store.mu.Unlock()

		key := "bot:" + ip
		log, exists := b.store.buckets[key]
		now := time.Now()

		if !exists || now.After(log.expires) {
			b.store.buckets[key] = &botRequestLog{count: 1, expires: now.Add(b.window)}
			next.ServeHTTP(w, r)
			return
		}

		log.count++
		if log.count > b.threshold {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", strconv.Itoa(int(b.window.Seconds())))
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error":"rate limit exceeded — possible bot"}`))
			return
		}

		next.ServeHTTP(w, r)
	})
}
