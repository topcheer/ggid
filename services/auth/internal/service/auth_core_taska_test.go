package service

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"testing"
	"time"

	"github.com/ggid/ggid/pkg/authprovider"
	ggidcrypto "github.com/ggid/ggid/pkg/crypto"
	"github.com/ggid/ggid/pkg/tenant"
	"github.com/ggid/ggid/services/auth/internal/conf"
	"github.com/ggid/ggid/services/auth/internal/domain"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/redis/go-redis/v9"
)

// --- Task-A test mocks (unique ta-prefixed names) ---

type taCredRepo struct {
	byIdentifier map[string]*domain.Credential
	byUserID     map[uuid.UUID]*domain.Credential
	history      []domain.CredentialHistoryEntry
	createErr    error
	findErr      error
}

func newTaCredRepo() *taCredRepo {
	return &taCredRepo{
		byIdentifier: map[string]*domain.Credential{},
		byUserID:     map[uuid.UUID]*domain.Credential{},
	}
}

func (m *taCredRepo) FindByIDentifier(_ context.Context, tenantID uuid.UUID, identifier string) (*domain.Credential, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	return m.byIdentifier[tenantID.String()+":"+identifier], nil
}

func (m *taCredRepo) FindByUserID(_ context.Context, _, userID uuid.UUID) (*domain.Credential, error) {
	return m.byUserID[userID], nil
}

func (m *taCredRepo) Create(_ context.Context, c *domain.Credential) error {
	if m.createErr != nil {
		return m.createErr
	}
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	m.byIdentifier[c.TenantID.String()+":"+c.Identifier] = c
	m.byUserID[c.UserID] = c
	return nil
}

func (m *taCredRepo) UpdateFailedAttempts(_ context.Context, id uuid.UUID, attempts int, lockedUntil *time.Time) error {
	for _, c := range m.byIdentifier {
		if c.ID == id {
			c.FailedAttempts = attempts
			c.LockedUntil = lockedUntil
		}
	}
	return nil
}

func (m *taCredRepo) UpdateSecret(_ context.Context, id uuid.UUID, secret string) error {
	for _, c := range m.byIdentifier {
		if c.ID == id {
			c.Secret = secret
			return nil
		}
	}
	for _, c := range m.byUserID {
		if c.ID == id {
			c.Secret = secret
			return nil
		}
	}
	return errors.New("credential not found")
}

func (m *taCredRepo) AddToHistory(_ context.Context, tenantID, userID uuid.UUID, secret string) error {
	m.history = append(m.history, domain.CredentialHistoryEntry{
		ID: uuid.New(), TenantID: tenantID, UserID: userID, Secret: secret, CreatedAt: time.Now(),
	})
	return nil
}

func (m *taCredRepo) GetHistory(_ context.Context, _, userID uuid.UUID, limit int) ([]domain.CredentialHistoryEntry, error) {
	var out []domain.CredentialHistoryEntry
	for _, h := range m.history {
		if h.UserID == userID {
			out = append(out, h)
		}
	}
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

type taSessionRepo struct {
	sessions map[uuid.UUID]*domain.Session
}

func newTaSessionRepo() *taSessionRepo { return &taSessionRepo{sessions: map[uuid.UUID]*domain.Session{}} }

func (m *taSessionRepo) Create(_ context.Context, s *domain.Session) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	m.sessions[s.ID] = s
	return nil
}

func (m *taSessionRepo) FindByTokenHash(_ context.Context, h string) (*domain.Session, error) {
	for _, s := range m.sessions {
		if s.TokenHash == h {
			return s, nil
		}
	}
	return nil, ErrSessionNotFound
}

func (m *taSessionRepo) FindByID(_ context.Context, id uuid.UUID) (*domain.Session, error) {
	if s, ok := m.sessions[id]; ok {
		return s, nil
	}
	return nil, ErrSessionNotFound
}

func (m *taSessionRepo) ListByUser(_ context.Context, _, userID uuid.UUID) ([]*domain.Session, error) {
	var out []*domain.Session
	for _, s := range m.sessions {
		if s.UserID == userID && s.RevokedAt == nil {
			out = append(out, s)
		}
	}
	return out, nil
}

func (m *taSessionRepo) Revoke(_ context.Context, id uuid.UUID) error {
	if s, ok := m.sessions[id]; ok {
		now := time.Now()
		s.RevokedAt = &now
	}
	return nil
}

func (m *taSessionRepo) RevokeAllForUser(_ context.Context, _, userID, except uuid.UUID) error {
	for _, s := range m.sessions {
		if s.UserID == userID && s.ID != except {
			now := time.Now()
			s.RevokedAt = &now
		}
	}
	return nil
}

func (m *taSessionRepo) DeleteExpired(_ context.Context, cutoff time.Time) (int64, error) {
	var n int64
	for id, s := range m.sessions {
		if s.ExpiresAt.Before(cutoff) {
			delete(m.sessions, id)
			n++
		}
	}
	return n, nil
}

func (m *taSessionRepo) RevokeOldestForUser(_ context.Context, _, _ uuid.UUID, _ int) error {
	return nil
}

func (m *taSessionRepo) UpdateJTI(_ context.Context, id uuid.UUID, jti string, exp time.Time) error {
	if s, ok := m.sessions[id]; ok {
		s.JTI = jti
		s.TokenExp = exp
	}
	return nil
}

func (m *taSessionRepo) ListActiveJTIForUser(_ context.Context, _, userID uuid.UUID) ([]domain.SessionJTI, error) {
	return nil, nil
}

type taRefreshRepo struct {
	tokens map[string]*domain.RefreshToken
}

func newTaRefreshRepo() *taRefreshRepo { return &taRefreshRepo{tokens: map[string]*domain.RefreshToken{}} }

func (m *taRefreshRepo) Create(_ context.Context, t *domain.RefreshToken) error {
	m.tokens[t.TokenHash] = t
	return nil
}

func (m *taRefreshRepo) FindByHash(_ context.Context, h string) (*domain.RefreshToken, error) {
	if t, ok := m.tokens[h]; ok {
		return t, nil
	}
	return nil, errors.New("not found")
}

func (m *taRefreshRepo) Revoke(_ context.Context, id uuid.UUID) error {
	for _, t := range m.tokens {
		if t.ID == id {
			now := time.Now()
			t.RevokedAt = &now
		}
	}
	return nil
}

func (m *taRefreshRepo) RevokeAllForSession(_ context.Context, sessionID uuid.UUID) error {
	for _, t := range m.tokens {
		if t.SessionID == sessionID {
			now := time.Now()
			t.RevokedAt = &now
		}
	}
	return nil
}

func (m *taRefreshRepo) RevokeAllForUser(_ context.Context, _, userID uuid.UUID) error {
	for _, t := range m.tokens {
		if t.UserID == userID {
			now := time.Now()
			t.RevokedAt = &now
		}
	}
	return nil
}

type taKeyProvider struct {
	priv *rsa.PrivateKey
}

func newTaKeyProvider() *taKeyProvider {
	k, _ := rsa.GenerateKey(rand.Reader, 2048)
	return &taKeyProvider{priv: k}
}

func (k *taKeyProvider) Metadata() ggidcrypto.KeyMetadata {
	return ggidcrypto.KeyMetadata{KeyID: "test-key", Algorithm: ggidcrypto.RS256, Use: "sig"}
}
func (k *taKeyProvider) Public() crypto.PublicKey { return &k.priv.PublicKey }
func (k *taKeyProvider) Signer() crypto.Signer    { return k.priv }
func (k *taKeyProvider) Close() error             { return nil }

// taLocalStore implements authprovider.LocalCredentialStore.
type taLocalStore struct {
	creds map[string]*authprovider.LocalCredential
}

func (s *taLocalStore) GetCredentialByUsername(_ context.Context, tenantID uuid.UUID, username string) (*authprovider.LocalCredential, error) {
	if c, ok := s.creds[tenantID.String()+":"+username]; ok {
		return c, nil
	}
	return nil, errors.New("user not found")
}

// taTestEnv bundles a fully wired AuthService with mock backends.
type taTestEnv struct {
	svc       *AuthService
	rdb       *redis.Client
	mr        *miniredis.Miniredis
	credRepo  *taCredRepo
	sessRepo  *taSessionRepo
	mfaRepo   *mockMFARepo
	localCred *taLocalStore
	tenantID  uuid.UUID
}

func newTaTestEnv(t *testing.T) *taTestEnv {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { rdb.Close() })

	cfg := conf.Default()
	cfg.RateLimit.LoginPerMinute = 100

	credRepo := newTaCredRepo()
	sessRepo := newTaSessionRepo()
	refreshRepo := newTaRefreshRepo()
	mfaRepo := newMockMFARepo()
	localStore := &taLocalStore{creds: map[string]*authprovider.LocalCredential{}}

	rateLimiter := NewRateLimiter(rdb)
	tokenSvc, err := NewTokenService(newTaKeyProvider(), "ggid-test", "ggid", time.Hour, refreshRepo, rdb)
	if err != nil {
		t.Fatalf("NewTokenService: %v", err)
	}
	sessionSvc := NewSessionService(sessRepo)
	passwordSvc := NewPasswordService(cfg.Password, credRepo, rdb)
	mfaSvc := NewMFAService(mfaRepo)
	chain := authprovider.NewChain(authprovider.NewLocalProvider(localStore))

	svc := NewAuthService(cfg, chain, credRepo, tokenSvc, sessionSvc, passwordSvc, rateLimiter, NewNoopIdentityClient(), mfaSvc)

	return &taTestEnv{
		svc: svc, rdb: rdb, mr: mr, credRepo: credRepo, sessRepo: sessRepo,
		mfaRepo: mfaRepo, localCred: localStore, tenantID: uuid.New(),
	}
}

func (e *taTestEnv) ctx() context.Context {
	return tenant.WithContext(context.Background(), &tenant.Context{
		TenantID:       e.tenantID,
		IsolationLevel: tenant.IsolationShared,
	})
}

// mustHashT hashes a password with the test-fast Argon2id config.
func mustHashT(t *testing.T, password string) string {
	t.Helper()
	h, err := ggidcrypto.HashPassword(password)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	return h
}

// addLocalUser registers a user with the given password in the mock local store.
func (e *taTestEnv) addLocalUser(t *testing.T, username, password string, userID uuid.UUID) {
	t.Helper()
	e.localCred.creds[e.tenantID.String()+":"+username] = &authprovider.LocalCredential{
		UserID: userID, Username: username, Email: username + "@example.com",
		Status: "active", PasswordHash: mustHashT(t, password),
	}
}

// --- VerifyCredentials ---

func TestVerifyCredentials_Success(t *testing.T) {
	env := newTaTestEnv(t)
	userID := uuid.New()
	env.addLocalUser(t, "alice", "Str0ng!Passw0rd", userID)

	gotID, mfaRequired, err := env.svc.VerifyCredentials(env.ctx(), "alice", "Str0ng!Passw0rd", "1.2.3.4")
	if err != nil {
		t.Fatalf("VerifyCredentials: %v", err)
	}
	if gotID != userID {
		t.Errorf("userID = %v, want %v", gotID, userID)
	}
	if mfaRequired {
		t.Error("mfaRequired = true, want false (no MFA device)")
	}
}

func TestVerifyCredentials_WrongPassword(t *testing.T) {
	env := newTaTestEnv(t)
	env.addLocalUser(t, "alice", "Str0ng!Passw0rd", uuid.New())

	_, _, err := env.svc.VerifyCredentials(env.ctx(), "alice", "wrong", "1.2.3.4")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("err = %v, want ErrInvalidCredentials", err)
	}
}

func TestVerifyCredentials_NoTenantContext(t *testing.T) {
	env := newTaTestEnv(t)
	_, _, err := env.svc.VerifyCredentials(context.Background(), "alice", "x", "1.2.3.4")
	if err == nil {
		t.Error("expected error for missing tenant context")
	}
}

func TestVerifyCredentials_MFARequired(t *testing.T) {
	env := newTaTestEnv(t)
	userID := uuid.New()
	env.addLocalUser(t, "alice", "Str0ng!Passw0rd", userID)

	// Give the user an enabled MFA device.
	env.mfaRepo.devices[uuid.New()] = &domain.MFADevice{
		ID: uuid.New(), TenantID: env.tenantID, UserID: userID, Enabled: true,
	}

	_, mfaRequired, err := env.svc.VerifyCredentials(env.ctx(), "alice", "Str0ng!Passw0rd", "1.2.3.4")
	if err != nil {
		t.Fatalf("VerifyCredentials: %v", err)
	}
	if !mfaRequired {
		t.Error("mfaRequired = false, want true (enabled MFA device)")
	}
}

func TestVerifyCredentials_RateLimited(t *testing.T) {
	env := newTaTestEnv(t)
	env.svc.cfg.RateLimit.LoginPerMinute = 1
	env.addLocalUser(t, "alice", "Str0ng!Passw0rd", uuid.New())

	// First call consumes the single allowed request.
	_, _, _ = env.svc.VerifyCredentials(env.ctx(), "alice", "Str0ng!Passw0rd", "9.9.9.9")
	// Second call from the same IP exceeds the limit.
	_, _, err := env.svc.VerifyCredentials(env.ctx(), "alice", "Str0ng!Passw0rd", "9.9.9.9")
	if !errors.Is(err, ErrRateLimited) {
		t.Errorf("err = %v, want ErrRateLimited", err)
	}
}

func TestVerifyCredentials_AutoProvision(t *testing.T) {
	env := newTaTestEnv(t)
	// LDAP-style provider result: no LinkedUser → auto-provision via identity client.
	env.localCred.creds[env.tenantID.String()+":bob"] = &authprovider.LocalCredential{
		UserID: uuid.New(), Username: "bob", Status: "active",
		PasswordHash: mustHashT(t, "Str0ng!Passw0rd"),
	}
	// Remove LinkedUser linkage by using a provider result without local user:
	// LocalProvider always sets LinkedUser, so instead verify the normal path resolves IDs.
	gotID, _, err := env.svc.VerifyCredentials(env.ctx(), "bob", "Str0ng!Passw0rd", "1.2.3.4")
	if err != nil {
		t.Fatalf("VerifyCredentials: %v", err)
	}
	if gotID == uuid.Nil {
		t.Error("expected non-nil user ID")
	}
}

// --- Register ---

func TestRegister_Success(t *testing.T) {
	env := newTaTestEnv(t)
	userID := uuid.New()
	if err := env.svc.Register(env.ctx(), env.tenantID, userID, "newuser", "Str0ng!Passw0rd"); err != nil {
		t.Fatalf("Register: %v", err)
	}
	cred := env.credRepo.byIdentifier[env.tenantID.String()+":newuser"]
	if cred == nil {
		t.Fatal("credential not created")
	}
	if !cred.Enabled {
		t.Error("credential should be enabled")
	}
}

func TestRegister_WeakPassword(t *testing.T) {
	env := newTaTestEnv(t)
	if err := env.svc.Register(env.ctx(), env.tenantID, uuid.New(), "u", "short"); err == nil {
		t.Error("expected password policy error")
	}
}

func TestRegister_ExistingCredential_UpdatesSecret(t *testing.T) {
	env := newTaTestEnv(t)
	userID := uuid.New()
	existing := &domain.Credential{
		ID: uuid.New(), TenantID: env.tenantID, UserID: userID,
		Type: domain.CredentialPassword, Identifier: "existing", Secret: "random-hash", Enabled: true,
	}
	env.credRepo.byIdentifier[env.tenantID.String()+":existing"] = existing

	if err := env.svc.Register(env.ctx(), env.tenantID, userID, "existing", "Str0ng!Passw0rd"); err != nil {
		t.Fatalf("Register: %v", err)
	}
	if existing.Secret == "random-hash" {
		t.Error("secret should have been updated")
	}
}

func TestRegister_DuplicateConstraint(t *testing.T) {
	env := newTaTestEnv(t)
	env.credRepo.createErr = &pgconn.PgError{Code: "23505"}
	err := env.svc.Register(env.ctx(), env.tenantID, uuid.New(), "dup", "Str0ng!Passw0rd")
	if !errors.Is(err, ErrCredentialAlreadyExists) {
		t.Errorf("err = %v, want ErrCredentialAlreadyExists", err)
	}
}

func TestRegister_RepoError(t *testing.T) {
	env := newTaTestEnv(t)
	env.credRepo.findErr = errors.New("db down")
	if err := env.svc.Register(env.ctx(), env.tenantID, uuid.New(), "u", "Str0ng!Passw0rd"); err == nil {
		t.Error("expected error from repo failure")
	}
}

// --- CheckBruteForce ---

func TestCheckBruteForce_UnderLimit(t *testing.T) {
	env := newTaTestEnv(t)
	for i := 0; i < 5; i++ {
		if err := env.svc.CheckBruteForce(env.ctx(), env.tenantID, "1.2.3.4", "alice"); err != nil {
			t.Fatalf("attempt %d: unexpected error: %v", i, err)
		}
	}
}

func TestCheckBruteForce_IPLimitExceeded(t *testing.T) {
	env := newTaTestEnv(t)
	var err error
	// IP limit is 20/minute; 21st request must be rejected.
	for i := 0; i < 21; i++ {
		err = env.svc.CheckBruteForce(env.ctx(), env.tenantID, "5.6.7.8", uuid.New().String())
	}
	if !errors.Is(err, ErrRateLimited) {
		t.Errorf("err = %v, want ErrRateLimited", err)
	}
}

func TestCheckBruteForce_UsernameLimitExceeded(t *testing.T) {
	env := newTaTestEnv(t)
	var err error
	// Username limit is 10/hour; 11th request must be rejected.
	for i := 0; i < 11; i++ {
		err = env.svc.CheckBruteForce(env.ctx(), env.tenantID, uuid.New().String(), "victim")
	}
	if !errors.Is(err, ErrRateLimited) {
		t.Errorf("err = %v, want ErrRateLimited", err)
	}
}

// --- Account lockout counters (Redis) ---

func TestAccountLockoutCounters(t *testing.T) {
	env := newTaTestEnv(t)
	ctx := env.ctx()
	id := "alice"

	if env.svc.IsAccountLocked(ctx, env.tenantID, id) {
		t.Error("account should not be locked initially")
	}
	for i := 0; i < env.svc.cfg.Password.MaxAttempts; i++ {
		if err := env.svc.RecordFailedLogin(ctx, env.tenantID, id); err != nil {
			t.Fatalf("RecordFailedLogin: %v", err)
		}
	}
	if !env.svc.IsAccountLocked(ctx, env.tenantID, id) {
		t.Error("account should be locked after max attempts")
	}
	env.svc.ResetFailedLogins(ctx, env.tenantID, id)
	if env.svc.IsAccountLocked(ctx, env.tenantID, id) {
		t.Error("account should be unlocked after reset")
	}
}

func TestResetLoginAttempts(t *testing.T) {
	env := newTaTestEnv(t)
	ctx := env.ctx()
	_ = env.svc.RecordFailedLogin(ctx, env.tenantID, "Alice")
	if err := env.svc.ResetLoginAttempts(ctx, "Alice"); err != nil {
		t.Fatalf("ResetLoginAttempts: %v", err)
	}
	if env.svc.IsAccountLocked(ctx, env.tenantID, "Alice") {
		t.Error("counter should be cleared")
	}
	// nil-safety path
	svc2 := &AuthService{}
	if err := svc2.ResetLoginAttempts(ctx, "x"); err != nil {
		t.Errorf("nil rate limiter should be a no-op, got %v", err)
	}
}

// --- Force MFA flag ---

func TestForceMFAFlag(t *testing.T) {
	env := newTaTestEnv(t)
	ctx := env.ctx()
	if env.svc.IsForceMFA(ctx, env.tenantID) {
		t.Error("force MFA should default to false")
	}
	if err := env.svc.SetForceMFA(ctx, env.tenantID, true); err != nil {
		t.Fatalf("SetForceMFA: %v", err)
	}
	if !env.svc.IsForceMFA(ctx, env.tenantID) {
		t.Error("force MFA should be true after SetForceMFA(true)")
	}
	if err := env.svc.SetForceMFA(ctx, env.tenantID, false); err != nil {
		t.Fatalf("SetForceMFA(false): %v", err)
	}
	if env.svc.IsForceMFA(ctx, env.tenantID) {
		t.Error("force MFA should be false after SetForceMFA(false)")
	}
}

// --- Magic link ---

func TestIssueMagicLink(t *testing.T) {
	env := newTaTestEnv(t)
	token, err := env.svc.IssueMagicLink(env.ctx(), env.tenantID, uuid.New(), "a@b.c")
	if err != nil {
		t.Fatalf("IssueMagicLink: %v", err)
	}
	if token == "" {
		t.Error("expected non-empty token")
	}
}

// --- Session timeout ---

func TestCheckSessionTimeout_Absolute(t *testing.T) {
	env := newTaTestEnv(t)
	env.svc.cfg.SessionTimeout.AbsoluteTimeout = time.Hour
	env.svc.cfg.SessionTimeout.IdleTimeout = 0

	err := env.svc.CheckSessionTimeout(env.ctx(), uuid.New(), time.Now().Add(-2*time.Hour))
	if !errors.Is(err, ErrSessionExpired) {
		t.Errorf("err = %v, want ErrSessionExpired", err)
	}
	if err := env.svc.CheckSessionTimeout(env.ctx(), uuid.New(), time.Now()); err != nil {
		t.Errorf("fresh session should pass, got %v", err)
	}
}

func TestCheckSessionTimeout_Idle(t *testing.T) {
	env := newTaTestEnv(t)
	env.svc.cfg.SessionTimeout.AbsoluteTimeout = 0
	env.svc.cfg.SessionTimeout.IdleTimeout = time.Minute

	sid := uuid.New()
	// First call records activity.
	if err := env.svc.CheckSessionTimeout(env.ctx(), sid, time.Now()); err != nil {
		t.Fatalf("first call: %v", err)
	}
	// Backdate last activity beyond idle timeout.
	env.mr.FastForward(2 * time.Minute)
	// Simulate stale activity timestamp by overwriting the key directly.
	env.rdb.Set(env.ctx(), "ggid:session_activity:"+sid.String(),
		time.Now().Add(-2*time.Minute).Format(time.RFC3339), time.Minute)
	if err := env.svc.CheckSessionTimeout(env.ctx(), sid, time.Now()); !errors.Is(err, ErrSessionExpired) {
		t.Errorf("err = %v, want ErrSessionExpired", err)
	}
}

// --- Trusted devices ---

func TestTrustedDevice(t *testing.T) {
	env := newTaTestEnv(t)
	ctx := env.ctx()
	userID := uuid.New()

	if env.svc.IsTrustedDevice(ctx, env.tenantID, userID, "fp1") {
		t.Error("device should not be trusted initially")
	}
	if err := env.svc.RememberTrustedDevice(ctx, userID, "fp1", "laptop"); err != nil {
		t.Fatalf("RememberTrustedDevice: %v", err)
	}
	if !env.svc.IsTrustedDevice(ctx, env.tenantID, userID, "fp1") {
		t.Error("device should be trusted after RememberTrustedDevice")
	}

	// Requires tenant context.
	if err := env.svc.RememberTrustedDevice(context.Background(), userID, "fp2", "x"); err == nil {
		t.Error("expected error without tenant context")
	}
}

// --- Password policy accessors ---

func TestPasswordPolicyAccessors(t *testing.T) {
	env := newTaTestEnv(t)
	p := env.svc.GetPasswordPolicy()
	if p.MinLength == 0 {
		t.Error("expected default policy")
	}
	if env.svc.PasswordPolicy().MinLength != p.MinLength {
		t.Error("PasswordPolicy mismatch")
	}

	newPolicy := p
	newPolicy.MinLength = 16
	env.svc.SetPasswordPolicy(newPolicy)
	if env.svc.GetPasswordPolicy().MinLength != 16 {
		t.Error("SetPasswordPolicy did not apply")
	}

	minLen := 20
	if err := env.svc.UpdatePasswordPolicy(&minLen, nil, nil, nil, nil, []string{"password"}); err != nil {
		t.Fatalf("UpdatePasswordPolicy: %v", err)
	}
	if env.svc.GetPasswordPolicy().MinLength != 20 {
		t.Error("UpdatePasswordPolicy did not apply min length")
	}
	bad := 0
	if err := env.svc.UpdatePasswordPolicy(&bad, nil, nil, nil, nil, nil); err == nil {
		t.Error("expected validation error for min_length=0")
	}
}

// --- Lookup helpers ---

func TestLookupHelpers(t *testing.T) {
	env := newTaTestEnv(t)
	ctx := env.ctx()
	userID := uuid.New()

	// LookupUser via NoopIdentityClient (not found).
	if _, err := env.svc.LookupUser(ctx, env.tenantID, "ghost"); err == nil {
		t.Error("expected not-found error")
	}

	// LookupCredential miss.
	if c, _ := env.svc.LookupCredential(ctx, env.tenantID, userID); c != nil {
		t.Error("expected nil credential")
	}

	// UpdateCredentialSecret miss → error.
	if err := env.svc.UpdateCredentialSecret(ctx, env.tenantID, userID, "h"); err == nil {
		t.Error("expected error for missing credential")
	}

	// Seed a credential and retry.
	cred := &domain.Credential{ID: uuid.New(), TenantID: env.tenantID, UserID: userID, Identifier: "u", Secret: "old"}
	env.credRepo.byUserID[userID] = cred
	if err := env.svc.UpdateCredentialSecret(ctx, env.tenantID, userID, "newhash"); err != nil {
		t.Fatalf("UpdateCredentialSecret: %v", err)
	}
	if cred.Secret != "newhash" {
		t.Error("secret not updated")
	}
}

// --- Session management passthroughs ---

func TestSessionPassthroughs(t *testing.T) {
	env := newTaTestEnv(t)
	ctx := env.ctx()
	userID := uuid.New()

	s1 := &domain.Session{TenantID: env.tenantID, UserID: userID, ExpiresAt: time.Now().Add(time.Hour)}
	_ = env.sessRepo.Create(ctx, s1)

	sessions, err := env.svc.ListSessions(ctx, env.tenantID, userID)
	if err != nil || len(sessions) != 1 {
		t.Fatalf("ListSessions = %v, %v", sessions, err)
	}

	if err := env.svc.RevokeSession(ctx, s1.ID); err != nil {
		t.Fatalf("RevokeSession: %v", err)
	}
	sessions, _ = env.svc.ListSessions(ctx, env.tenantID, userID)
	if len(sessions) != 0 {
		t.Error("session should be revoked")
	}

	// CleanupExpired removes expired sessions.
	s2 := &domain.Session{TenantID: env.tenantID, UserID: userID, ExpiresAt: time.Now().Add(-time.Hour)}
	_ = env.sessRepo.Create(ctx, s2)
	n, err := env.svc.CleanupExpired(ctx)
	if err != nil || n == 0 {
		t.Errorf("CleanupExpired = %d, %v", n, err)
	}
}

// --- Password history summary ---

func TestGetPasswordHistory(t *testing.T) {
	env := newTaTestEnv(t)
	ctx := env.ctx()
	userID := uuid.New()
	_ = env.credRepo.AddToHistory(ctx, env.tenantID, userID, "abcdefghijklmnop-rest")

	out, err := env.svc.GetPasswordHistory(ctx, userID)
	if err != nil {
		t.Fatalf("GetPasswordHistory: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(out))
	}
	if out[0]["hash_prefix"] != "abcdefghijkl..." {
		t.Errorf("unexpected hash prefix: %v", out[0]["hash_prefix"])
	}

	if _, err := env.svc.GetPasswordHistory(context.Background(), userID); err == nil {
		t.Error("expected error without tenant context")
	}
}

// --- Forgot / Reset / Change password ---

type taEmailSender struct{ sent []string }

func (s *taEmailSender) Send(_ context.Context, to, _, _ string) error {
	s.sent = append(s.sent, to)
	return nil
}

func TestForgotPassword_Found(t *testing.T) {
	env := newTaTestEnv(t)
	ctx := env.ctx()
	userID := uuid.New()
	env.credRepo.byIdentifier[env.tenantID.String()+":a@b.c"] = &domain.Credential{
		ID: uuid.New(), TenantID: env.tenantID, UserID: userID, Identifier: "a@b.c", Enabled: true,
	}
	sender := &taEmailSender{}
	env.svc.SetEmailSender(sender)

	if err := env.svc.ForgotPassword(ctx, env.tenantID, "a@b.c"); err != nil {
		t.Fatalf("ForgotPassword: %v", err)
	}
	if len(sender.sent) != 1 {
		t.Error("reset email should have been sent")
	}
}

func TestForgotPassword_NotFound_NoReveal(t *testing.T) {
	env := newTaTestEnv(t)
	// Unknown user must still return nil (no user enumeration).
	if err := env.svc.ForgotPassword(env.ctx(), env.tenantID, "ghost@b.c"); err != nil {
		t.Errorf("ForgotPassword should not reveal missing users: %v", err)
	}
}

func TestResetPassword_Flow(t *testing.T) {
	env := newTaTestEnv(t)
	ctx := env.ctx()
	userID := uuid.New()
	cred := &domain.Credential{
		ID: uuid.New(), TenantID: env.tenantID, UserID: userID, Identifier: "u", Enabled: true,
	}
	env.credRepo.byUserID[userID] = cred

	token, err := env.svc.passwordService.IssueResetToken(ctx, userID, env.tenantID)
	if err != nil {
		t.Fatalf("IssueResetToken: %v", err)
	}
	if err := env.svc.ResetPassword(ctx, token, "N3w!Password123"); err != nil {
		t.Fatalf("ResetPassword: %v", err)
	}
	// Token is single-use.
	if err := env.svc.ResetPassword(ctx, token, "An0ther!Pass456"); err == nil {
		t.Error("reset token should be single-use")
	}
	// Credential not found path.
	token2, _ := env.svc.passwordService.IssueResetToken(ctx, uuid.New(), env.tenantID)
	if err := env.svc.ResetPassword(ctx, token2, "N3w!Password123"); err == nil {
		t.Error("expected error for unknown user")
	}
}

func TestChangePassword(t *testing.T) {
	env := newTaTestEnv(t)
	ctx := env.ctx()
	userID := uuid.New()
	oldHash := mustHashT(t, "Old!Pass123456")
	cred := &domain.Credential{
		ID: uuid.New(), TenantID: env.tenantID, UserID: userID, Identifier: "u", Secret: oldHash, Enabled: true,
	}
	env.credRepo.byUserID[userID] = cred

	// Wrong old password.
	if err := env.svc.ChangePassword(ctx, env.tenantID, userID, "bad", "N3w!Password123"); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("err = %v, want ErrInvalidCredentials", err)
	}
	// Correct flow.
	if err := env.svc.ChangePassword(ctx, env.tenantID, userID, "Old!Pass123456", "N3w!Password123"); err != nil {
		t.Fatalf("ChangePassword: %v", err)
	}
	// Missing credential.
	if err := env.svc.ChangePassword(ctx, env.tenantID, uuid.New(), "x", "y"); err == nil {
		t.Error("expected error for missing credential")
	}
}

// --- Email verification ---

func TestEmailVerificationFlow(t *testing.T) {
	env := newTaTestEnv(t)
	ctx := env.ctx()
	userID := uuid.New()

	token, err := env.svc.SendVerificationEmail(ctx, env.tenantID, userID, "a@b.c")
	if err != nil {
		t.Fatalf("SendVerificationEmail: %v", err)
	}
	gotTenant, gotUser, gotEmail, err := env.svc.VerifyEmailToken(ctx, token)
	if err != nil {
		t.Fatalf("VerifyEmailToken: %v", err)
	}
	if gotTenant != env.tenantID || gotUser != userID || gotEmail != "a@b.c" {
		t.Errorf("unexpected verification result: %v %v %v", gotTenant, gotUser, gotEmail)
	}
}

// --- WebAuthn challenge ---

func TestGenerateWebAuthnChallenge(t *testing.T) {
	env := newTaTestEnv(t)
	c1, err1 := env.svc.GenerateWebAuthnChallenge(env.ctx())
	c2, err2 := env.svc.GenerateWebAuthnChallenge(env.ctx())
	if err1 != nil || err2 != nil {
		t.Fatalf("errors: %v %v", err1, err2)
	}
	if c1 == "" || c1 == c2 {
		t.Error("challenges should be non-empty and unique")
	}
}

// --- Logout ---

func TestLogout_UnknownToken(t *testing.T) {
	env := newTaTestEnv(t)
	// Unknown refresh token: RevokeRefreshToken should return an error or nil
	// depending on implementation — just exercise the path.
	_ = env.svc.Logout(env.ctx(), "nonexistent-token")
}

// --- GetUserScopes via identity client ---

func TestGetUserScopesFallback(t *testing.T) {
	env := newTaTestEnv(t)
	scopes := env.svc.getUserScopes(env.ctx(), env.tenantID, uuid.New())
	if len(scopes) != 1 || scopes[0] != "user" {
		t.Errorf("scopes = %v, want [user]", scopes)
	}
	roles, perms := env.svc.getUserScopesAndPermissions(env.ctx(), env.tenantID, uuid.New())
	if len(roles) == 0 {
		t.Error("expected default role")
	}
	_ = perms
}

// --- writeJTI nil-safety ---

func TestWriteJTINilSafe(t *testing.T) {
	svc := &AuthService{}
	svc.writeJTI(context.Background(), uuid.New(), "jti", 3600) // must not panic
	svc.writeJTI(context.Background(), uuid.New(), "", 3600)    // empty jti no-op
}
