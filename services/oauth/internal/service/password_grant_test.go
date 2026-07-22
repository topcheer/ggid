package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	pkgcrypto "github.com/ggid/ggid/pkg/crypto"
	"github.com/ggid/ggid/services/oauth/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// --- PasswordGrant pool fakes ---

type fakeRow struct {
	scanFn func(dest ...any) error
}

func (r fakeRow) Scan(dest ...any) error { return r.scanFn(dest...) }

// fakePool implements PoolQuerier for PasswordGrant tests.
// QueryRow calls are answered in invocation order via rowFns; Query always
// errors so fetchUserPermissions/Roles degrade to empty lists.
type fakePool struct {
	userID   uuid.UUID
	credHash string
}

func (p *fakePool) QueryRow(_ context.Context, sql string, _ ...any) pgx.Row {
	switch {
	case strings.Contains(sql, "FROM users WHERE username"):
		return fakeRow{scanFn: func(dest ...any) error {
			if len(dest) == 1 {
				if idp, ok := dest[0].(*uuid.UUID); ok {
					*idp = p.userID
					return nil
				}
			}
			return errors.New("bad dest")
		}}
	case strings.Contains(sql, "FROM credentials"):
		return fakeRow{scanFn: func(dest ...any) error {
			if len(dest) == 1 {
				if sp, ok := dest[0].(*string); ok {
					*sp = p.credHash
					return nil
				}
			}
			return errors.New("bad dest")
		}}
	}
	return fakeRow{scanFn: func(dest ...any) error { return errors.New("no rows") }}
}

func (p *fakePool) Query(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
	return nil, errors.New("not implemented")
}

// --- Tests ---

func TestPasswordGrant_OfflineAccessIssuesRefreshToken(t *testing.T) {
	svc, clientRepo, _, tokenRepo := newTestOAuthService()
	userID := uuid.New()
	hash, err := pkgcrypto.HashPassword("correct-pass-123")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	svc.SetPool(&fakePool{userID: userID, credHash: hash})

	client := &domain.OAuthClient{
		ID:         uuid.New(),
		TenantID:   testTenantID,
		ClientID:   "ggid-console",
		Name:       "Console",
		Type:       domain.ClientTypePublic,
		GrantTypes: []string{"password", "authorization_code", "refresh_token"},
		Enabled:    true,
	}
	_ = clientRepo.CreateClient(context.Background(), client)

	resp, err := svc.PasswordGrant(context.Background(), &PasswordGrantRequest{
		TenantID: testTenantID,
		Username: "admin",
		Password: "correct-pass-123",
		ClientID: "ggid-console",
		Scope:    []string{"openid", "profile", "email", "offline_access"},
	})
	if err != nil {
		t.Fatalf("PasswordGrant: %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("expected access token")
	}
	if resp.RefreshToken == "" {
		t.Fatal("offline_access password grant must issue a refresh token")
	}
	if len(tokenRepo.refreshTokens) != 1 {
		t.Fatalf("expected 1 stored refresh token, got %d", len(tokenRepo.refreshTokens))
	}
	rec := tokenRepo.refreshTokens[0]
	if rec.FamilyID != rec.ID.String() {
		t.Errorf("FamilyID = %q, want family root %q", rec.FamilyID, rec.ID.String())
	}
	if rec.ClientID != client.ID || rec.UserID != userID {
		t.Errorf("record binding wrong: client=%v user=%v", rec.ClientID, rec.UserID)
	}
}

func TestPasswordGrant_NoOfflineAccess_NoRefreshToken(t *testing.T) {
	svc, _, _, tokenRepo := newTestOAuthService()
	hash, _ := pkgcrypto.HashPassword("correct-pass-123")
	svc.SetPool(&fakePool{userID: uuid.New(), credHash: hash})

	resp, err := svc.PasswordGrant(context.Background(), &PasswordGrantRequest{
		TenantID: testTenantID,
		Username: "admin",
		Password: "correct-pass-123",
		ClientID: "ggid-console",
		Scope:    []string{"openid", "profile"},
	})
	if err != nil {
		t.Fatalf("PasswordGrant: %v", err)
	}
	if resp.RefreshToken != "" {
		t.Error("no refresh token without offline_access")
	}
	if len(tokenRepo.refreshTokens) != 0 {
		t.Error("nothing should be stored")
	}
}

func TestPasswordGrant_WrongPassword(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	hash, _ := pkgcrypto.HashPassword("correct-pass-123")
	svc.SetPool(&fakePool{userID: uuid.New(), credHash: hash})

	_, err := svc.PasswordGrant(context.Background(), &PasswordGrantRequest{
		TenantID: testTenantID,
		Username: "admin",
		Password: "wrong",
		ClientID: "ggid-console",
		Scope:    []string{"openid"},
	})
	if err == nil {
		t.Error("wrong password must fail")
	}
}

// TestPasswordGrant_NoCredential_FailsClosed is the regression test for the
// P0 auth bypass: when the credentials lookup errors or returns no row, the
// grant must REJECT — never skip verification.
func TestPasswordGrant_NoCredential_FailsClosed(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	// credHash empty → credentials query "returns no row".
	svc.SetPool(&fakePool{userID: uuid.New(), credHash: ""})

	_, err := svc.PasswordGrant(context.Background(), &PasswordGrantRequest{
		TenantID: testTenantID,
		Username: "admin",
		Password: "anything",
		ClientID: "ggid-console",
		Scope:    []string{"openid"},
	})
	if err == nil {
		t.Fatal("missing credential must fail closed, not skip verification")
	}
}
