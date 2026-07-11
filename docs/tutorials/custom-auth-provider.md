# Tutorial: Custom Auth Provider

> How to implement a custom authentication provider that plugs into GGID's provider chain.

---

## Overview

GGID uses a chain-of-responsibility pattern for authentication. Each provider implements the `Provider` interface and is tried in order. This tutorial shows how to implement your own provider.

## The Provider Interface

```go
// pkg/authprovider/provider.go

type Provider interface {
    Name() string
    Authenticate(ctx context.Context, cred Credentials) (*AuthResult, error)
    Available(ctx context.Context) bool
}

type Credentials struct {
    Username string
    Password string
    Token    string // For OAuth/social flows
    Method   string // "password", "oauth", "webauthn", "saml"
}

type AuthResult struct {
    UserID     string
    TenantID   string
    Username   string
    Email      string
    Roles      []string
    Provider   string
    Attributes map[string]string
}
```

---

## Example 1: Database-Backed Provider

A custom provider that authenticates against an external database table:

```go
package myapp

import (
    "context"
    "crypto/sha256"
    "database/sql"
    "encoding/hex"

    "github.com/ggid/ggid/pkg/authprovider"
)

type ExternalDBProvider struct {
    db *sql.DB
}

func NewExternalDBProvider(db *sql.DB) *ExternalDBProvider {
    return &ExternalDBProvider{db: db}
}

func (p *ExternalDBProvider) Name() string {
    return "external_db"
}

func (p *ExternalDBProvider) Available(ctx context.Context) bool {
    return p.db != nil && p.db.PingContext(ctx) == nil
}

func (p *ExternalDBProvider) Authenticate(ctx context.Context, cred authprovider.Credentials) (*authprovider.AuthResult, error) {
    // Hash password for comparison
    hash := sha256.Sum256([]byte(cred.Password))
    hashStr := hex.EncodeToString(hash[:])

    var (
        userID string
        email  string
    )

    err := p.db.QueryRowContext(ctx,
        `SELECT id, email FROM external_users WHERE username = $1 AND password_hash = $2 AND active = true`,
        cred.Username, hashStr,
    ).Scan(&userID, &email)

    if err == sql.ErrNoRows {
        return nil, authprovider.ErrInvalidCredentials
    }
    if err != nil {
        return nil, err
    }

    return &authprovider.AuthResult{
        UserID:   userID,
        Username: cred.Username,
        Email:    email,
        Provider: p.Name(),
        Roles:    []string{"end_user"},
    }, nil
}
```

---

## Example 2: Custom LDAP Filter Provider

A provider with custom LDAP filter logic (e.g., filtering by department):

```go
package myapp

import (
    "context"
    "fmt"

    "github.com/ggid/ggid/pkg/authprovider"
)

type DepartmentLDAPProvider struct {
    baseDN     string
    department string
    inner      authprovider.Provider // wraps standard LDAP provider
}

func NewDepartmentLDAPProvider(baseDN, department string, inner authprovider.Provider) *DepartmentLDAPProvider {
    return &DepartmentLDAPProvider{
        baseDN:     baseDN,
        department: department,
        inner:      inner,
    }
}

func (p *DepartmentLDAPProvider) Name() string {
    return fmt.Sprintf("ldap_dept_%s", p.department)
}

func (p *DepartmentLDAPProvider) Available(ctx context.Context) bool {
    return p.inner.Available(ctx)
}

func (p *DepartmentLDAPProvider) Authenticate(ctx context.Context, cred authprovider.Credentials) (*authprovider.AuthResult, error) {
    result, err := p.inner.Authenticate(ctx, cred)
    if err != nil {
        return nil, err
    }

    // Check department attribute
    if dept, ok := result.Attributes["department"]; !ok || dept != p.department {
        return nil, authprovider.ErrInvalidCredentials
    }

    result.Provider = p.Name()
    return result, nil
}
```

---

## Example 3: Social Provider Stub

A stub for a custom social login provider:

```go
package myapp

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "net/url"

    "github.com/ggid/ggid/pkg/authprovider"
)

type CustomSocialProvider struct {
    clientID     string
    clientSecret string
    authURL      string
    tokenURL     string
    userInfoURL  string
}

func NewCustomSocialProvider(clientID, clientSecret, authURL, tokenURL, userInfoURL string) *CustomSocialProvider {
    return &CustomSocialProvider{
        clientID:     clientID,
        clientSecret: clientSecret,
        authURL:      authURL,
        tokenURL:     tokenURL,
        userInfoURL:  userInfoURL,
    }
}

func (p *CustomSocialProvider) Name() string {
    return "custom_social"
}

func (p *CustomSocialProvider) Available(ctx context.Context) bool {
    return p.clientID != ""
}

func (p *CustomSocialProvider) Authenticate(ctx context.Context, cred authprovider.Credentials) (*authprovider.AuthResult, error) {
    // Step 1: Exchange code for access token
    resp, err := http.PostForm(p.tokenURL, url.Values{
        "grant_type":    {"authorization_code"},
        "code":          {cred.Token},
        "client_id":     {p.clientID},
        "client_secret": {p.clientSecret},
    })
    if err != nil {
        return nil, fmt.Errorf("token exchange failed: %w", err)
    }
    defer resp.Body.Close()

    var tokenResp struct {
        AccessToken string `json:"access_token"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
        return nil, err
    }

    // Step 2: Get user info
    req, _ := http.NewRequestWithContext(ctx, "GET", p.userInfoURL, nil)
    req.Header.Set("Authorization", "Bearer "+tokenResp.AccessToken)

    resp2, err := http.DefaultClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("user info fetch failed: %w", err)
    }
    defer resp2.Body.Close()

    var user struct {
        ID    string `json:"id"`
        Email string `json:"email"`
        Name  string `json:"name"`
    }
    body, _ := io.ReadAll(resp2.Body)
    if err := json.Unmarshal(body, &user); err != nil {
        return nil, err
    }

    return &authprovider.AuthResult{
        UserID:   user.ID,
        Username: user.Email,
        Email:    user.Email,
        Provider: p.Name(),
        Roles:    []string{"end_user"},
        Attributes: map[string]string{
            "name": user.Name,
        },
    }, nil
}
```

---

## Wiring Into the Chain

```go
// services/auth/cmd/main.go

func buildAuthChain(db *sql.DB, ldapConfig *LDAPConfig) *authprovider.Chain {
    chain := authprovider.NewChain(
        authprovider.NewLocalProvider(db, cryptoSvc),
    )

    // Add LDAP if configured
    if ldapConfig.URL != "" {
        chain.Add(authprovider.NewLDAPProvider(ldapConfig))
    }

    // Add your custom provider
    chain.Add(NewExternalDBProvider(externalDB))

    return chain
}
```

---

## Testing Your Provider

```go
func TestExternalDBProvider(t *testing.T) {
    db := setupTestDB(t)
    provider := NewExternalDBProvider(db)

    // Insert test user
    db.Exec("INSERT INTO external_users (id, username, password_hash, email, active) VALUES (?, ?, ?, ?, true)",
        "usr_1", "testuser", hashPassword("pass123"), "test@example.com")

    tests := []struct {
        name     string
        cred     authprovider.Credentials
        wantErr  bool
    }{
        {
            name: "valid credentials",
            cred: authprovider.Credentials{Username: "testuser", Password: "pass123", Method: "password"},
            wantErr: false,
        },
        {
            name: "wrong password",
            cred: authprovider.Credentials{Username: "testuser", Password: "wrong", Method: "password"},
            wantErr: true,
        },
        {
            name: "nonexistent user",
            cred: authprovider.Credentials{Username: "nobody", Password: "pass", Method: "password"},
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := provider.Authenticate(context.Background(), tt.cred)
            if tt.wantErr {
                require.Error(t, err)
            } else {
                require.NoError(t, err)
                assert.Equal(t, "external_db", result.Provider)
                assert.Equal(t, "test@example.com", result.Email)
            }
        })
    }
}
```

---

## Summary

- Implement `authprovider.Provider` interface (3 methods)
- Add to chain via `chain.Add(yourProvider)`
- Chain tries providers in order — first success wins
- All providers produce the same `AuthResult` for downstream consistency
- See [ADR-003: Provider Chain](../design/adr-003-provider-chain.md) for design rationale

---

*Last updated: 2025-07-11*