# Identity Token Prefetch Strategy

Preemptive refresh, background rotation, client-side prediction, grace period, offline fallback, and per-app integration patterns.

## Overview

Token prefetch proactively refreshes access tokens before they expire, eliminating latency spikes from on-demand refresh and improving user experience.

## Preemptive Refresh

### Background Timer

```javascript
class TokenManager {
  constructor(authClient) {
    this.authClient = authClient;
    this.refreshThreshold = 60; // Refresh 60s before expiry
    this.timer = null;
  }

  start() {
    this.scheduleRefresh();
  }

  scheduleRefresh() {
    const token = this.authClient.getAccessToken();
    const decoded = jwtDecode(token);
    const expiresAt = decoded.exp * 1000;
    const refreshAt = expiresAt - (this.refreshThreshold * 1000);
    const delay = refreshAt - Date.now();

    this.timer = setTimeout(async () => {
      try {
        await this.authClient.refreshToken();
        this.scheduleRefresh(); // Schedule next
      } catch (err) {
        // Fallback: refresh on next API call
        console.warn('Background refresh failed, will retry on next request');
      }
    }, delay);
  }

  stop() {
    clearTimeout(this.timer);
  }
}
```

## Client-Side Prediction

### Adaptive Threshold

```javascript
getAdaptiveThreshold() {
  const history = this.refreshHistory; // Last 10 refresh latencies
  const avgLatency = average(history) || 1000; // Default 1s
  const networkBuffer = 2000; // 2s buffer for slow networks
  return Math.max(avgLatency + networkBuffer, 30); // Min 30s
}
```

### Usage Pattern Prediction

```javascript
// Learn when user is active
class UsageTracker {
  constructor() {
    this.activeHours = new Set();
  }

  shouldPrefetch() {
    const hour = new Date().getHours();
    if (this.activeHours.has(hour)) {
      return true; // User typically active now → prefetch
    }
    return false; // User typically inactive → skip (save resources)
  }

  recordActivity() {
    this.activeHours.add(new Date().getHours());
  }
}
```

## Grace Period

```javascript
async makeRequest(url, options) {
  const token = this.authClient.getAccessToken();

  // Check if token is in grace period (expired but within grace window)
  const decoded = jwtDecode(token);
  const expiredBy = Date.now() / 1000 - decoded.exp;

  if (expiredBy > 0 && expiredBy < this.gracePeriod) {
    // Token expired but within grace — try request, refresh on 401
    try {
      return await fetch(url, { ...options, headers: { Authorization: `Bearer ${token}` } });
    } catch (err) {
      if (err.status === 401) {
        await this.authClient.refreshToken();
        return this.makeRequest(url, options); // Retry with new token
      }
      throw err;
    }
  }

  // Token valid or outside grace → normal request
  return fetch(url, { ...options, headers: { Authorization: `Bearer ${token}` } });
}
```

## Offline Fallback

When offline or refresh fails:

```javascript
class OfflineTokenStore {
  constructor() {
    this.queue = [];
  }

  async makeRequest(url, options) {
    try {
      return await fetch(url, options);
    } catch (err) {
      if (!navigator.onLine) {
        // Queue request for when back online
        return new Promise((resolve, reject) => {
          this.queue.push({ url, options, resolve, reject });
        });
      }
      throw err;
    }
  }

  onOnline() {
    // Process queued requests
    this.queue.forEach(async ({ url, options, resolve, reject }) => {
      try {
        // Refresh token first
        await this.authClient.refreshToken();
        const response = await fetch(url, options);
        resolve(response);
      } catch (err) {
        reject(err);
      }
    });
    this.queue = [];
  }
}
```

## Per-App Integration Patterns

### SPA (React)

```tsx
import { useEffect } from 'react';
import { useGGID } from '@ggid/react-sdk';

function App() {
  const { token, refreshToken, expiresAt } = useGGID();

  useEffect(() => {
    if (!expiresAt) return;

    const refreshIn = (expiresAt * 1000) - Date.now() - 60000;
    if (refreshIn <= 0) {
      refreshToken(); // Already expiring soon
      return;
    }

    const timer = setTimeout(refreshToken, refreshIn);
    return () => clearTimeout(timer);
  }, [expiresAt, refreshToken]);

  return <MainApp />;
}
```

### Mobile (iOS/Swift)

```swift
class TokenManager {
    private var refreshTimer: Timer?

    func scheduleRefresh(expiresAt: Date) {
        let refreshDate = expiresAt.addingTimeInterval(-60) // 60s before
        let interval = refreshDate.timeIntervalSinceNow

        refreshTimer = Timer.scheduledTimer(withTimeInterval: interval, repeats: false) { _ in
            Task {
                try? await self.authClient.refreshToken()
            }
        }
    }
}
```

### Server-to-Server (Go)

```go
type ServerTokenManager struct {
    token   atomic.Value // string
    expires time.Time
    mu      sync.Mutex
}

func (m *ServerTokenManager) GetToken(ctx context.Context) (string, error) {
    // Check if token expires within 60s
    if time.Until(m.expires) < 60*time.Second {
        m.mu.Lock()
        defer m.mu.Unlock()

        // Double-check after acquiring lock
        if time.Until(m.expires) < 60*time.Second {
            token, exp, err := m.fetchNewToken(ctx)
            if err != nil { return "", err }
            m.token.Store(token)
            m.expires = exp
        }
    }
    return m.token.Load().string, nil
}
```

## Configuration

```yaml
token_prefetch:
  threshold_seconds: 60     # Refresh 60s before expiry
  retry_attempts: 3
  retry_backoff: [1s, 5s, 15s]
  grace_period_seconds: 10  # Allow expired token for 10s (network latency)
  offline_queue: true       # Queue requests when offline
  max_queue_size: 100
```

## Monitoring

| Metric | Target | Alert |
|--------|--------|-------|
| Background refresh success rate | >99% | <95% → network or server issue |
| On-demand refresh rate | <5% | >10% → prefetch not working |
| 401 errors | 0 | Any → token not refreshed in time |
| Offline queue depth | 0 | >50 → user offline long |

## See Also

- [OAuth Refresh Token Rotation](oauth-refresh-token-rotation.md)
- [Token Introspection Design](token-introspection-design.md)
- [SDK Integration Guide](sdk-integration-guide.md)
- [JWT Security Best Practices](jwt-security-best-practices.md)
