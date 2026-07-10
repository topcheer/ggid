# Observability for IAM Systems

> Research document for the GGID IAM platform — distributed tracing, structured logging,
> metrics, SLOs, alerting, and security observability. Includes gap analysis of the
> current GGID codebase with concrete action items.

---

## Table of Contents

1. [Three Pillars of Observability](#1-three-pillars-of-observability)
2. [Distributed Tracing with OpenTelemetry](#2-distributed-tracing-with-opentelemetry)
3. [Structured Logging](#3-structured-logging)
4. [Metrics (RED Method)](#4-metrics-red-method)
5. [SLO/SLI Definitions for IAM](#5-slosli-definitions-for-iam)
6. [Alerting Rules](#6-alerting-rules)
7. [Security Observability](#7-security-observability)
8. [GGID Observability Gap Analysis](#8-ggid-observability-gap-analysis)
9. [Gap Analysis & Recommendations](#9-gap-analysis--recommendations)

---

## 1. Three Pillars of Observability

The three pillars — **distributed tracing**, **structured logging**, and **metrics** —
are not interchangeable. Each answers a different class of question, and IAM systems
need all three because the failure modes span security forensics, performance
debugging, and operational health.

### Why All Three Matter for IAM

| Pillar | Primary Consumer | IAM Use Case | Example Question |
|--------|-----------------|--------------|-----------------|
| **Logs** | Security analysts, compliance auditors | Audit trail, forensic investigation | "Who accessed the user-management API at 3:47 AM from IP 10.0.0.5?" |
| **Traces** | Engineers debugging latency | Cross-service request flow | "Why does SAML SSO take 2.3 seconds when OIDC takes 200ms?" |
| **Metrics** | SRE on-call, auto-scaling, alerting | System health, SLO tracking | "What is the auth success rate for tenant X in the last 5 minutes?" |

### How They Correlate

The key to observability is **exemplar correlation** — the ability to pivot from a
metric spike to the specific log entries and traces that explain it:

```
Metric: auth_failures_total{tenant="acme", ip="10.0.0.5"} +47 in 5m
    → exemplar: trace_id=0af7651916cd43dd8448eb211c80319c
        → Trace: gateway → auth.VerifyCredentials → 401 (password mismatch)
            → Log: {trace_id: "0af7...", user: "admin@acme.com", event: "auth_failure", reason: "invalid_password"}
```

Three correlation mechanisms:

1. **Trace ID in logs** — every log line includes the trace ID from the request's
   span context, enabling log-to-trace and trace-to-log navigation.
2. **Exemplars in metrics** — Prometheus histograms can attach exemplars
   (trace_id + value) to individual observations, linking metric → trace.
3. **Metric tags** — labels like `tenant_id`, `endpoint`, `status_code` appear in
   all three pillars, enabling cross-pillar filtering.

For IAM specifically, **tenant_id** must propagate through all three pillars.
A security incident in tenant A should never require sifting through tenant B's data.

---

## 2. Distributed Tracing with OpenTelemetry

### Why OTel, Not Custom Tracing

GGID currently has a custom tracing implementation (`middleware/otel.go`) that
generates W3C traceparent headers and exports spans via OTLP/HTTP. While
functional, it lacks:

- **Auto-instrumentation** for HTTP servers/clients, gRPC, database drivers
- **Context propagation** via Go `context.Context` (uses custom key, not OTel API)
- **Ecosystem integration** — no SpanExporter plugins for Jaeger, Zipkin, Datadog
- **Baggage** support for cross-service attribute propagation

The official OpenTelemetry Go SDK (`go.opentelemetry.io/otel`) provides all of
these and is the CNCF standard.

### OTel SDK Setup for a Go Microservice

```go
package main

import (
	"context"
	"log"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

// initTracer sets up the OTel tracer provider with OTLP/HTTP exporter.
// The collector endpoint is configured via OTEL_EXPORTER_OTLP_ENDPOINT env var.
func initTracer(ctx context.Context, serviceName string) func() {
	// OTLP/HTTP exporter — sends to http://collector:4318/v1/traces
	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")),
		otlptracehttp.WithInsecure(), // use TLS in production
	)
	if err != nil {
		log.Fatalf("failed to create OTLP exporter: %v", err)
	}

	// Resource identifies the service in trace backends
	res, _ := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion("1.0.0"),
			attribute.String("deployment.environment", os.Getenv("ENV")),
		),
	)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter,
			sdktrace.WithBatchTimeout(5*time.Second),
			sdktrace.WithMaxExportBatchSize(512),
		),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(0.1)), // 10% sampling
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, // W3C TraceContext
		propagation.Baggage{},      // W3C Baggage
	))

	return func() {
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		_ = tp.Shutdown(ctx)
	}
}
```

### Auto-Instrumentation for HTTP and gRPC

```go
import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
)

// HTTP server with auto-instrumentation
func newHTTPServer(handler http.Handler) *http.Handler {
	return otelhttp.NewHandler(handler, "gateway",
		otelhttp.WithMessageEvents(otelhttp.ReadEvents, otelhttp.WriteEvents),
	)
}

// HTTP client with context propagation
func httpClientDo(ctx context.Context, url string) (*http.Response, error) {
	client := http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}
	return client.Do(otelhttp.WithTraceName(ctx, "proxy-to-backend"), "GET", url)
}

// gRPC server with auto-instrumentation
func newGRPCServer() *grpc.Server {
	return grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
	)
}

// gRPC client with context propagation
func newGRPCConn(target string) (*grpc.ClientConn, error) {
	return grpc.NewClient(target,
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
}
```

### IAM Request Trace: Gateway → Auth → DB → NATS

For a typical GGID login request, the trace tree looks like:

```
POST /api/v1/auth/login [gateway]  (root span, 250ms)
├── auth.VerifyCredentials [auth-service]  (180ms)
│   ├── pgpool.Query users WHERE username=? [auth-service]  (12ms)
│   └── bcrypt.CompareHashAndPassword [auth-service]  (155ms)
├── jwt.SignAccessToken [auth-service]  (8ms)
├── nats.Publish audit.login.success [audit]  (3ms)
└── redis.SET session:abc123 [auth-service]  (2ms)
```

Each span carries IAM-specific attributes:

```go
func traceLogin(ctx context.Context, username, tenantID string) {
	tracer := otel.Tracer("ggid/auth")
	ctx, span := tracer.Start(ctx, "auth.VerifyCredentials")
	defer span.End()

	span.SetAttributes(
		attribute.String("iam.username", username),       // NOT the password
		attribute.String("iam.tenant_id", tenantID),
		attribute.String("iam.auth_method", "password"),
	)

	// ... verify logic ...

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid credentials")
	}
}
```

### Context Propagation Across Services

W3C TraceContext propagation via the `traceparent` header is automatic with OTel.
The gateway injects the context, and each downstream service extracts it:

```go
// Gateway: inject trace context into outgoing HTTP request
otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))

// Auth service: extract trace context from incoming request
ctx = otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(r.Header))
```

**For GGID**: the existing custom `TracingMiddleware` in `middleware/otel.go` already
implements W3C traceparent parsing, but it stores the trace context in a custom
context key instead of using the OTel API. Migrating to the official SDK would
enable auto-instrumentation and eliminate the need for manual span management.

---

## 3. Structured Logging

### Why slog, Not log.Printf

GGID currently uses `log.Printf` for request logging (`middleware/middleware.go:79`):

```go
log.Printf("%s %s %d %d %s req=%s", r.Method, r.URL.Path, sr.status, sr.size, ...)
```

This is unstructured — it produces space-separated text that requires regex parsing.
Structured JSON logging with Go 1.21+ `slog` enables:

- **Machine-parseable** output for log aggregation (ELK, Loki, Datadog)
- **Field-level querying** — filter by `tenant_id`, `trace_id`, `event_type`
- **Consistent schema** across all services
- **Secret redaction** via a centralized handler

### Required Fields for IAM

Every IAM log line MUST include these fields for security forensics:

| Field | Source | Why Required |
|-------|--------|-------------|
| `timestamp` | `slog` auto | Event ordering, correlation |
| `level` | `slog` auto | Severity filtering |
| `trace_id` | OTel context | Pivot to trace/span |
| `tenant_id` | JWT claim / header | Multi-tenant isolation in queries |
| `user_id` | JWT claim | Accountability — who did what |
| `event_type` | Application | Categorization (auth.success, user.created, etc.) |
| `ip_address` | Request RemoteAddr | Geo-analysis, impossible travel detection |
| `user_agent` | Request header | Device fingerprinting, bot detection |
| `request_id` | X-Request-ID header | Single-request correlation across services |

### Go Code for Structured Logging Setup

```go
package observability

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"go.opentelemetry.io/otel/trace"
)

// IAMLogHandler wraps slog.JSONHandler with secret redaction.
type IAMLogHandler struct {
	inner  slog.Handler
	redact map[string]bool
}

var defaultRedactFields = map[string]bool{
	"password":           true,
	"token":              true,
	"access_token":       true,
	"refresh_token":      true,
	"authorization":      true,
	"secret":             true,
	"api_key":            true,
	"private_key":        true,
	"jws_signature":      true,
	"client_secret":      true,
}

func NewIAMLogger(level slog.Level) *slog.Logger {
	jsonHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
		// Add source file and line number
		AddSource: true,
	})
	wrapped := &IAMLogHandler{
		inner:  jsonHandler,
		redact: defaultRedactFields,
	}
	return slog.New(wrapped)
}

// Handle redacts sensitive fields before delegating to the inner handler.
func (h *IAMLogHandler) Handle(ctx context.Context, r slog.Record) error {
	// Inject trace_id and span_id from OTel context
	if spanCtx := trace.SpanContextFromContext(ctx); spanCtx.IsValid() {
		r.AddAttrs(
			slog.String("trace_id", spanCtx.TraceID().String()),
			slog.String("span_id", spanCtx.SpanID().String()),
		)
	}

	// Redact sensitive attributes
	var attrs []slog.Attr
	r.Attrs(func(a slog.Attr) bool {
		if h.redact[strings.ToLower(a.Key)] {
			a.Value = slog.StringValue("[REDACTED]")
		}
		attrs = append(attrs, a)
		return true
	})

	// Reconstruct record with redacted attrs
	r = slog.Record{}
	r.AddAttrs(attrs...)

	return h.inner.Handle(ctx, r)
}

func (h *IAMLogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *IAMLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &IAMLogHandler{inner: h.inner.WithAttrs(attrs), redact: h.redact}
}

func (h *IAMLogHandler) WithGroup(name string) slog.Handler {
	return &IAMLogHandler{inner: h.inner.WithGroup(name), redact: h.redact}
}

// RequestLogger middleware logs every request with IAM fields.
func RequestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sr := &statusRecorder{ResponseWriter: w, status: 200}

			next.ServeHTTP(sr, r)

			// Extract trace_id from request context (OTel or custom)
			traceID := trace.SpanContextFromContext(r.Context()).TraceID().String()

			logger.LogAttrs(r.Context(), slog.LevelInfo, "http_request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", sr.status),
				slog.Int("size", sr.size),
				slog.Duration("duration", time.Since(start)),
				slog.String("trace_id", traceID),
				slog.String("request_id", middleware.RequestIDFromCtx(r.Context())),
				slog.String("ip_address", extractIP(r)),
				slog.String("user_agent", r.UserAgent()),
				// tenant_id and user_id would come from JWT claims in context
				slog.String("tenant_id", tenantFromCtx(r.Context())),
				slog.String("user_id", userFromCtx(r.Context())),
				slog.String("event_type", "http.request"),
			)
		})
	}
}
```

### Log Levels for IAM

| Level | When to Use | IAM Example |
|-------|-------------|-------------|
| `DEBUG` | Development only, verbose tracing | "Redis cache hit for JWKS" |
| `INFO` | Normal operations, audit-worthy events | "User admin@acme.com logged in successfully" |
| `WARN` | Degraded but functional, suspicious activity | "Rate limit threshold 80% reached for IP 10.0.0.5" |
| `ERROR` | Operation failed, needs investigation | "Failed to verify SAML assertion: signature invalid" |

**Rule**: Never log at INFO or above in hot paths (per-request DB queries).
Use DEBUG for those and rely on log level configuration in production.

### Secret Redaction

IAM systems handle passwords, tokens, private keys, and certificates. The
redaction handler above intercepts known sensitive field names and replaces
values with `[REDACTED]`. Additional patterns to watch:

- JWT tokens in `Authorization` headers (redact the token value, keep `Bearer`)
- Connection strings containing passwords (`postgres://user:PASS@host`)
- LDAP bind DN passwords
- WebAuthn challenge/response data (may contain biometric-derived secrets)

---

## 4. Metrics (RED Method)

### The RED Method

The **RED method** (Rate, Errors, Duration) is the minimum viable metrics set
for any HTTP service:

| Metric | Type | What It Measures | GGID Example |
|--------|------|-----------------|-------------|
| **Rate** | Counter | Requests per second per endpoint | `POST /auth/login: 45/s` |
| **Errors** | Counter | Failed requests per endpoint | `401 responses: 12/s` |
| **Duration** | Histogram | Latency distribution per endpoint | `p99: 340ms` |

### Standard Prometheus Metrics (Already in GGID)

GGID's gateway middleware (`metrics.go`, `metrics_enhanced.go`) already defines:

```go
// metrics.go — basic RED metrics
requestsTotal   = CounterVec{labels: [method, path, status]}    // Rate
requestDuration = HistogramVec{labels: [method, path]}           // Duration
authFailures    = CounterVec{labels: [reason]}                   // Errors
activeSessions  = Gauge                                          // State

// metrics_enhanced.go — enriched metrics
ggid_http_request_duration_seconds  HistogramVec{[method, route, status]}
ggid_http_request_size_bytes        HistogramVec{[method, route]}
ggid_http_response_size_bytes       HistogramVec{[method, route, status]}
ggid_go_goroutines                  GaugeFunc
ggid_go_memory_alloc_bytes          GaugeFunc
ggid_go_gc_duration_seconds         GaugeFunc
ggid_go_gc_total                    CounterFunc
ggid_go_cpu_count                   GaugeFunc
ggid_go_heap_objects                GaugeFunc
```

### Custom IAM Metrics (Missing)

IAM-specific business metrics that should be added:

```go
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Authentication outcomes — critical for SLO tracking
	AuthSuccessTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ggid_auth_success_total",
			Help: "Total successful authentications",
		},
		[]string{"tenant_id", "method"}, // method: password, ldap, oauth, saml, webauthn
	)

	AuthFailureTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ggid_auth_failure_total",
			Help: "Total failed authentications",
		},
		[]string{"tenant_id", "method", "reason"}, // reason: invalid_password, locked, expired, mfa_failed
	)

	// Token lifecycle
	TokenIssuanceTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ggid_token_issuance_total",
			Help: "Total tokens issued",
		},
		[]string{"tenant_id", "type"}, // type: access, refresh, id
	)

	TokenValidationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ggid_token_validation_seconds",
			Help:    "JWT validation latency",
			Buckets: []float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.05, 0.1},
		},
		[]string{"tenant_id"},
	)

	// MFA
	MFAChallengeTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ggid_mfa_challenge_total",
			Help: "Total MFA challenges issued",
		},
		[]string{"tenant_id", "type"}, // type: totp, sms, webauthn
	)

	MFASuccessTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ggid_mfa_success_total",
			Help: "Total MFA challenges passed",
		},
		[]string{"tenant_id", "type"},
	)

	// Active sessions gauge
	ActiveSessions = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ggid_active_sessions",
			Help: "Number of active sessions",
		},
		[]string{"tenant_id"},
	)

	// Policy evaluation latency — important for ABAC performance
	PolicyEvalDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ggid_policy_eval_seconds",
			Help:    "Policy evaluation latency",
			Buckets: []float64{0.00001, 0.00005, 0.0001, 0.0005, 0.001, 0.005},
		},
		[]string{"tenant_id", "decision"}, // decision: allow, deny
	)

	// NATS publish latency — audit event delivery
	NATSPublishDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ggid_nats_publish_seconds",
			Help:    "NATS message publish latency",
			Buckets: []float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.05},
		},
		[]string{"subject"},
	)
)
```

### Usage in Auth Handler

```go
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromCtx(r.Context())

	result, err := h.authService.VerifyCredentials(r.Context(), req.Username, req.Password)
	if err != nil {
		metrics.AuthFailureTotal.WithLabelValues(tenantID, "password", err.Reason()).Inc()
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	metrics.AuthSuccessTotal.WithLabelValues(tenantID, "password").Inc()
	metrics.TokenIssuanceTotal.WithLabelValues(tenantID, "access").Inc()

	// ...
}
```

---

## 5. SLO/SLI Definitions for IAM

### Service Level Indicators (SLIs)

For an IAM system, the critical SLIs are:

| SLI | Target | Measurement Window | Why It Matters |
|-----|--------|-------------------|----------------|
| Auth success rate | 99.9% | 5m rolling | Users can authenticate |
| Login latency (p99) | < 500ms | 5m rolling | Login UX responsiveness |
| Token issuance latency (p99) | < 200ms | 5m rolling | API access after auth |
| Token validation latency (p99) | < 10ms | 5m rolling | Every API call validates JWT |
| MFA challenge delivery | < 5s | 5m rolling | MFA UX — SMS/push delivery |
| Audit event delivery | < 2s | 5m rolling | Forensic timeliness |

### Error Budget Calculation

For a 99.9% SLO over a 30-day month:

```
Total minutes: 30 * 24 * 60 = 43,200 minutes
Error budget: 0.1% = 43.2 minutes ≈ 43 minutes of allowable downtime per month
```

This means:
- At 1000 auth requests/min, you can fail 1 request/min on average
- A 5-minute outage consumes ~7% of the monthly budget
- Three 5-minute outages consume ~21% — still within budget

### SLO Metrics in Prometheus

```go
// SLO metrics — these enable error budget tracking and burn rate alerts

var (
	// SLO: auth success rate — numerator/denominator for ratio calculation
	AuthSLOSuccess = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ggid_slo_auth_success",
		Help: "Successful auth events for SLO calculation",
	})

	AuthSLOTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ggid_slo_auth_total",
		Help: "Total auth events for SLO calculation",
	})

	// SLO: login latency — histogram with SLO-aligned buckets
	LoginLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "ggid_slo_login_latency_seconds",
		Help:    "Login latency for SLO (target: p99 < 500ms)",
		Buckets: []float64{0.05, 0.1, 0.2, 0.3, 0.4, 0.5, 0.75, 1.0, 2.0, 5.0},
	})
)

// RecordAuthOutcome is called by the auth handler to record SLO metrics.
func RecordAuthOutcome(success bool, duration time.Duration) {
	AuthSLOTotal.Inc()
	if success {
		AuthSLOSuccess.Inc()
	}
	LoginLatency.Observe(duration.Seconds())
}
```

### Burn Rate Alerts

Burn rate tells you how fast you're consuming your error budget:

- **1x burn rate**: You'll exhaust the budget exactly at the end of the period
- **6x burn rate**: You'll exhaust the budget in 1/6th of the period (5 days for a 30-day SLO)
- **30x burn rate**: Budget exhausted in ~1 day

Multi-window multi-burn-rate alerting (recommended):

```yaml
# Fast burn: 14.4x over 1h + 5m windows — catches rapid degradation
# Slow burn: 6x over 6h + 30m windows — catches gradual degradation
# See alerting rules section for full Prometheus rules
```

---

## 6. Alerting Rules

### Critical IAM Alerts

```yaml
# /etc/prometheus/rules/iam-alerts.yml

groups:
  - name: iam-critical
    interval: 30s
    rules:

      # P0: Auth failure spike from single IP — potential brute force
      - alert: AuthFailureSpike
        expr: |
          sum(rate(ggid_auth_failure_total[1m])) by (ip_address) > 10
        for: 1m
        labels:
          severity: critical
          team: security
        annotations:
          summary: "Auth failure spike from {{ $labels.ip_address }}"
          description: "{{ $value }} auth failures/min from a single IP — possible brute force"

      # P0: SLO burn rate — 14.4x over 1h window
      - alert: AuthSLOBurnRateCritical
        expr: |
          (
            (1 - (sum(rate(ggid_slo_auth_success[5m])) / sum(rate(ggid_slo_auth_total[5m]))))
            >
            (1 - 0.999) * 14.4
          )
          and
          (
            (1 - (sum(rate(ggid_slo_auth_success[1h])) / sum(rate(ggid_slo_auth_total[1h]))))
            >
            (1 - 0.999) * 14.4
          )
        for: 2m
        labels:
          severity: critical
          team: sre
        annotations:
          summary: "Auth SLO burn rate critical (14.4x)"
          description: "Consuming error budget 14.4x faster than allowed"

      # P0: Tenant data leak — cross-tenant access detected
      - alert: TenantDataLeak
        expr: |
          sum(rate(ggid_cross_tenant_access_denied_total[5m])) by (source_tenant, target_tenant) > 0
        for: 1m
        labels:
          severity: critical
          team: security
        annotations:
          summary: "Cross-tenant access attempt: {{ $labels.source_tenant }} → {{ $labels.target_tenant }}"

      # P1: Certificate expiry — TLS or signing cert expiring soon
      - alert: CertExpiringSoon
        expr: |
          ggid_cert_expiry_timestamp - time() < 7 * 24 * 3600
        for: 1h
        labels:
          severity: warning
          team: platform
        annotations:
          summary: "Certificate {{ $labels.name }} expires in < 7 days"

      # P1: DB connection pool exhaustion
      - alert: DBPoolExhaustion
        expr: |
          pgxpool_idle_connections == 0
          and pgxpool_acquired_connections / pgxpool_max_connections > 0.9
        for: 2m
        labels:
          severity: warning
          team: platform
        annotations:
          summary: "DB connection pool {{ $labels.service }} nearly exhausted"

      # P1: NATS stream lag — audit events backing up
      - alert: NATSStreamLag
        expr: |
          nats_jetstream_stream_pending > 10000
        for: 5m
        labels:
          severity: warning
          team: platform
        annotations:
          summary: "NATS stream {{ $labels.stream }} lag: {{ $value }} pending messages"

      # P1: Login latency SLO violation
      - alert: LoginLatencyHigh
        expr: |
          histogram_quantile(0.99,
            sum(rate(ggid_slo_login_latency_seconds_bucket[5m])) by (le)
          ) > 0.5
        for: 5m
        labels:
          severity: warning
          team: sre
        annotations:
          summary: "Login p99 latency above 500ms SLO"
          description: "Current p99: {{ $value }}s"
```

### Alertmanager Routing

```yaml
# /etc/alertmanager/config.yml

route:
  receiver: default
  group_by: ['alertname', 'tenant_id']
  group_wait: 10s
  group_interval: 5m
  repeat_interval: 1h
  routes:
    # P0 — page on-call immediately
    - matchers: ['severity="critical"']
      receiver: pagerduty-oncall
      group_wait: 0s
      repeat_interval: 30m

    # P1 — notify Slack, no page
    - matchers: ['severity="warning"']
      receiver: slack-alerts
      repeat_interval: 2h

receivers:
  - name: pagerduty-oncall
    pagerduty_configs:
      - service_key: '${PAGERDUTY_KEY}'
        severity: critical
        description: '{{ .GroupLabels.alertname }}'

  - name: slack-alerts
    slack_configs:
      - api_url: '${SLACK_WEBHOOK}'
        channel: '#iam-alerts'
        title: '[{{ .Status }}] {{ .GroupLabels.alertname }}'
        text: '{{ range .Alerts }}{{ .Annotations.summary }}\n{{ end }}'

  - name: default
    slack_configs:
      - api_url: '${SLACK_WEBHOOK}'
        channel: '#iam-alerts'
```

---

## 7. Security Observability

### Audit Trail Correlation

A single user action in an IAM system generates a chain of events. Security
observability requires correlating these across services:

```
1. [auth]     event=auth.success     user=admin@acme.com  ip=10.0.0.5  trace_id=abc123
2. [gateway]  event=http.request     path=/api/v1/users   user=admin@acme.com  trace_id=abc123
3. [identity] event=user.read        user_id=<target>     actor=admin@acme.com  trace_id=abc123
4. [identity] event=user.update      user_id=<target>     actor=admin@acme.com  fields=[role_id]  trace_id=abc123
5. [audit]    event=audit.publish    subject=user.update  consumer_ack=true    trace_id=abc123
```

The `trace_id` ties all five events together, enabling reconstruction of the
full action chain for forensic investigation.

### Detecting Suspicious Patterns

#### Impossible Travel

```go
// Detects when the same user authenticates from two geographically
// impossible locations within a short timeframe.
func (d *Detector) CheckImpossibleTravel(ctx context.Context, userID, ip string) bool {
	currentLoc := d.geoIP.Lookup(ip)
	lastAuth := d.getLastAuthLocation(ctx, userID)

	if lastAuth == nil {
		return false
	}

	distance := haversineDistance(currentLoc, lastAuth.Location)
	timeDiff := time.Since(lastAuth.Timestamp)

	// If distance implies speed > 900 km/h (commercial flight), flag as suspicious
	maxSpeed := 900.0 // km/h
	requiredTime := time.Duration(distance/maxSpeed*3600) * time.Second

	if timeDiff < requiredTime {
		d.logger.Warn("impossible travel detected",
			"user_id", userID,
			"prev_location", lastAuth.Location,
			"curr_location", currentLoc,
			"distance_km", distance,
			"time_diff", timeDiff,
			"event_type", "security.impossible_travel",
		)
		d.metrics.SecurityAnomalyTotal.WithLabelValues("impossible_travel").Inc()
		return true
	}
	return false
}
```

#### Brute Force Detection

```go
// Detects repeated authentication failures from a single IP or
// against a single username across multiple IPs.
func (d *Detector) CheckBruteForce(ctx context.Context, ip, username string) bool {
	ipFailures := d.rateCounter.Count(ctx, "auth_fail:"+ip, time.Minute)
	userFailures := d.rateCounter.Count(ctx, "auth_fail_user:"+username, time.Minute)

	if ipFailures > 10 || userFailures > 5 {
		d.logger.Warn("brute force detected",
			"ip_address", ip,
			"username", username,
			"ip_failures_1m", ipFailures,
			"user_failures_1m", userFailures,
			"event_type", "security.brute_force",
		)
		d.metrics.SecurityAnomalyTotal.WithLabelValues("brute_force").Inc()
		return true
	}
	return false
}
```

#### Privilege Escalation

```go
// Detects when a user's role changes to a higher privilege level
// without a corresponding approval workflow.
func (d *Detector) CheckPrivilegeEscalation(ctx context.Context, userID string, newRole Role) error {
	oldRole := d.getUserRole(ctx, userID)

	if oldRole == nil {
		// New role assignment — check if it's a privileged role
		if newRole.IsAdmin() {
			d.logger.Warn("admin role assigned directly",
				"user_id", userID,
				"role", newRole.Name,
				"event_type", "security.privilege_escalation",
			)
			return ErrUnprivilegedRoleChange
		}
		return nil
	}

	if newRole.PrivilegeLevel() > oldRole.PrivilegeLevel() {
		d.logger.Warn("privilege escalation: role upgrade",
			"user_id", userID,
			"old_role", oldRole.Name,
			"new_role", newRole.Name,
			"event_type", "security.privilege_escalation",
		)
		d.metrics.SecurityAnomalyTotal.WithLabelValues("privilege_escalation").Inc()
	}
	return nil
}
```

### SIEM Integration via NATS

GGID's audit system already publishes events to NATS JetStream. For SIEM
integration, the pipeline is:

```
[auth/gateway/policy] → NATS audit.events → [Logstash/Fluentd consumer] → [Elasticsearch/Splunk]
                                                                      → [SIEM correlation engine]
```

```go
// SecurityEventEmitter publishes structured security events to NATS
// for downstream SIEM correlation.
type SecurityEventEmitter struct {
	js      nats.JetStreamContext
	logger  *slog.Logger
	metrics *Metrics
}

type SecurityEvent struct {
	EventType    string                 `json:"event_type"`     // security.auth_failure, security.privilege_change
	Timestamp    time.Time              `json:"timestamp"`
	TraceID      string                 `json:"trace_id"`
	TenantID     string                 `json:"tenant_id"`
	UserID       string                 `json:"user_id"`
	IPAddress    string                 `json:"ip_address"`
	UserAgent    string                 `json:"user_agent"`
	Resource     string                 `json:"resource"`       // e.g., "user:uuid", "role:admin"
	Action       string                 `json:"action"`         // read, write, delete, auth
	Result       string                 `json:"result"`         // success, failure, denied
	Details      map[string]interface{} `json:"details,omitempty"`
}

func (e *SecurityEventEmitter) Emit(ctx context.Context, event SecurityEvent) error {
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.IsValid() {
		event.TraceID = spanCtx.TraceID().String()
	}

	data, _ := json.Marshal(event)
	subject := fmt.Sprintf("audit.events.%s.%s", event.TenantID, event.EventType)

	_, err := e.js.Publish(subject, data,
		nats.Context(ctx),
	)
	if err != nil {
		e.logger.Error("failed to publish security event",
			"event_type", event.EventType,
			"error", err,
		)
		e.metrics.SecurityEventPublishFailures.Inc()
		return err
	}

	e.metrics.SecurityEventPublished.WithLabelValues(event.EventType).Inc()
	return nil
}
```

---

## 8. GGID Observability Gap Analysis

### What Exists

| Capability | Location | Status | Notes |
|-----------|----------|--------|-------|
| **Prometheus metrics** | `gateway/internal/middleware/metrics.go` | Partial | RED metrics (rate, errors, duration) defined; runtime metrics (goroutines, memory, GC) in `metrics_enhanced.go` |
| **Metrics endpoint** | `gateway/internal/router/router.go:181` | Working | `/metrics` registered in gateway router |
| **Custom tracing** | `gateway/internal/middleware/otel.go` | Partial | Custom W3C traceparent parsing, OTLP/HTTP export, sampling, child spans — NOT using official OTel SDK |
| **Request logging** | `gateway/internal/middleware/middleware.go:73` | Basic | `log.Printf` with method/path/status/size/duration/requestID — NOT structured JSON, no trace_id/tenant_id/user_id |
| **Response time tracking** | `gateway/internal/middleware/response_time.go` | Working | Prometheus histogram + X-Response-Time header |
| **Health scoring** | `gateway/internal/middleware/health_score.go` | Working | Backend health score for load balancing |
| **Auth failure counter** | `gateway/internal/middleware/metrics.go:30` | Basic | `auth_failures_total` by reason — no tenant_id dimension |
| **Circuit breaker** | `gateway/internal/middleware/circuitbreaker.go` | Working | Has metrics integration |
| **Audit events** | `pkg/audit/`, `services/audit/` | Working | NATS JetStream publish + REST query |
| **Anomaly detection** | `services/auth/internal/service/anomaly_detection.go` | Partial | Impossible travel, brute force detection exists in auth service |

### What's Missing

| Gap | Severity | Impact |
|-----|----------|--------|
| **No structured logging (slog)** | HIGH | `log.Printf` produces unparseable output; no trace_id, tenant_id, or user_id correlation in logs |
| **Metrics only in gateway** | HIGH | Auth, identity, oauth, policy, org, audit services have zero Prometheus metrics or `/metrics` endpoints |
| **No official OTel SDK** | MEDIUM | Custom tracing implementation lacks auto-instrumentation, gRPC instrumentation, baggage propagation |
| **No custom IAM metrics** | HIGH | Missing auth_success_total, token_issuance_total, mfa_challenge_total, policy_eval_duration, NATS publish latency |
| **No tenant_id dimension on metrics** | HIGH | Cannot filter metrics by tenant — critical for multi-tenant SLO tracking |
| **No SLO/SLI definitions** | HIGH | No error budget tracking, no burn rate alerts |
| **No alerting rules** | HIGH | No Prometheus alert rules, no Alertmanager configuration |
| **No secret redaction** | HIGH | `log.Printf` could log passwords, tokens, private keys |
| **No SIEM integration pipeline** | MEDIUM | Audit events go to NATS but no documented consumer → SIEM pipeline |
| **Tracing not propagated to backend services** | MEDIUM | Only gateway has tracing; auth/identity/policy services have no span creation |
| **No gRPC tracing/metrics** | MEDIUM | gRPC interceptor logs method/duration/code but doesn't create spans or record metrics |
| **No log aggregation config** | MEDIUM | No Fluentd/Fluent Bit/Logstash config in docker-compose |

### Key Codebase Findings

**Request logging** (`middleware/middleware.go:79`):
```go
// Current: unstructured, no correlation fields
log.Printf("%s %s %d %d %s req=%s", r.Method, r.URL.Path, sr.status, sr.size, ...)
```
Missing: `trace_id`, `tenant_id`, `user_id`, `event_type`, `ip_address`, `user_agent`.

**Auth failure metrics** (`middleware/metrics.go:30`):
```go
// Current: no tenant_id dimension
authFailures = CounterVec{labels: ["reason"]}
```
Cannot determine which tenant is experiencing auth failures.

**Tracing** (`middleware/otel.go`):
Uses custom `traceContextKey{}` and manual span management. Does not integrate
with OTel API, so `trace.SpanContextFromContext()` returns empty in downstream
code — trace_id cannot be extracted in log handlers.

**Metrics not registered in non-gateway services**: A grep for `prometheus` in
`services/auth/`, `services/identity/`, `services/policy/`, `services/org/`,
`services/audit/`, `services/oauth/` returns zero matches.

---

## 9. Gap Analysis & Recommendations

### Priority Action Items

#### Action 1: Adopt slog with IAM Fields and Secret Redaction
**Effort**: 2-3 days | **Priority**: P0

Replace all `log.Printf` calls across services with `slog` structured logging.
Create a shared `pkg/observability/logging.go` package with:
- JSON handler with trace_id/span_id injection from OTel context
- Secret field redaction (password, token, secret, api_key, etc.)
- Standard IAM fields (tenant_id, user_id, event_type, ip_address)

This is the foundation — without structured logs, forensic investigation and
SIEM correlation are impossible.

#### Action 2: Migrate to Official OpenTelemetry SDK
**Effort**: 3-5 days | **Priority**: P1

Replace the custom tracing implementation in `middleware/otel.go` with the
official `go.opentelemetry.io/otel` SDK:
- Add OTel SDK as a dependency (`go get go.opentelemetry.io/otel@latest`)
- Implement tracer provider init in each service's `cmd/main.go`
- Add HTTP auto-instrumentation (`otelhttp`) for all services
- Add gRPC auto-instrumentation (`otelgrpc`) for gateway-to-backend calls
- Propagate trace context via W3C TraceContext automatically
- Add DB span instrumentation via OTel SQL driver wrapper

This enables true distributed tracing across all 7 microservices.

#### Action 3: Deploy Metrics in All Services with tenant_id Dimension
**Effort**: 3-4 days | **Priority**: P0

Add Prometheus metrics to auth, identity, oauth, policy, org, and audit services:
- Create `pkg/observability/metrics.go` with shared metric definitions
- Add IAM-specific metrics: auth_success_total, auth_failure_total,
  token_issuance_total, mfa_challenge_total, policy_eval_duration
- Add tenant_id label to all request-scoped metrics
- Register `/metrics` endpoint in each service's HTTP server
- Ensure the metrics endpoint is excluded from auth middleware

#### Action 4: Define SLOs and Alert Rules
**Effort**: 2-3 days | **Priority**: P1

- Define Prometheus recording rules for SLO calculations (auth success rate,
  login p99 latency, token issuance latency)
- Create multi-window burn rate alerts (14.4x/1h for P0, 6x/6h for P1)
- Create security alerts: auth failure spike, cross-tenant access, cert expiry
- Configure Alertmanager routing (PagerDuty for critical, Slack for warning)
- Add SLO dashboard to Grafana (error budget remaining, burn rate, latency p99)

#### Action 5: SIEM Integration Pipeline
**Effort**: 3-5 days | **Priority**: P2

- Create a NATS consumer that subscribes to `audit.events.>` and forwards to
  Logstash/Fluentd in a structured format (ECS — Elastic Common Schema)
- Add the security event emitter pattern from Section 7
- Configure anomaly detection rules in the SIEM (impossible travel, brute force,
  privilege escalation)
- Document the audit trail correlation flow for security analysts
- Add a `docker-compose` observability stack (Prometheus, Grafana, Jaeger,
  Loki, Alertmanager) for local development

### Summary

GGID has a solid foundation — Prometheus metrics in the gateway, custom W3C
tracing, and NATS-based audit events. However, the observability is **gateway-only**
and **unstructured**. The five actions above would bring all 7 microservices to
a state where:
- Every request has a trace ID visible in logs, traces, and metrics exemplars
- Every metric can be filtered by tenant
- Security analysts can reconstruct any user action chain via trace_id
- SRE teams have SLO-based alerting with error budget tracking
- SIEM integration provides automated anomaly detection
