# Auth0 "First 5 Minutes" vs GGID Quickstart Experience

> A focused, step-by-step comparison of the developer onboarding experience:
> how long it takes to go from zero to a working authenticated app in Auth0
> versus GGID. Includes friction analysis, an improvement roadmap, and a sample
> app gallery plan.
>
> Last updated: 2025-07-11
> GGID quickstart docs verified against source at time of writing.

---

## Table of Contents

1. [Auth0 Quickstart Experience](#1-auth0-quickstart-experience)
2. [GGID Quickstart Experience](#2-ggid-quickstart-experience)
3. [Step-by-Step Comparison](#3-step-by-step-comparison)
4. [What Auth0 Does Better](#4-what-auth0-does-better)
5. [What GGID Does Better](#5-what-ggid-does-better)
6. [Quickstart Improvement Plan](#6-quickstart-improvement-plan)
7. [Sample App Gallery](#7-sample-app-gallery)
8. [Existing Quickstart Doc Review](#8-existing-quickstart-doc-review)
9. [Conclusion](#9-conclusion)

---

## 1. Auth0 Quickstart Experience

Auth0's onboarding is widely considered the gold standard in the IAM space.
The marketing promise is "add authentication to your app in 5 minutes," and
they come remarkably close. Here is every step a new developer takes.

### Step-by-Step Walkthrough

| Step | Action | What Happens |
|------|--------|--------------|
| 1 | Sign up at auth0.com | Email/password or social login (Google, GitHub, Microsoft) |
| 2 | Verify email | Click confirmation link |
| 3 | Choose region | US, EU, AU, or Japan — determines data residency |
| 4 | Tenant created | A subdomain is auto-provisioned: `your-tenant.us.auth0.com` |
| 5 | Create application | Dashboard wizard asks: name, app type (Regular Web App, SPA, Native, M2M) |
| 6 | Choose technology | Pick from 30+ framework quickstarts (React, Next.js, Express, Spring Boot, etc.) |
| 7 | Download sample app | Pre-configured sample with `.env` file, ready to `npm install && npm start` |
| 8 | Copy Domain + Client ID | Displayed on the quickstart page, one click to copy |
| 9 | Configure callback URLs | Set `http://localhost:3000/callback` in dashboard |
| 10 | Install SDK | `npm install @auth0/auth0-react` — shown in the quickstart |
| 11 | Wrap app with Auth0Provider | 4 lines of JSX in `index.js` |
| 12 | Add login button | `<LoginButton />` component — 1 line of JSX |
| 13 | First login | Click button → Auth0 hosted login page → redirect back with tokens |
| 14 | Access user profile | `const { user } = useAuth0()` — user object available |

### What Makes Auth0 Feel Easy

**1. Hosted Login Page (Universal Login)**
The developer never writes a login form. Auth0 hosts a fully styled, customizable
login page at `your-tenant.auth0.com`. The SDK redirects there, handles all the
OAuth/OIDC complexity, and redirects back with tokens. Zero UI code required.

**2. Copy-Paste Snippets**
Every quickstart page shows exact code for the selected framework, with the
domain and client ID pre-filled. The developer copies, pastes, and it works.

**3. Dashboard-Driven Configuration**
Everything is configured through a visual dashboard — callback URLs, social
connections, branding, email templates. No YAML, no environment file editing
for basic setup.

**4. Pre-Built Sample Apps**
For every framework, Auth0 provides a downloadable sample app that is already
configured. The developer runs `npm install && npm start`, and the app works.
This eliminates the "blank page" problem.

**5. Progressive Disclosure**
The quickstart starts with "3 lines to login" and progressively reveals more
complex features (RBAC, rules, actions, enterprise connections) only when needed.

### Total Step Count: 14 steps
### Estimated Time: 5-8 minutes (for SPA with React SDK)

### Where Auth0 Adds Hidden Friction

- **Account signup required**: You must create an Auth0 account before touching
  any code. This is a trust barrier for self-hosted advocates.
- **Vendor lock-in**: Everything lives in Auth0's cloud. Migrating away means
  rewriting integration points.
- **Pricing surprise**: Free tier is generous (7,000 MAU) but enterprise features
  (SAML, custom domains, logs) require paid plans starting at $35/month.
- **No local development**: You cannot run Auth0 locally. All development hits
  their hosted infrastructure.
- **Opaque internals**: You cannot inspect or debug the token issuance pipeline.

---

## 2. GGID Quickstart Experience

GGID's quickstart takes a fundamentally different approach: self-hosted,
Go-native, full source access. Here is the step-by-step walkthrough based on
the actual quickstart documentation at `docs/quickstart/`.

### Step-by-Step Walkthrough (5-Minute JWT Path)

| Step | Action | What Happens |
|------|--------|--------------|
| 1 | Clone repository | `git clone https://github.com/ggid/ggid && cd ggid` |
| 2 | Start Docker Compose | `docker compose -f deploy/docker-compose.yaml up -d` |
| 3 | Wait for healthchecks | `sleep 30` — all 12 containers must be healthy |
| 4 | Know the default tenant | `X-Tenant-ID: 00000000-0000-0000-0000-000000000001` (documented) |
| 5 | Register a user | `curl POST /api/v1/auth/register` with JSON body and X-Tenant-ID header |
| 6 | Login to get JWT | `curl POST /api/v1/auth/login` → `access_token` in response |
| 7 | Use JWT on protected API | `curl GET /api/v1/users -H "Authorization: Bearer $JWT"` |

### Step-by-Step Walkthrough (SDK Integration Path)

| Step | Action | What Happens |
|------|--------|--------------|
| 1 | Start GGID (Docker) | `docker compose -f deploy/docker-compose.yaml up -d` |
| 2 | Get JWT secret | Read from `.env` or `deploy/.env.example` — must match server config |
| 3 | Install SDK | `go get github.com/ggid/ggid/sdk/go@latest` |
| 4 | Create verifier | `ggid.NewVerifier("http://localhost:8080", "secret")` |
| 5 | Wrap handler | `middleware.Protect(myHandler)` |
| 6 | Access claims | `claims := ggid.ClaimsFromContext(r.Context())` |

### What Makes GGID's Quickstart Work

**1. Docker Compose One-Command Start**
A single `docker compose up -d` brings up all 12 containers (7 microservices +
PostgreSQL + Redis + NATS + LDAP + Console). No account signup, no cloud
provisioning, no credit card. The entire IAM stack runs locally.

**2. Copy-Paste curl Snippets**
The 5-minute JWT quickstart provides ready-to-run curl commands for register,
login, and API calls. No SDK install required for initial evaluation.

**3. Multi-Language Examples**
The same 3-step flow (register → login → use JWT) is shown in curl, Go, and
Node.js in a single page. Developers pick whichever language they know.

**4. SDK Quickstarts for Go and Node**
Dedicated quickstart pages for Go SDK and Node SDK show "3-line" JWT
verification and middleware setup. Gin and Express integration guides provide
framework-specific patterns.

### Friction Points in GGID's Quickstart

| # | Friction Point | Impact | Details |
|---|----------------|--------|---------|
| F1 | **No hosted login page** | High | Developers must build their own login UI. Auth0 provides this out of the box. GGID has no equivalent — the quickstart uses raw curl, not a browser-based login flow. |
| F2 | **Tenant ID is a manual header** | Medium | Every request must include `X-Tenant-ID: 00000000-0000-0000-0000-000000000001`. This is friction for first-time users who don't understand multi-tenancy yet. Auth0 has no equivalent concept in the quickstart. |
| F3 | **JWT secret discovery** | Medium | The SDK needs the JWT secret, but it's not obvious where to find it. Developer must read `.env.example` or inspect Docker Compose environment variables. Auth0 handles key management invisibly. |
| F4 | **No sample app download** | High | There is no downloadable, pre-configured sample app. Developers must start from scratch or copy code from docs into their own project. Auth0 provides a working sample for every framework. |
| F5 | **30-second startup wait** | Low | `sleep 30` for healthchecks is slower than Auth0's instant dashboard. Acceptable but noticeable. |
| F6 | **No "create application" wizard** | Medium | Auth0's wizard guides the developer through app creation and auto-selects the right quickstart. GGID has no equivalent — the developer must know to look at `docs/quickstart/`. |
| F7 | **No OAuth client auto-creation** | Medium | For OAuth flows, the developer must manually register an OAuth client via curl (see oauth-login.md). Auth0 auto-creates a default application during signup. |
| F8 | **Docker prerequisite** | Low | Requires Docker Desktop installed and running. Not all developers have this. Auth0 requires only a browser. |
| F9 | **No browser-based login demo** | High | There is no URL the developer can visit to see a login page in action. The Console exists but isn't positioned as a quickstart demo. The entire quickstart is API-only (curl). |
| F10 | **Fragmented documentation entry point** | Low | The `docs/quickstart/` directory has 5 files. A new developer must choose which one to start with. A single "Start Here" landing page would help. |

### Total Step Count: 7 steps (API path), 6 steps (SDK path)
### Estimated Time: 8-12 minutes (including Docker startup)

---

## 3. Step-by-Step Comparison

### Side-by-Side: First Authenticated Request

| Phase | Auth0 | GGID |
|-------|-------|------|
| **Prerequisite** | Auth0 account (email verify) | Docker Desktop installed |
| **Setup** | Auto-provisioned tenant subdomain | `docker compose up -d` + 30s wait |
| **Application config** | Dashboard wizard (app type, framework) | Know default tenant UUID |
| **Get credentials** | Copy Domain + Client ID from dashboard | Find JWT secret in .env |
| **Install SDK** | `npm install @auth0/auth0-react` | `go get .../sdk/go` or `npm install @ggid/sdk-node` |
| **Integration** | Wrap app + add `<LoginButton />` | Create verifier + wrap handler |
| **Login UI** | Hosted login page (zero code) | Build your own or use curl |
| **First authenticated call** | `useAuth0()` hook gives user object | `curl -H "Authorization: Bearer $JWT"` |

### Detailed Step Count Comparison

| Metric | Auth0 | GGID | Winner |
|--------|-------|------|--------|
| Total steps to first login | 14 | 7 (API) / 6 (SDK) | GGID (fewer steps) |
| Steps requiring UI/dashboard interaction | 5 | 0 | GGID (fully scriptable) |
| Steps requiring code writing | 4 | 3 | GGID (slightly fewer) |
| Steps requiring external service (network) | 3 | 0 (after Docker pull) | GGID (offline-capable) |
| Time to first authenticated request | 5-8 min | 8-12 min | Auth0 (faster startup) |
| Time to production-ready login UI | 5 min (hosted) | 2+ hours (build your own) | Auth0 |
| Time to OAuth flow working | 10 min (wizard) | 15-20 min (manual client reg) | Auth0 |
| Offline development | No | Yes | GGID |
| Data sovereignty | No (cloud-hosted) | Yes (self-hosted) | GGID |

### Where Each Wins

**Auth0 is simpler when:**
- You need a login UI immediately (hosted login page)
- You want zero infrastructure setup
- You're building an SPA and want drop-in components
- You want visual configuration (dashboard)
- You want sample apps per framework

**GGID is simpler when:**
- You want to evaluate locally without account signup
- You need full source code access
- You're building a Go backend (native SDK, no Node.js)
- You need data sovereignty / self-hosting
- You want scriptable, reproducible setup (Docker Compose)

---

## 4. What Auth0 Does Better

### 4.1 Hosted Login Page

Auth0's Universal Login is their single biggest UX advantage. The developer
never writes a login form, never handles credential storage, never styles a
forgot-password flow. Auth0 hosts all of it at `tenant.auth0.com`.

**Impact on quickstart:** This is why Auth0 can promise "5 minutes." The
developer's integration is literally a redirect + callback. GGID requires the
developer to build (or find) a login UI, which is hours of work.

**Gap size:** Large. GGID would need either:
- A hosted login page service (controversial for a self-hosted project), or
- A drop-in React/Vue login component that calls the GGID API, or
- A Console-based login flow that can serve as the "hosted page"

### 4.2 Universal Login (One Config for All Apps)

Auth0's Universal Login means one login page configuration serves every
application connected to the tenant. Change branding once, all apps update.
New applications inherit the configuration automatically.

**Impact on quickstart:** When a developer creates a new application in Auth0,
the login experience is already configured. In GGID, each application needs
its own login UI (or a shared component the developer must build).

### 4.3 Pre-Built SDK with 3-Line Integration

Auth0's React SDK (`@auth0/auth0-react`) provides:
```jsx
// 1. Wrap app
<Auth0Provider domain="..." clientId="..." redirectUri={window.location.origin}>
  <App />
</Auth0Provider>

// 2. Login button
<LoginButton />Log In</LoginButton>

// 3. Access user
const { user, isAuthenticated } = useAuth0();
```

Three lines. GGID's SDK quickstart is also 3 lines for JWT verification, but
it doesn't include a login button or user context — the developer still needs
to handle the authentication flow themselves.

### 4.4 Dashboard for Configuration

Auth0's dashboard provides:
- Application management (create, edit, delete)
- Connection management (social, enterprise, database)
- User management (search, view, block, delete)
- Branding editor (logo, colors, custom domain)
- Log viewer (real-time authentication events)
- Rules/Actions editor (custom logic on auth events)

GGID has an Admin Console (Next.js) but it is positioned as an admin tool, not
a developer onboarding tool. It doesn't guide new developers through setup.

### 4.5 Sample App Download

For every framework, Auth0 provides a downloadable, pre-configured sample app:
- The `.env` file is pre-filled with the developer's domain and client ID
- The app runs immediately after `npm install`
- The code demonstrates best practices (protected routes, token refresh, logout)

GGID has no equivalent. The quickstart docs show code snippets, but there is
no `git clone`-able sample app that works out of the box.

### 4.6 Progressive Quickstart Paths

Auth0 detects the developer's framework and shows a tailored quickstart. A
React developer sees React code; a Spring Boot developer sees Java code. The
quickstart adapts to the developer, not the other way around.

GGID's quickstart directory requires the developer to choose the right file.
There is no framework detection or adaptive guidance.

---

## 5. What GGID Does Better

### 5.1 Self-Hosted (Data Never Leaves Your Infra)

This is GGID's fundamental differentiator. Every byte of authentication data —
user credentials, tokens, audit logs — stays in the developer's infrastructure.
No data flows to any third-party cloud.

**Why this matters:**
- GDPR/CCPA compliance is simpler (no data processor agreement needed)
- Industries with data residency requirements (healthcare, finance, government)
can use GGID where Auth0 is prohibited
- No vendor data breach risk (Auth0/Okta has had breaches)

**Quickstart impact:** The developer can evaluate GGID entirely locally. No
data is sent anywhere. This is a trust advantage that Auth0 cannot match.

### 5.2 Go-Native (No Node.js Dependency)

GGID is written in Go. The Go SDK is a first-class citizen, not a port. For
Go-based organizations, this means:
- No Node.js runtime required for the IAM layer
- Native performance (no V8 overhead)
- Compile-time type safety in the SDK
- Direct integration with Go middleware ecosystems (Gin, Echo, Chi, net/http)

Auth0's SDKs are JavaScript-first. The Go ecosystem is an afterthought.

### 5.3 Docker Compose One-Command Start

```bash
docker compose -f deploy/docker-compose.yaml up -d
```

One command. Twelve containers. Full IAM stack: gateway, identity, auth, oauth,
policy, org, audit, plus PostgreSQL, Redis, NATS, LDAP, and the admin console.

Auth0 cannot match this because it is a hosted service. The closest equivalent
is Auth0's "try in your browser" sandbox, which doesn't give you a real
instance.

### 5.4 Full Source Code Access

Every line of GGID is open source (Apache 2.0). Developers can:
- Read the token issuance pipeline to understand exactly what happens
- Debug issues by adding logging to the actual auth service
- Customize behavior by forking (self-hosted advantage)
- Audit the security implementation (no black box)

Auth0 is a black box. Developers cannot inspect the token issuance code, cannot
add custom logging to the core pipeline, and must trust Auth0's security claims.

### 5.5 No Account Signup Needed

GGID requires no email verification, no account creation, no credit card.
`git clone && docker compose up` and you have a working IAM system.

Auth0 requires:
1. Account creation
2. Email verification
3. Region selection
4. Acceptance of terms of service

This is 3-4 steps that GGID eliminates entirely.

### 5.6 Reproducible, Scriptable Setup

GGID's Docker Compose setup is fully reproducible. The same command produces
the same environment every time. This enables:
- CI/CD integration (spin up GGID in GitHub Actions for integration tests)
- Development environment parity (everyone on the team has identical setup)
- Automated testing (E2E tests can provision GGID in seconds)

Auth0's hosted model makes automated, isolated test environments difficult.

---

## 6. Quickstart Improvement Plan

For each friction point identified in Section 2, here is the fix, effort
estimate, and priority. Target: match Auth0's effective step count (7 steps
to first authenticated request) while keeping GGID's self-hosted advantage.

### Priority Matrix

| ID | Friction Point | Fix | Effort | Priority | Sprint |
|----|----------------|-----|--------|----------|--------|
| F9 | No browser-based login demo | Create a demo login page at `/demo/login` in the Console that calls `/api/v1/auth/login` and displays the JWT | M (3-5 days) | P0 | 1 |
| F4 | No sample app download | Create 3 sample apps (React, Go net/http, Express) in `/examples/` directory, each with README and Dockerfile | L (1-2 weeks) | P0 | 1-2 |
| F1 | No hosted login page | Build a drop-in React `<GGIDLogin />` component published as `@ggid/react-login` that handles login form + token storage | L (1-2 weeks) | P1 | 2 |
| F2 | Tenant ID is manual header | Auto-detect tenant from JWT or make default tenant implicit for single-tenant setups | M (3-5 days) | P1 | 2 |
| F3 | JWT secret discovery | Add a `ggid info` CLI command that prints the current JWT secret and gateway URL from `.env` | S (1-2 days) | P1 | 1 |
| F7 | No OAuth client auto-creation | Seed a default OAuth client (`demo-app`) during Docker Compose startup | S (1-2 days) | P2 | 2 |
| F6 | No "create application" wizard | Add an onboarding wizard to the Console that guides new developers through setup | M (5-7 days) | P2 | 3 |
| F10 | Fragmented entry point | Create a single `docs/quickstart/README.md` landing page that routes to the right quickstart based on developer type | S (1 day) | P1 | 1 |
| F5 | 30-second startup wait | Add a `ggid wait` command that polls healthchecks and exits when ready (replaces `sleep 30`) | S (1 day) | P2 | 2 |
| F8 | Docker prerequisite | Document Docker Desktop installation in prerequisites; provide a `ggid doctor` command that checks prerequisites | S (1 day) | P2 | 1 |

### Detailed Improvement Roadmap

#### Sprint 1 (Month 1): Eliminate P0 Friction

**1.1 Demo Login Page (F9)**
Add a public route `/demo/login` to the Console that:
- Shows a styled login form (email/password)
- Calls `POST /api/v1/auth/login` with the default tenant
- Displays the returned JWT and decoded claims
- Includes a "Try Protected API" button that calls `/api/v1/users`

This gives developers a URL to visit immediately after `docker compose up`.
No more "the quickstart is only curl" — they can see authentication in the
browser.

**1.2 Sample App Gallery — Phase 1 (F4)**
Create `/examples/` directory with:
- `examples/react-app/` — Create React App with GGID login + protected route
- `examples/go-http/` — Go net/http server with GGID middleware
- `examples/express-app/` — Express server with GGID middleware

Each example includes:
- Complete source code (not snippets)
- `README.md` with one-command run instructions
- `.env.example` with all required variables
- `Dockerfile` for containerized testing

**1.3 Quickstart Landing Page (F10)**
Create `docs/quickstart/README.md` that:
- Asks: "What are you building?" (SPA, backend API, mobile, microservices)
- Routes to the right quickstart based on answer
- Provides a "I just want to try it" path → demo login page

**1.4 GGID CLI Doctor Command (F3, F8)**
Create a `ggid` CLI tool with:
```bash
ggid doctor    # Check Docker, ports, health
ggid info      # Print gateway URL, JWT secret, default tenant
ggid wait      # Wait for all services to be healthy
```

#### Sprint 2 (Month 2): Close the UX Gap

**2.1 Drop-in Login Component (F1)**
Publish `@ggid/react-login` npm package:
```jsx
import { GGIDLogin, useGGID } from '@ggid/react-login';

function App() {
  const { user, login, logout } = useGGID({
    gatewayURL: 'http://localhost:8080',
    tenantID: 'default',
  });
  
  if (!user) return <GGIDLogin />;
  return <Dashboard user={user} />;
}
```

This is GGID's answer to Auth0's `<LoginButton />`. Same developer experience,
self-hosted.

**2.2 Default Tenant Simplification (F2)**
Make the default tenant implicit for single-tenant setups:
- If no `X-Tenant-ID` header is provided, use the default tenant
- Document this as "single-tenant mode"
- Multi-tenant users explicitly set the header

**2.3 Seed OAuth Client (F7)**
During Docker Compose startup, seed a default OAuth client:
```json
{
  "client_id": "demo-app",
  "client_secret": "demo-secret",
  "redirect_uris": ["http://localhost:3000/callback"],
  "grant_types": ["authorization_code"]
}
```

This eliminates the manual `curl` step in the OAuth quickstart.

#### Sprint 3 (Month 3): Polish and Parity

**3.1 Onboarding Wizard (F6)**
Add a first-run wizard to the Console:
1. Welcome screen
2. Create admin user
3. Create first application (generates client_id/secret)
4. Choose framework → show integration snippet
5. "Test your setup" button

**3.2 Additional Sample Apps (F4 expansion)**
Add:
- `examples/nextjs-app/` — Next.js with GGID SSR authentication
- `examples/spring-boot/` — Spring Boot with GGID JWT filter

**3.3 GGID Wait Command (F5)**
Replace `sleep 30` in all docs with:
```bash
ggid wait  # polls /healthz on all services, exits when all healthy
```

### Target Metrics After 3 Months

| Metric | Current | Target | Auth0 Benchmark |
|--------|---------|--------|-----------------|
| Steps to first authenticated request | 7 | 5 | 5 |
| Steps to working login UI | N/A (build your own) | 3 (install component + 2 lines) | 3 |
| Time to first authenticated request | 8-12 min | 5-7 min | 5-8 min |
| Sample apps available | 0 | 5 | 30+ |
| Browser-based demo | No | Yes | Yes |

---

## 7. Sample App Gallery

Auth0's sample app library is one of their strongest assets. Here is GGID's
priority plan for building an equivalent.

### Priority 1: Immediate (Sprint 1)

#### React SPA Sample
```
examples/react-app/
├── src/
│   ├── App.tsx          # GGIDLogin + protected dashboard
│   ├── api.ts           # Fetch wrapper with JWT injection
│   └── main.tsx
├── .env.example         # VITE_GGID_URL, VITE_TENANT_ID
├── Dockerfile
└── README.md            # One-command start: npm install && npm run dev
```

**Why first:** React is the most popular SPA framework. A working React sample
immediately demonstrates GGID's value to frontend developers. It also serves
as the foundation for the drop-in login component (improvement F1).

**Key features to demonstrate:**
- Login form calling `/api/v1/auth/login`
- JWT storage (localStorage or httpOnly cookie)
- Protected route with JWT verification
- Token refresh
- Logout

#### Go net/http Sample
```
examples/go-http/
├── main.go              # HTTP server with GGID middleware
├── go.mod
├── .env.example
├── Dockerfile
└── README.md
```

**Why first:** Go is GGID's native language. A Go sample showcases the
first-class SDK experience and is the fastest path to a working integration
for Go developers.

**Key features to demonstrate:**
- JWT verification middleware
- Claims extraction from context
- Scope-based authorization
- Tenant-aware data queries

#### Express.js Sample
```
examples/express-app/
├── src/
│   ├── app.ts           # Express with GGID middleware
│   ├── routes/
│   └── middleware/
├── .env.example
├── Dockerfile
└── README.md
```

**Why first:** Express is the most popular Node.js framework. Combined with
the Node SDK, this sample demonstrates GGID's cross-language capability.

### Priority 2: Near-Term (Sprint 2-3)

#### Next.js Sample
```
examples/nextjs-app/
├── app/
│   ├── layout.tsx       # GGIDProvider
│   ├── login/page.tsx   # Login page
│   ├── dashboard/       # Protected SSR page
│   └── api/             # API routes with GGID middleware
├── middleware.ts        # Edge middleware for JWT verification
├── .env.example
└── README.md
```

**Why Priority 2:** Next.js is the fastest-growing React framework. SSR
authentication is a common need. This sample demonstrates server-side JWT
verification and edge middleware patterns.

**Key features:**
- SSR authentication (JWT verified on server)
- Edge runtime middleware
- App Router integration
- API route protection

#### Spring Boot Sample
```
examples/spring-boot/
├── src/main/java/dev/ggid/demo/
│   ├── DemoApplication.java
│   ├── SecurityConfig.java   # GGID JWT filter
│   └── UserController.java
├── src/main/resources/application.yml
├── pom.xml
└── README.md
```

**Why Priority 2:** Enterprise Java is a key market for IAM. A Spring Boot
sample demonstrates the Java SDK and Spring Security integration. The
integration guide already exists; a full sample app is the natural next step.

### Priority 3: Future

| Sample App | Framework | Why | Timeline |
|------------|-----------|-----|----------|
| Vue SPA | Vue 3 + Vite | Second most popular SPA framework | Month 4 |
| Go Gin | Gin framework | Popular Go web framework, SDK already supports it | Month 4 |
| Python FastAPI | FastAPI | Python is #1 language; FastAPI is fastest-growing Python framework | Month 5 |
| React Native | Expo / React Native | Mobile authentication demo | Month 5 |
| .NET ASP.NET | ASP.NET Core | Enterprise .NET market | Month 6 |
| Django | Django | Python enterprise framework | Month 6 |
| Flutter | Flutter | Cross-platform mobile | Month 6 |

### Sample App Quality Standards

Every sample app must meet these standards (modeled on Auth0's):
1. **One-command start**: `npm install && npm start` or `go run .`
2. **Pre-filled `.env.example`**: Works with default GGID Docker Compose setup
3. **README with screenshots**: Show the login flow visually
4. **Dockerfile included**: Runnable in CI/CD
5. **Tested with latest GGID**: CI pipeline validates against main branch
6. **Demonstrates best practices**: Token refresh, error handling, logout
7. **Framework-appropriate patterns**: Uses the framework's conventions, not generic code

---

## 8. Existing Quickstart Doc Review

### docs/quickstart/5-minute-jwt.md

**Quality: Good (7/10)**

Strengths:
- Clean 3-step flow (register → login → use JWT)
- Three languages in one page (curl, Go, Node.js)
- Copy-paste ready snippets
- Concise — no unnecessary explanation

Weaknesses:
- Prerequisites section is too sparse (`sleep 30` with no context)
- No explanation of what the JWT contains
- No error handling examples (what if login fails?)
- No link to "what to do next" after the quickstart
- Go example has a bug: `io.Reader(resp.Body)` should be `resp.Body`
- No explanation of the X-Tenant-ID header (why is it needed?)

Recommendations:
- Add a "What just happened?" section explaining the flow
- Add error handling to each snippet
- Fix the Go `io.Reader` bug
- Add a "Next Steps" section linking to SDK quickstarts

### docs/quickstart/go-sdk.md

**Quality: Good (8/10)**

Strengths:
- True "3-line" JWT verification claim
- Clear progression: verify → protect handler → full example
- Gin integration section (most popular Go framework)
- Clean, well-formatted code

Weaknesses:
- No explanation of where to get the JWT secret
- No error handling in the verify example
- Missing Echo and Chi middleware examples
- No "how to test this" guidance

Recommendations:
- Add a note about finding the JWT secret
- Add error handling (`if err != nil`)
- Add Echo middleware example

### docs/quickstart/node-sdk.md

**Quality: Good (8/10)**

Strengths:
- npm and pnpm install instructions
- Clean Express middleware example
- Full example with scope checking
- `requireScope` helper function shown

Weaknesses:
- Package `@ggid/sdk-node` may not be published to npm yet
- No TypeScript examples (most Node.js devs use TS now)
- No error handling for token verification
- No Fastify or Koa examples

Recommendations:
- Verify package is published to npm
- Add TypeScript examples
- Add Fastify adapter

### docs/quickstart/oauth-login.md

**Quality: Moderate (6/10)**

Strengths:
- Complete OAuth 2.1 + PKCE flow documented
- All 5 steps are clear and sequential
- Shows expected token response

Weaknesses:
- Requires manual OAuth client registration (curl step) — high friction
- PKCE generation uses shell commands — should be a one-liner or SDK call
- No explanation of what PKCE is or why it's needed
- Authorization URL has placeholder `CHALLENGE` instead of `$CHALLENGE`
- No browser-based flow — purely curl

Recommendations:
- Seed default OAuth client in Docker Compose (improvement F7)
- Add a "What is PKCE?" sidebar
- Fix the `CHALLENGE` → `$CHALLENGE` bug
- Add a Node.js/Go SDK example for the OAuth flow

### docs/quickstart/rbac-permissions.md

**Quality: Good (7/10)**

Strengths:
- Clear 3-step flow (create role → assign → check)
- Shows the policy check API
- Documents permission format clearly

Weaknesses:
- Requires completing 5-minute JWT first (dependency chain)
- `USER_ID="usr_abc123"` is a placeholder — user won't know their actual ID
- No explanation of ABAC (only RBAC shown)
- No example of wildcard permissions

Recommendations:
- Show how to extract user_id from the registration response
- Add ABAC example
- Add wildcard permission example

### docs/integration-guides/express.md

**Quality: Very Good (9/10)**

Strengths:
- Comprehensive: minimal setup → scope auth → tenant-aware queries → error handling
- Shows optional auth pattern (public + protected routes)
- Environment variables documented
- Clean, production-ready code patterns

Weaknesses:
- No testing guidance
- No TypeScript types

### docs/integration-guides/gin.md

**Quality: Very Good (9/10)**

Strengths:
- Complete Gin integration with middleware
- Scope check middleware pattern
- Tenant-aware handler example
- GGID client usage example
- Environment variables documented

Weaknesses:
- No testing guidance
- No graceful shutdown example

### docs/integration-guides/spring-boot.md

**Quality: Excellent (9/10)**

Strengths:
- Complete Spring Security integration
- Maven and Gradle instructions
- JWT filter implementation with proper exception handling
- `@PreAuthorize` with scope-based access control
- `application.yml` configuration
- Production-ready patterns

Weaknesses:
- Long — might benefit from a "TL;DR" at the top
- No testing guidance

### Overall Doc Quality Assessment

| Document | Quality | Lines | Verdict |
|----------|---------|-------|---------|
| 5-minute-jwt.md | 7/10 | 103 | Good foundation, needs polish |
| go-sdk.md | 8/10 | 69 | Excellent conciseness |
| node-sdk.md | 8/10 | 79 | Strong, needs TypeScript |
| oauth-login.md | 6/10 | 83 | Functional but high friction |
| rbac-permissions.md | 7/10 | 66 | Good, needs user_id guidance |
| express.md | 9/10 | 121 | Production-ready guide |
| gin.md | 9/10 | 133 | Production-ready guide |
| spring-boot.md | 9/10 | 191 | Enterprise-ready guide |

**Overall: 7.9/10 average** — solid foundation, integration guides are
excellent, quickstarts need polish and sample apps.

### Missing Docs

| Missing Doc | Priority | Description |
|-------------|----------|-------------|
| quickstart/README.md | P0 | Landing page routing to the right quickstart |
| quickstart/docker-setup.md | P1 | Detailed Docker Compose setup with troubleshooting |
| quickstart/typescript.md | P1 | TypeScript-specific quickstart (types, decorators) |
| integration-guides/nextjs.md | P1 | Next.js SSR + edge middleware |
| integration-guides/react.md | P1 | React SPA with drop-in login component |
| integration-guides/fastapi.md | P2 | Python FastAPI integration |
| integration-guides/dotnet.md | P2 | ASP.NET Core integration |

---

## 9. Conclusion

### The Core Trade-Off

Auth0 and GGID represent fundamentally different philosophies:

- **Auth0** optimizes for **time-to-first-login**. The hosted login page,
  dashboard wizard, and sample apps get a developer from zero to authenticated
  in 5 minutes. The cost is vendor lock-in, data residency concerns, and a
  black-box implementation.

- **GGID** optimizes for **control and sovereignty**. Docker Compose, full
  source code, and Go-native performance give developers complete ownership.
  The cost is more steps to a working login UI and no sample apps.

### Where GGID Wins

GGID is the better choice for developers who:
- Need self-hosting (data sovereignty, compliance, air-gapped environments)
- Are building Go backends (native SDK, no Node.js dependency)
- Want to understand and audit their IAM implementation
- Need reproducible, scriptable environments (CI/CD, testing)
- Want to avoid vendor lock-in and per-MAU pricing

### Where Auth0 Wins

Auth0 is the better choice for developers who:
- Want the fastest possible time-to-market
- Need a hosted login page (no UI development)
- Want sample apps for every framework
- Prefer dashboard-driven configuration
- Are building SPAs with React/Vue

### The Path to Parity

GGID can close the quickstart gap within 3 months by focusing on three
high-impact improvements:

1. **Demo login page** — Give developers a URL to visit after `docker compose up`
2. **Sample app gallery** — 5 working apps (React, Go, Express, Next.js, Spring Boot)
3. **Drop-in React login component** — GGID's answer to `<LoginButton />`

With these three improvements, GGID's step count drops from 7 to 5 (matching
Auth0), and the developer experience transforms from "API-only curl quickstart"
to "click, log in, see it work."

### Final Score

| Dimension | Auth0 | GGID |
|-----------|-------|------|
| Time to first login | 9/10 | 6/10 |
| Login UI out of box | 10/10 | 2/10 |
| SDK quality | 9/10 | 7/10 |
| Sample apps | 10/10 | 1/10 |
| Self-hosted / sovereignty | 1/10 | 10/10 |
| Source code access | 0/10 | 10/10 |
| Go ecosystem | 3/10 | 10/10 |
| Documentation | 9/10 | 8/10 |
| Reproducibility | 4/10 | 10/10 |
| **Overall** | **7.2/10** | **7.5/10** |

GGID is already competitive on overall score. The quickstart improvements
outlined in Section 6 would push GGID to 8.5/10 — ahead of Auth0 for
developers who value control alongside ease of use.

---

*Related docs: [Auth0 Deep Comparison](./auth0-comparison.md) | [Feature Matrix](./auth0-keycloak-ggid-matrix.md) | [GGID Quickstarts](../quickstart/)*
