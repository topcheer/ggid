package middleware

import (
	"net/http"
	"strings"
	"sync"
)

// GeoRegion represents a geographic data center.
type GeoRegion struct {
	Name       string // e.g. "us-east", "eu-west", "ap-southeast"
	BackendURL string // e.g. "http://identity-us-east:8080"
	Countries  []string // ISO 3166-1 alpha-2 codes, e.g. ["US", "CA"]
	Latencies  map[string]int // country code → typical latency in ms (for sorting)
}

// GeoRouter routes requests to the nearest backend based on GeoIP data.
type GeoRouter struct {
	mu         sync.RWMutex
	regions    map[string]*GeoRegion // region name → region
	fallback   string                // fallback region name
	headerName string                // header to read country code from
}

// NewGeoRouter creates a GeoRouter with the given fallback region.
// The country code is read from the Cloudflare-style header
// "CF-IPCountry" by default.
func NewGeoRouter(fallbackRegion string) *GeoRouter {
	return &GeoRouter{
		regions:    make(map[string]*GeoRegion),
		fallback:   fallbackRegion,
		headerName: "CF-IPCountry",
	}
}

// AddRegion registers a geographic region.
func (g *GeoRouter) AddRegion(region *GeoRegion) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.regions[region.Name] = region
}

// SetHeaderName configures which request header carries the country code.
func (g *GeoRouter) SetHeaderName(name string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.headerName = name
}

// ResolveBackend returns the backend URL for the request's geo location.
// Falls back to the default region if no match is found.
func (g *GeoRouter) ResolveBackend(r *http.Request) string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	country := strings.ToUpper(strings.TrimSpace(r.Header.Get(g.headerName)))
	if country == "" {
		// Try alternate headers
		country = strings.ToUpper(strings.TrimSpace(r.Header.Get("X-GeoIP-Country")))
	}
	if country == "" {
		country = strings.ToUpper(strings.TrimSpace(r.Header.Get("X-A Country-Code")))
	}

	if country != "" {
		// Find the region that serves this country
		for _, region := range g.regions {
			for _, c := range region.Countries {
				if strings.ToUpper(c) == country {
					return region.BackendURL
				}
			}
		}
	}

	// Fallback
	if fb, ok := g.regions[g.fallback]; ok {
		return fb.BackendURL
	}
	return ""
}

// ResolveRegion returns the region name (not URL) for the request.
func (g *GeoRouter) ResolveRegion(r *http.Request) string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	country := strings.ToUpper(strings.TrimSpace(r.Header.Get(g.headerName)))
	if country == "" {
		country = strings.ToUpper(strings.TrimSpace(r.Header.Get("X-GeoIP-Country")))
	}

	if country != "" {
		for _, region := range g.regions {
			for _, c := range region.Countries {
				if strings.ToUpper(c) == country {
					return region.Name
				}
			}
		}
	}

	return g.fallback
}

// GeoRoutingMiddleware injects the resolved region into request context
// and adds an X-Served-By header to the response.
func GeoRoutingMiddleware(router *GeoRouter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			region := router.ResolveRegion(r)
			w.Header().Set("X-Served-By", region)
			next.ServeHTTP(w, r)
		})
	}
}

// Regions returns all configured region names.
func (g *GeoRouter) Regions() []string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	names := make([]string, 0, len(g.regions))
	for name := range g.regions {
		names = append(names, name)
	}
	return names
}

// FallbackRegion returns the fallback region name.
func (g *GeoRouter) FallbackRegion() string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.fallback
}
