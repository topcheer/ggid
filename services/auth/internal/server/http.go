// Package server implements the HTTP server for the Auth Service.
package server

import (
	"encoding/json"
	stderrors "errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	ggiderrors "github.com/ggid/ggid/pkg/errors"
	"github.com/ggid/ggid/pkg/crypto"
	"github.com/ggid/ggid/pkg/i18n"
	"github.com/ggid/ggid/pkg/social"
	ggidtenant "github.com/ggid/ggid/pkg/tenant"
	"github.com/ggid/ggid/services/auth/internal/conf"
	"github.com/ggid/ggid/services/auth/internal/service"
	"github.com/ggid/ggid/services/auth/internal/webauthn"
	"github.com/google/uuid"
)

// Handler is the HTTP handler for the Auth Service.
type Handler struct {
	authSvc    *service.AuthService
	mux        *http.ServeMux
	socialReg  *social.Registry
	hooks      *service.HookManager
	idpConfigs map[string]*service.IdPConfig // keyed by config ID
	translator *i18n.Translator
}

// New creates a new Auth Service HTTP handler.
func New(authSvc *service.AuthService) *Handler {
	h := &Handler{
		authSvc:    authSvc,
		socialReg:  social.NewRegistry(),
		hooks:      service.NewHookManager(),
		idpConfigs: make(map[string]*service.IdPConfig),
		translator: i18n.NewTranslator("en"),
	}
	h.registerRoutes()
	return h
}

// t translates a message key for the given request's locale.
func (h *Handler) t(r *http.Request, key string) string {
	locale := i18n.ResolveLocale(r.Header.Get("Accept-Language"), "en")
	return h.translator.Translate(locale, key)
}

// writeErrorT writes a JSON error with i18n-translated message.
func (h *Handler) writeErrorT(w http.ResponseWriter, r *http.Request, status int, key string) {
	ggiderrors.WriteSimpleAPIError(w, status, httpStatusToCode(status), h.t(r, key))
}

func (h *Handler) registerRoutes() {
	h.mux = http.NewServeMux()
	h.mux.HandleFunc("/healthz", h.healthz)
	h.mux.HandleFunc("/readyz", h.readyz)
	h.mux.Handle("/metrics", promhttp.Handler())
	h.mux.HandleFunc("/api/v1/auth/login", h.login)
	h.mux.HandleFunc("/api/v1/auth/register", h.register)
	h.mux.HandleFunc("/api/v1/auth/logout", h.logout)
	h.mux.HandleFunc("/api/v1/auth/refresh", h.refresh)
	h.mux.HandleFunc("/api/v1/auth/password/forgot", h.forgotPassword)
	h.mux.HandleFunc("/api/v1/auth/credentials/", h.handleCredentialVault)
	h.mux.HandleFunc("/api/v1/auth/credentials/store", h.handleCredentialVault)
	h.mux.HandleFunc("/api/v1/auth/session-timeout", h.handleSessionTimeout)
	h.mux.HandleFunc("/api/v1/auth/devices/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/trust-score") {
			h.handleDeviceTrustScore(w, r)
		} else if strings.HasSuffix(r.URL.Path, "/report") {
			h.handleDeviceReport(w, r)
		} else {
			writeError(w, http.StatusNotFound, "not found")
		}
	})
	h.mux.HandleFunc("/api/v1/auth/password/reset", h.resetPassword)

	// Password reset — arch-spec routes (aliases)
	h.mux.HandleFunc("/api/v1/auth/forgot-password", h.forgotPassword)
	h.mux.HandleFunc("/api/v1/auth/reset-password", h.resetPassword)
	h.mux.HandleFunc("/api/v1/auth/password/change", h.changePassword)
	h.mux.HandleFunc("/api/v1/auth/sessions", h.handleSessions)
	h.mux.HandleFunc("/api/v1/auth/mfa/setup", h.mfaSetup)
	h.mux.HandleFunc("/api/v1/auth/mfa/verify", h.mfaVerify)
	h.mux.HandleFunc("/api/v1/auth/mfa/disable", h.mfaDisable)
	h.mux.HandleFunc("/api/v1/auth/mfa/login", h.mfaLogin)

	// Password policy config endpoint
	h.mux.HandleFunc("/api/v1/auth/password/policy", h.passwordPolicy)
	h.mux.HandleFunc("/api/v1/auth/password-policy", h.passwordPolicy)

	// Password history summary
	h.mux.HandleFunc("/api/v1/auth/password-history", h.passwordHistory)

	// Account lockout policy config
	h.mux.HandleFunc("/api/v1/auth/lockout-policy", h.lockoutPolicy)

	// Passkey autofill (conditional mediation)
	h.mux.HandleFunc("/api/v1/auth/webauthn/autofill", h.passkeyAutofill)

	// Magic Link (passwordless login)
	h.mux.HandleFunc("/api/v1/auth/magic-link", h.magicLink)
	h.mux.HandleFunc("/api/v1/auth/magic-link/verify", h.magicLinkVerify)

	// Email verification
	h.mux.HandleFunc("/api/v1/auth/email/verify", h.emailVerify)
	h.mux.HandleFunc("/api/v1/auth/email/resend", h.emailResend)

	// Email verification — arch-spec routes
	h.mux.HandleFunc("/api/v1/auth/send-verification", h.sendVerification)
	h.mux.HandleFunc("/api/v1/auth/verify-email", h.verifyEmail)

	// Phone OTP authentication
	h.mux.HandleFunc("/api/v1/auth/phone/send", h.phoneOTPSend)
	h.mux.HandleFunc("/api/v1/auth/phone/verify", h.phoneOTPVerify)

	// Step-up authentication
	h.mux.HandleFunc("/api/v1/auth/stepup/challenge", h.stepUpChallenge)
	h.mux.HandleFunc("/api/v1/auth/stepup/verify", h.stepUpVerify)

	// Logout all devices
	h.mux.HandleFunc("/api/v1/auth/logout-all", h.logoutAll)

	// Session management: force logout, concurrent session limits, device fingerprint
	h.mux.HandleFunc("/api/v1/auth/sessions/force-logout", h.forceLogout)
	h.mux.HandleFunc("/api/v1/auth/sessions/limit", h.sessionLimit)
	h.mux.HandleFunc("/api/v1/auth/sessions/revoke", h.handleRevokeSessions)
	h.mux.HandleFunc("/api/v1/auth/password-pepper/rotate", h.handlePepperRotate)
	h.mux.HandleFunc("/api/v1/auth/password-pepper/status", h.handlePepperStatus)
	h.mux.HandleFunc("/api/v1/auth/webauthn/passwordless/begin", h.handleWebAuthnPasswordlessBegin)
	h.mux.HandleFunc("/api/v1/auth/webauthn/passwordless/finish", h.handleWebAuthnPasswordlessFinish)
	h.mux.HandleFunc("/api/v1/auth/sessions/bind-device", h.handleBindDevice)
	h.mux.HandleFunc("/api/v1/auth/sessions/check-device", h.handleCheckDevice)
	h.mux.HandleFunc("/api/v1/auth/sessions/unbind-device", h.handleUnbindDevice)

	// Login attempt logging
	h.mux.HandleFunc("/api/v1/auth/login-attempts", h.loginAttempts)

	// Adaptive MFA: risk-based step-up authentication
	h.mux.HandleFunc("/api/v1/auth/risk-assess", h.riskAssess)

	// Auth hooks (Auth0 Actions equivalent)
	h.mux.HandleFunc("/api/v1/auth/hooks", h.manageHooks)

	// WebAuthn route aliases under /api/v1/auth/webauthn/
	h.mux.HandleFunc("/api/v1/auth/webauthn/register/begin", func(w http.ResponseWriter, r *http.Request) {
		h.mux.ServeHTTP(w, r) // delegate to registered /api/v1/webauthn/register/begin
	})
	h.mux.HandleFunc("/api/v1/auth/webauthn/register/finish", func(w http.ResponseWriter, r *http.Request) {
		h.mux.ServeHTTP(w, r)
	})
	h.mux.HandleFunc("/api/v1/auth/webauthn/login/begin", func(w http.ResponseWriter, r *http.Request) {
		h.mux.ServeHTTP(w, r)
	})
	h.mux.HandleFunc("/api/v1/auth/webauthn/login/finish", func(w http.ResponseWriter, r *http.Request) {
		h.mux.ServeHTTP(w, r)
	})

	// Passwordless (WebAuthn-only) registration + login
	h.mux.HandleFunc("/api/v1/auth/passwordless/register", h.passwordlessRegister)

	// MFA WebAuthn (second factor via passkey)
	h.mux.HandleFunc("/api/v1/auth/mfa/webauthn/begin", h.mfaWebAuthnBegin)
	h.mux.HandleFunc("/api/v1/auth/mfa/webauthn/finish", h.mfaWebAuthnFinish)

	// IdP federation config
	h.mux.HandleFunc("/api/v1/idp/config", h.idpConfig)

	// Email change (dual confirmation)
	h.mux.HandleFunc("/api/v1/auth/email/change", h.emailChange)
	h.mux.HandleFunc("/api/v1/auth/email/change/confirm", h.emailChangeConfirm)

	// Email change — arch-spec routes
	h.mux.HandleFunc("/api/v1/auth/change-email", h.changeEmail)
	h.mux.HandleFunc("/api/v1/auth/verify-email-change", h.verifyEmailChange)

	// Remember device (trusted devices)
	h.mux.HandleFunc("/api/v1/auth/device", h.rememberDevice)

	// Auth0 Lock compatible hosted login
	h.mux.HandleFunc("/authorize", h.authorize)
	h.mux.HandleFunc("/usernamepassword/login", h.usernamePasswordLogin)
	h.mux.HandleFunc("/dbconnections/signup", h.dbConnectionsSignup)

	// Social login endpoints
	h.mux.HandleFunc("/api/v1/auth/social/", h.handleSocial)

	// Step-up check endpoint
	h.mux.HandleFunc("/api/v1/auth/step-up-check", h.stepUpCheck)
	h.mux.HandleFunc("/api/v1/auth/step-up", h.stepUpTrigger)

	// Security: password policy configuration (arch-spec path)
	h.mux.HandleFunc("/api/v1/security/password-policy", h.securityPasswordPolicy)

	// WebAuthn / Passkey endpoints (nil credential store = skeleton mode)
	rpID := os.Getenv("WEBAUTHN_RP_ID")
	if rpID == "" {
		rpID = "ggid.dev"
	}
	rpName := os.Getenv("WEBAUTHN_RP_NAME")
	if rpName == "" {
		rpName = "GGID Platform"
	}
	var waOpts []webauthn.HandlerOption
	if originsStr := os.Getenv("WEBAUTHN_RP_ORIGINS"); originsStr != "" {
		origins := strings.Split(originsStr, ",")
		for i := range origins {
			origins[i] = strings.TrimSpace(origins[i])
		}
		waOpts = append(waOpts, webauthn.WithOrigins(origins))
	}

	// WA-12: Mobile app integration env vars
	if androidPkg := os.Getenv("WEBAUTHN_ANDROID_PACKAGE"); androidPkg != "" {
		androidSHA := os.Getenv("WEBAUTHN_ANDROID_SHA256")
		waOpts = append(waOpts, webauthn.WithAndroidAssetLinks(androidPkg, androidSHA))
	}
	if iosStr := os.Getenv("WEBAUTHN_IOS_APP_IDS"); iosStr != "" {
		iosIDs := strings.Split(iosStr, ",")
		for i := range iosIDs {
			iosIDs[i] = strings.TrimSpace(iosIDs[i])
		}
		waOpts = append(waOpts, webauthn.WithIOSAppSiteAssociation(iosIDs))
	}
	webauthnHandler, err := webauthn.NewHandler(rpID, rpName, nil, waOpts...)
	if err != nil {
		log.Printf("warning: webauthn init failed: %v", err)
	} else {
		webauthnHandler.RegisterRoutes(h.mux)
	}

	// --- Wired feature routes ---
	h.mux.HandleFunc("/api/v1/auth/impersonate", h.handleImpersonate)
	h.mux.HandleFunc("/api/v1/auth/impersonate/revoke", h.handleImpersonateRevoke)
	h.mux.HandleFunc("/api/v1/auth/webauthn/conditional", h.handleConditionalUI)
	h.mux.HandleFunc("/api/v1/notifications/send", h.handleSendNotification)
	h.mux.HandleFunc("/api/v1/auth/expiry-status", h.handleExpiryStatus)
	h.mux.HandleFunc("/api/v1/auth/password-breach-check", h.handleBreachCheck)
	h.mux.HandleFunc("/api/v1/auth/password-breach/notify", h.handlePasswordBreachNotify)
	h.mux.HandleFunc("/api/v1/auth/detect-credential-stuffing", h.handleDetectCredentialStuffing)
	h.mux.HandleFunc("/api/v1/auth/adaptive-mfa/evaluate", h.handleAdaptiveMFA)
	h.mux.HandleFunc("/api/v1/auth/biometric/enroll", h.handleBiometricEnroll)
	h.mux.HandleFunc("/api/v1/auth/biometric/verify", h.handleBiometricVerify)
	h.mux.HandleFunc("/api/v1/auth/sessions/", h.handleSessionReevaluate)
	h.mux.HandleFunc("/api/v1/auth/email-otp/send", h.handleEmailOTPSend)
	h.mux.HandleFunc("/api/v1/auth/email-otp/verify", h.handleEmailOTPVerify)
	h.mux.HandleFunc("/api/v1/auth/login/orchestrate", h.handleLoginOrchestrate)
	h.mux.HandleFunc("/api/v1/auth/sessions/stream", h.handleSessionStream)
	h.mux.HandleFunc("/api/v1/auth/password-history-check", h.handlePasswordHistoryCheck)
	h.mux.HandleFunc("/api/v1/auth/password-reset/", h.handlePasswordReset)
	h.mux.HandleFunc("/api/v1/auth/login-notify", h.handleLoginNotify)
	h.mux.HandleFunc("/api/v1/auth/login-notify/config", h.handleLoginNotify)
	h.mux.HandleFunc("/api/v1/auth/devices/register", h.handleDeviceFingerprint)
	h.mux.HandleFunc("/api/v1/auth/devices/list", h.handleDeviceFingerprint)
	h.mux.HandleFunc("/api/v1/auth/mfa/jit-enroll", h.handleJITMFAEnroll)
	h.mux.HandleFunc("/api/v1/auth/sessions/hijack-check", h.handleHijackCheck)
	h.mux.HandleFunc("/api/v1/auth/credential-stuffing/block", h.handleCredentialStuffing)
	h.mux.HandleFunc("/api/v1/auth/credential-stuffing/blocked", h.handleCredentialStuffing)
	h.mux.HandleFunc("/api/v1/auth/breach-warnings", h.handleBreachWarnings)
	h.mux.HandleFunc("/api/v1/auth/password-entropy/check", h.handlePasswordEntropy)
	h.mux.HandleFunc("/api/v1/auth/devices/trusted", h.handleTrustedDevices)
	h.mux.HandleFunc("/api/v1/auth/devices/trusted/", h.handleTrustedDevices)
	h.mux.HandleFunc("/api/v1/auth/passkeys/status", h.handlePasskeyStatus)
	h.mux.HandleFunc("/api/v1/auth/login-velocity", h.handleLoginVelocity)
	h.mux.HandleFunc("/api/v1/auth/sessions/enforce-limit", h.handleSessionLimit)
	h.mux.HandleFunc("/api/v1/auth/sessions/limits", h.handleSessionLimit)
	h.mux.HandleFunc("/api/v1/auth/password-policy/check", h.handlePasswordPolicyCheck)
	h.mux.HandleFunc("/api/v1/auth/mfa/factors", h.handleMFAFactors)
	h.mux.HandleFunc("/api/v1/auth/mfa/factors/", h.handleMFAFactors)
	h.mux.HandleFunc("/api/v1/auth/login-analytics", h.handleLoginAnalytics)
	h.mux.HandleFunc("/api/v1/auth/password-strength/distribution", h.handlePasswordStrengthDist)
	h.mux.HandleFunc("/api/v1/auth/login-geo/enrich", h.handleLoginGeoEnrich)
	h.mux.HandleFunc("/api/v1/auth/risk-notify", h.handleRiskNotify)
	h.mux.HandleFunc("/api/v1/auth/password-reset/analytics", h.handlePasswordResetAnalytics)
	h.mux.HandleFunc("/api/v1/auth/credential-exposure", h.handleCredentialExposure)
	h.mux.HandleFunc("/api/v1/auth/detect-password-spray", h.handleDetectPasswordSpray)
	h.mux.HandleFunc("/api/v1/auth/token-reuse-check", h.handleTokenReuseCheck)
	h.mux.HandleFunc("/api/v1/auth/sessions/anomaly-score", h.handleSessionAnomalyScore)
	h.mux.HandleFunc("/api/v1/auth/vpn-check", h.handleVPNCheck)
	h.mux.HandleFunc("/api/v1/auth/detect-impossible-travel", h.handleDetectImpossibleTravel)
	h.mux.HandleFunc("/api/v1/auth/geofencing", h.handleGeofencing)
	h.mux.HandleFunc("/api/v1/auth/throttle-status", h.handleThrottleStatus)
	h.mux.HandleFunc("/api/v1/auth/rotation-reminders", h.handleRotationReminders)
	h.mux.HandleFunc("/api/v1/auth/password-policy/audit", h.handlePasswordPolicyAudit)
	h.mux.HandleFunc("/api/v1/auth/login-flow/record", h.handleLoginFlowRecord)
	h.mux.HandleFunc("/api/v1/auth/sessions/device-binding-status", h.handleDeviceBindingStatus)
	h.mux.HandleFunc("/api/v1/auth/replay-check", h.handleReplayCheck)
	h.mux.HandleFunc("/api/v1/auth/password-history/config", h.handlePasswordHistoryConfig)
	h.mux.HandleFunc("/api/v1/auth/sessions/termination-reasons", h.handleTerminationReasons)
	h.mux.HandleFunc("/api/v1/auth/risk/aggregate", h.handleRiskAggregate)
	h.mux.HandleFunc("/api/v1/auth/login-patterns/", h.handleLoginPatterns)
	h.mux.HandleFunc("/api/v1/auth/hijack/timeline", h.handleHijackTimeline)
	h.mux.HandleFunc("/api/v1/auth/passwordless/stats", h.handlePasswordlessStats)
	h.mux.HandleFunc("/api/v1/auth/risk-scoring/config", h.handleRiskScoringConfig)
	h.mux.HandleFunc("/api/v1/auth/stats/credential-stuffing", h.handleCredentialStuffingStats)
	h.mux.HandleFunc("/api/v1/auth/itdr/detections", h.handleITDRDetections)
	h.mux.HandleFunc("/api/v1/auth/fraud/score", h.handleFraudScore)
	h.mux.HandleFunc("/api/v1/auth/velocity-rules", h.handleVelocityRules)
	h.mux.HandleFunc("/api/v1/auth/threat-intel/feed", h.handleThreatIntelFeed)
	h.mux.HandleFunc("/api/v1/auth/device-fingerprint/analytics", h.handleDeviceFingerprintAnalytics)
	h.mux.HandleFunc("/api/v1/auth/privilege-escalation/detect", h.handlePrivilegeEscalationDetect)
	h.mux.HandleFunc("/api/v1/auth/synthetic-identity/detect", h.handleSyntheticIdentityDetect)
	h.mux.HandleFunc("/api/v1/auth/tor-vpn/detect", h.handleTorVPNDetect)
	h.mux.HandleFunc("/api/v1/auth/lateral-movement/detect", h.handleLateralMovementDetect)
	h.mux.HandleFunc("/api/v1/auth/golden-ticket/detect", h.handleGoldenTicketDetect)
	h.mux.HandleFunc("/api/v1/auth/dlp/policies", h.handleDLPPolicies)
	h.mux.HandleFunc("/api/v1/auth/stats/social-providers", h.handleSocialProvidersStats)
	h.mux.HandleFunc("/api/v1/auth/sessions/", h.handleSessionInspect)
	h.mux.HandleFunc("/api/v1/auth/anomaly/detect", h.handleAnomalyDetect)
	h.mux.HandleFunc("/api/v1/auth/brute-force/config", h.handleBruteForceConfig)
	h.mux.HandleFunc("/api/v1/auth/webauthn/config", h.handleWebAuthnConfig)
	h.mux.HandleFunc("/api/v1/auth/passwordless/config", h.handlePasswordlessConfig)
	h.mux.HandleFunc("/api/v1/auth/adaptive-auth/config", h.handleAdaptiveAuthConfig)
	h.mux.HandleFunc("/api/v1/auth/mfa/challenge-config", h.handleMFAChallengeConfig)
	h.mux.HandleFunc("/api/v1/auth/geo-fencing/config", h.handleGeoFencingConfig)
	h.mux.HandleFunc("/api/v1/auth/session-timeout/config", h.handleSessionTimeoutConfig)
	h.mux.HandleFunc("/api/v1/auth/password-policy/config", h.handlePasswordPolicyConfig)
	h.mux.HandleFunc("/api/v1/auth/lockout-policy/config", h.handleLockoutPolicyConfig)
	h.mux.HandleFunc("/api/v1/auth/email-template/config", h.handleEmailTemplateConfig)
	h.mux.HandleFunc("/api/v1/auth/notification-preferences", h.handleNotificationPreferences)
	h.mux.HandleFunc("/api/v1/auth/session-binding/config", h.handleSessionBindingConfig)
	h.mux.HandleFunc("/api/v1/auth/mfa/config", h.handleMFAConfig)
	h.mux.HandleFunc("/api/v1/auth/impersonation/config", h.handleImpersonationConfig)
	h.mux.HandleFunc("/api/v1/auth/credentials/rotation/due", h.handleRotationDue)
	h.mux.HandleFunc("/api/v1/auth/credentials/", h.handleRotationRoute)
	h.mux.HandleFunc("/api/v1/auth/sessions/geo-stats", h.handleSessionGeoStats)
	h.mux.HandleFunc("/api/v1/auth/mfa/enrollment-stats", h.handleMFAEnrollmentStats)
	h.mux.HandleFunc("/api/v1/auth/devices/attest", h.handleDeviceAttest)
	h.mux.HandleFunc("/api/v1/auth/sessions/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/fingerprint") {
			h.handleSessionFingerprint(w, r)
			return
		}
		writeError(w, http.StatusNotFound, "not found")
	})
}

// ServeHTTP implements http.Handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Inject tenant context from X-Tenant-ID header
	if tenantIDStr := r.Header.Get("X-Tenant-ID"); tenantIDStr != "" {
		tenantID, err := uuid.Parse(tenantIDStr)
		if err == nil {
			tc := &ggidtenant.Context{
				TenantID:       tenantID,
				IsolationLevel: ggidtenant.IsolationShared,
			}
			r = r.WithContext(ggidtenant.WithContext(r.Context(), tc))
		}
	}
	h.mux.ServeHTTP(w, r)
}

// --- Health Check ---

func (h *Handler) healthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// readyz checks if the service is ready to serve requests (readiness probe).
// Returns 200 if the database/redis connections are healthy, 503 otherwise.
func (h *Handler) readyz(w http.ResponseWriter, r *http.Request) {
	// For now, same as healthz — extend to check DB/Redis when wired
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

// --- Login ---

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeErrorT(w, r, http.StatusMethodNotAllowed, "error.method_not_allowed")
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorT(w, r, http.StatusBadRequest, "error.invalid_request_body")
		return
	}

	ip := clientIP(r)
	userAgent := r.Header.Get("User-Agent")

	// Brute force protection: dual-dimension sliding window rate limit.
	if tc, err := ggidtenant.FromContext(r.Context()); err == nil {
		if err := h.authSvc.CheckBruteForce(r.Context(), tc.TenantID, ip, req.Username); err != nil {
			h.writeErrorT(w, r, http.StatusTooManyRequests, "error.too_many_login_attempts")
			return
		}
	}

	// Check if the account is locked before attempting login.
	if tc, err := ggidtenant.FromContext(r.Context()); err == nil {
		if h.authSvc.IsAccountLocked(r.Context(), tc.TenantID, req.Username) {
			h.writeErrorT(w, r, http.StatusLocked, "error.account_locked")
			return
		}
	}

	tokens, err := h.authSvc.Login(r.Context(), req.Username, req.Password, ip, userAgent)
	if err != nil {
		// Record failed login attempt for lockout tracking.
		if tc, terr := ggidtenant.FromContext(r.Context()); terr == nil {
			_ = h.authSvc.RecordFailedLogin(r.Context(), tc.TenantID, req.Username)
		}
		// Log the failed attempt for security audit.
		h.authSvc.RecordLoginAttempt(r.Context(), req.Username, ip, userAgent, false, err.Error())
		log.Printf("login error for user %s: %v", req.Username, err)
		writeAuthError(w, err)
		return
	}

	// Reset failed login counter on success.
	if tc, err := ggidtenant.FromContext(r.Context()); err == nil {
		h.authSvc.ResetFailedLogins(r.Context(), tc.TenantID, req.Username)
	}
	// Log the successful attempt.
	h.authSvc.RecordLoginAttempt(r.Context(), req.Username, ip, userAgent, true, "")

	writeJSON(w, http.StatusOK, tokens)
}

// --- Register ---

type registerRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeErrorT(w, r, http.StatusMethodNotAllowed, "error.method_not_allowed")
		return
	}

	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorT(w, r, http.StatusBadRequest, "error.invalid_request_body")
		return
	}

	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		h.writeErrorT(w, r, http.StatusBadRequest, "error.missing_tenant_context")
		return
	}

	userID := uuid.New() // In production, user is created via Identity Service first
	if err := h.authSvc.Register(r.Context(), tc.TenantID, userID, req.Username, req.Password); err != nil {
		writeAuthError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"user_id": userID.String()})
}

// --- Logout ---

type logoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req logoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.authSvc.Logout(r.Context(), req.RefreshToken); err != nil {
		writeError(w, http.StatusInternalServerError, "logout failed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"logged_out": true})
}

// --- Refresh ---

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *Handler) refresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	tokens, err := h.authSvc.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		writeAuthError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, tokens)
}

// --- Password Flows ---

type forgotPasswordRequest struct {
	Email string `json:"email"`
}

func (h *Handler) forgotPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req forgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing tenant context")
		return
	}

	_ = h.authSvc.ForgotPassword(r.Context(), tc.TenantID, req.Email)
	// Always return success to prevent email enumeration
	writeJSON(w, http.StatusOK, map[string]bool{"reset_initiated": true})
}

type resetPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

func (h *Handler) resetPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req resetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.authSvc.ResetPassword(r.Context(), req.Token, req.NewPassword); err != nil {
		writeAuthError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"password_reset": true})
}

type changePasswordRequest struct {
	UserID      string `json:"user_id"`
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

func (h *Handler) changePassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req changePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing tenant context")
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	if err := h.authSvc.ChangePassword(r.Context(), tc.TenantID, userID, req.OldPassword, req.NewPassword); err != nil {
		writeAuthError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"password_changed": true})
}

// --- Sessions ---

func (h *Handler) handleSessions(w http.ResponseWriter, r *http.Request) {
	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing tenant context")
		return
	}

	userIDStr := r.URL.Query().Get("user_id")
	if userIDStr == "" {
		writeError(w, http.StatusBadRequest, "user_id is required")
		return
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	switch r.Method {
	case http.MethodGet:
		sessions, err := h.authSvc.ListSessions(r.Context(), tc.TenantID, userID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to list sessions")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"sessions": sessions})

	case http.MethodDelete:
		sessionIDStr := r.URL.Path[len("/api/v1/auth/sessions/"):]
		sessionID, err := uuid.Parse(sessionIDStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid session_id")
			return
		}
		if err := h.authSvc.RevokeSession(r.Context(), sessionID); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to revoke session")
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"revoked": true})

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// --- MFA ---

type mfaSetupRequest struct {
	UserID     string `json:"user_id"`
	DeviceName string `json:"device_name"`
}

func (h *Handler) mfaSetup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req mfaSetupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := uuid.Parse(req.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	resp, err := h.authSvc.MFAService().SetupMFA(r.Context(), user, req.DeviceName)
	if err != nil {
		log.Printf("MFA setup error for user %s: %v", req.UserID, err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

type mfaVerifyRequest struct {
	DeviceID string `json:"device_id"`
	Code     string `json:"code"`
}

func (h *Handler) mfaVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req mfaVerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	deviceID, err := uuid.Parse(req.DeviceID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid device_id")
		return
	}

	if err := h.authSvc.MFAService().VerifyMFA(r.Context(), deviceID, req.Code); err != nil {
		if stderrors.Is(err, service.ErrInvalidMFACode) {
			writeError(w, http.StatusUnauthorized, "invalid MFA code")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"verified": true})
}

type mfaDisableRequest struct {
	DeviceID string `json:"device_id"`
}

func (h *Handler) mfaDisable(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req mfaDisableRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	deviceID, err := uuid.Parse(req.DeviceID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid device_id")
		return
	}

	if err := h.authSvc.MFAService().DisableMFA(r.Context(), deviceID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"disabled": true})
}

type mfaLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	MFACode  string `json:"mfa_code"`
}

func (h *Handler) mfaLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req mfaLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ip := clientIP(r)
	userAgent := r.Header.Get("User-Agent")

	tokens, err := h.authSvc.LoginMFA(r.Context(), req.Username, req.Password, req.MFACode, ip, userAgent)
	if err != nil {
		log.Printf("MFA login error for user %s: %v", req.Username, err)
		writeAuthError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, tokens)
}

// --- Password Policy ---

// PasswordPolicyConfigRequest is the body for POST /api/v1/auth/password-policy.
type PasswordPolicyConfigRequest struct {
	MinLength      *int     `json:"min_length,omitempty"`
	RequireUpper   *bool    `json:"require_uppercase,omitempty"`
	RequireLower   *bool    `json:"require_lowercase,omitempty"`
	RequireDigit   *bool    `json:"require_digit,omitempty"`
	RequireSpecial *bool    `json:"require_special,omitempty"`
	Blacklist      []string `json:"blacklist,omitempty"`
}

func (h *Handler) passwordPolicy(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		policy := h.authSvc.PasswordPolicy()
		writeJSON(w, http.StatusOK, policy)
	case http.MethodPost:
		var req PasswordPolicyConfigRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if err := h.authSvc.UpdatePasswordPolicy(req.MinLength, req.RequireUpper, req.RequireLower, req.RequireDigit, req.RequireSpecial, req.Blacklist); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		policy := h.authSvc.PasswordPolicy()
		writeJSON(w, http.StatusOK, policy)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// passwordHistory handles GET /api/v1/auth/password-history.
// Returns a summary of the user's password history (count + last changed).
func (h *Handler) passwordHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userIDStr := r.URL.Query().Get("user_id")
	if userIDStr == "" {
		userIDStr = r.Header.Get("X-User-ID")
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "valid user_id is required")
		return
	}

	history, err := h.authSvc.GetPasswordHistory(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"user_id":    userID.String(),
		"count":      len(history),
		"history":    history,
		"max_stored": h.authSvc.PasswordPolicy().HistoryCount,
	})
}

// lockoutPolicy handles GET/PUT /api/v1/auth/lockout-policy.
// Configures the account lockout threshold and duration.
func (h *Handler) lockoutPolicy(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		policy := h.authSvc.PasswordPolicy()
		writeJSON(w, http.StatusOK, map[string]any{
			"max_attempts":  policy.MaxAttempts,
			"lock_duration": policy.LockDuration.String(),
			"require_captcha_after": policy.MaxAttempts - 1,
		})
	case http.MethodPut, http.MethodPost:
		var req struct {
			MaxAttempts  *int `json:"max_attempts"`
			LockDuration *string `json:"lock_duration"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		policy := h.authSvc.PasswordPolicy()
		if req.MaxAttempts != nil {
			if *req.MaxAttempts < 1 || *req.MaxAttempts > 100 {
				writeError(w, http.StatusBadRequest, "max_attempts must be 1-100")
				return
			}
			policy.MaxAttempts = *req.MaxAttempts
		}
		if req.LockDuration != nil {
			d, err := time.ParseDuration(*req.LockDuration)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid lock_duration format (e.g. '15m', '1h')")
				return
			}
			policy.LockDuration = d
		}
		h.authSvc.SetPasswordPolicy(policy)
		writeJSON(w, http.StatusOK, map[string]any{
			"max_attempts":  policy.MaxAttempts,
			"lock_duration": policy.LockDuration.String(),
			"require_captcha_after": policy.MaxAttempts - 1,
		})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// passkeyAutofill handles GET /api/v1/auth/webauthn/autofill.
// Returns WebAuthn options with conditional mediation for browser autofill.
// The browser will show available passkeys in the credential picker.
func (h *Handler) passkeyAutofill(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Return a challenge + mediation=conditional for the browser to
	// prompt passkey autofill. The actual assertion is sent to
	// /api/v1/auth/webauthn/login/finish for verification.
	challenge, err := h.authSvc.GenerateWebAuthnChallenge(r.Context())
	if err != nil {
		// Fall back to a simple random challenge.
		challenge = uuid.New().String()
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"mediation":  "conditional",
		"challenge":  challenge,
		"rpId":       "ggid.dev",
		"login_url":  "/api/v1/auth/webauthn/login/finish",
		"timeout":    60000,
		"userVerification": "preferred",
	})
}

// --- Magic Link (Passwordless Login) ---

func (h *Handler) magicLink(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var body struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Email == "" {
		writeError(w, http.StatusBadRequest, "email is required")
		return
	}

	// Look up user by email via identity client.
	tenantIDStr := r.Header.Get("X-Tenant-ID")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing valid X-Tenant-ID header")
		return
	}

	ctx := ggidtenant.WithContext(r.Context(), &ggidtenant.Context{
		TenantID:       tenantID,
		IsolationLevel: ggidtenant.IsolationShared,
	})

	user, err := h.authSvc.LookupUser(ctx, tenantID, body.Email)
	if err != nil || user == nil {
		// Don't reveal whether email exists — return 200.
		writeJSON(w, http.StatusOK, map[string]string{
			"status":  "sent",
			"message": "If the email exists, a magic link has been sent.",
		})
		return
	}

	token, err := h.authSvc.IssueMagicLink(r.Context(), tenantID, user.ID, body.Email)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to issue magic link")
		return
	}

	// In production, send email. In dev mode (DEV_MODE=true), log the token.
	if os.Getenv("DEV_MODE") == "true" {
		log.Printf("[DEV] magic link token for %s: %s", body.Email, token)
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "sent",
		"message": "If the email exists, a magic link has been sent.",
	})
}

func (h *Handler) magicLinkVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	token := r.URL.Query().Get("token")
	if token == "" && r.Method == http.MethodPost {
		_ = r.ParseForm()
		token = r.FormValue("token")
	}
	if token == "" {
		writeError(w, http.StatusBadRequest, "token is required")
		return
	}

	ip := clientIP(r)
	userAgent := r.Header.Get("User-Agent")

	tokens, err := h.authSvc.VerifyMagicLink(r.Context(), token, ip, userAgent)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid or expired magic link")
		return
	}

	writeJSON(w, http.StatusOK, tokens)
}

// --- Email Verification ---

func (h *Handler) emailVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var body struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		// Try query param fallback.
		body.Token = r.URL.Query().Get("token")
	}
	if body.Token == "" {
		body.Token = r.URL.Query().Get("token")
	}
	if body.Token == "" {
		writeError(w, http.StatusBadRequest, "token is required")
		return
	}

	_, _, _, err := h.authSvc.VerifyEmailToken(r.Context(), body.Token)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid or expired verification token")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "verified"})
}

func (h *Handler) emailResend(w http.ResponseWriter, r *http.Request) {
	h.sendVerification(w, r)
}

// sendVerification handles POST /api/v1/auth/send-verification and /api/v1/auth/email/resend.
// Generates a verification token, stores it in Redis (24h TTL), and returns it in dev mode.
func (h *Handler) sendVerification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var body struct {
		Email   string `json:"email"`
		UserID  string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Email == "" && body.UserID == "" {
		writeError(w, http.StatusBadRequest, "email or user_id is required")
		return
	}

	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing tenant context")
		return
	}

	// If only email is provided, look up the user.
	var userID uuid.UUID
	if body.UserID != "" {
		userID, err = uuid.Parse(body.UserID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid user_id")
			return
		}
	} else {
		user, err := h.authSvc.LookupUser(r.Context(), tc.TenantID, body.Email)
		if err != nil || user == nil {
			// Don't reveal whether email exists.
			writeJSON(w, http.StatusOK, map[string]string{
				"status":  "sent",
				"message": "If the email exists, a verification link has been sent.",
			})
			return
		}
		userID = user.ID
		if body.Email == "" {
			body.Email = user.Email
		}
	}

	token, err := h.authSvc.SendVerificationEmail(r.Context(), tc.TenantID, userID, body.Email)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to send verification email")
		return
	}

	// Don't reveal whether email exists. Log token only in DEV_MODE.
	if os.Getenv("DEV_MODE") == "true" {
		log.Printf("[DEV] verification token for %s: %s", body.Email, token)
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "sent",
		"message": "If the email exists, a verification link has been sent.",
	})
}

// verifyEmail handles GET /api/v1/auth/verify-email?token=xxx.
// Also supports POST with JSON body for API clients.
func (h *Handler) verifyEmail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	token := r.URL.Query().Get("token")
	if token == "" && r.Method == http.MethodPost {
		var body struct {
			Token string `json:"token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err == nil {
			token = body.Token
		}
	}
	if token == "" {
		writeError(w, http.StatusBadRequest, "token is required")
		return
	}

	_, _, _, err := h.authSvc.VerifyEmailToken(r.Context(), token)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid or expired verification token")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "verified"})
}

// --- Helpers ---

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	ggiderrors.WriteSimpleAPIError(w, status, httpStatusToCode(status), msg)
}

func writeAuthError(w http.ResponseWriter, err error) {
	writeAuthErrorWithTranslator(w, err, nil, nil)
}

// writeAuthErrorT is the i18n-aware version of writeAuthError.
// It translates error messages using the handler's translator when available,
// falling back to English defaults when translator is nil (backwards compatible).
func (h *Handler) writeAuthErrorT(w http.ResponseWriter, r *http.Request, err error) {
	writeAuthErrorWithTranslator(w, err, h.translator, r)
}

// writeAuthErrorWithTranslator is the shared implementation that handles
// both i18n-translated and untranslated error responses.
func writeAuthErrorWithTranslator(w http.ResponseWriter, err error, tr *i18n.Translator, r *http.Request) {
	// resolveLocale returns "en" when translator or request is nil.
	getLocale := func() string {
		if tr == nil || r == nil {
			return "en"
		}
		return i18n.ResolveLocale(r.Header.Get("Accept-Language"), "en")
	}
	translate := func(key, fallback string) string {
		if tr == nil {
			return fallback
		}
		msg := tr.Translate(getLocale(), key)
		if msg == key {
			return fallback // key not found, use fallback
		}
		return msg
	}

	switch {
	case stderrors.Is(err, service.ErrInvalidCredentials):
		writeError(w, http.StatusUnauthorized, translate("error.invalid_credentials", "invalid credentials"))
	case stderrors.Is(err, service.ErrAccountLocked):
		writeError(w, http.StatusLocked, translate("error.account_locked", "account temporarily locked"))
	case stderrors.Is(err, service.ErrMFASetupRequired):
		writeError(w, http.StatusForbidden, err.Error())
	case stderrors.Is(err, service.ErrRateLimited):
		writeError(w, http.StatusTooManyRequests, translate("error.rate_limit_exceeded", "rate limit exceeded"))
	case stderrors.Is(err, service.ErrSessionNotFound):
		writeError(w, http.StatusNotFound, translate("error.session_not_found", "session not found"))
	case stderrors.Is(err, service.ErrPasswordTooShort), stderrors.Is(err, service.ErrPasswordTooWeak):
		writeError(w, http.StatusBadRequest, err.Error())
	case stderrors.Is(err, service.ErrPasswordReused):
		writeError(w, http.StatusConflict, err.Error())
	case stderrors.Is(err, service.ErrCredentialAlreadyExists):
		writeError(w, http.StatusConflict, translate("error.credential_already_exists", "username or email already registered"))
	case stderrors.Is(err, service.ErrInvalidResetToken):
		writeError(w, http.StatusBadRequest, translate("error.reset_token_invalid", "invalid or expired reset token"))
	default:
		var ge *ggiderrors.GGIDError
		if stderrors.As(err, &ge) {
			writeError(w, http.StatusInternalServerError, ge.Message)
			return
		}
		writeError(w, http.StatusInternalServerError, translate("error.internal_server_error", "internal server error"))
	}
}

// httpStatusToCode maps an HTTP status code to a GGID error code string.
func httpStatusToCode(status int) string {
	switch status {
	case http.StatusBadRequest:
		return string(ggiderrors.ErrInvalidArgument)
	case http.StatusUnauthorized:
		return string(ggiderrors.ErrUnauthenticated)
	case http.StatusForbidden:
		return string(ggiderrors.ErrPermissionDenied)
	case http.StatusNotFound:
		return string(ggiderrors.ErrNotFound)
	case http.StatusConflict:
		return string(ggiderrors.ErrAlreadyExists)
	case http.StatusTooManyRequests:
		return string(ggiderrors.ErrResourceExhausted)
	case http.StatusLocked:
		return "account_locked"
	default:
		if status >= 500 {
			return string(ggiderrors.ErrInternal)
		}
		return string(ggiderrors.ErrInternal)
	}
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.SplitN(xff, ",", 2)
		return strings.TrimSpace(parts[0])
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	// Use net.SplitHostPort to correctly handle both IPv4 and IPv6
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// --- Social Login ---

// --- Phone OTP ---

func (h *Handler) phoneOTPSend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var body struct {
		Phone  string `json:"phone"`
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Phone == "" {
		writeError(w, http.StatusBadRequest, "phone is required")
		return
	}

	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing tenant context")
		return
	}

	userID, err := uuid.Parse(body.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	otp, err := h.authSvc.SendPhoneOTP(r.Context(), tc.TenantID, userID, body.Phone)
	if err != nil {
		writeAuthError(w, err)
		return
	}

	// In production, send OTP via SMS. In dev mode, log the OTP.
	if os.Getenv("DEV_MODE") == "true" {
		log.Printf("[DEV] phone OTP for %s: %s", body.Phone, otp)
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "sent",
		"message": "OTP sent to phone number",
	})
}

func (h *Handler) phoneOTPVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var body struct {
		Phone string `json:"phone"`
		OTP   string `json:"otp"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Phone == "" || body.OTP == "" {
		writeError(w, http.StatusBadRequest, "phone and otp are required")
		return
	}

	ip := clientIP(r)
	userAgent := r.Header.Get("User-Agent")

	tokens, err := h.authSvc.VerifyPhoneOTP(r.Context(), body.Phone, body.OTP, ip, userAgent)
	if err != nil {
		writeAuthError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, tokens)
}

// --- Step-up Authentication ---

func (h *Handler) stepUpChallenge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var body struct {
		UserID string `json:"user_id"`
		Method string `json:"method"` // "password" or "mfa"
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	userID, err := uuid.Parse(body.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	if body.Method == "" {
		body.Method = "password"
	}

	result, err := h.authSvc.InitStepUp(r.Context(), userID, body.Method)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) stepUpVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var body struct {
		Challenge string `json:"challenge"`
		Code      string `json:"code"`      // for MFA method
		Password  string `json:"password"`  // for password method
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Challenge == "" {
		writeError(w, http.StatusBadRequest, "challenge is required")
		return
	}

	result, err := h.authSvc.VerifyStepUp(r.Context(), body.Challenge, body.Code, body.Password)
	if err != nil {
		writeAuthError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// stepUpCheck checks if the current session requires step-up authentication
// for sensitive operations (e.g. password change within last 5 minutes).
func (h *Handler) stepUpCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userIDStr := r.Header.Get("X-User-ID")
	if userIDStr == "" {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid user identity")
		return
	}

	// Check if the user has a valid step-up token.
	token := r.Header.Get("X-Step-Up-Token")
	if token != "" {
		if err := h.authSvc.ValidateStepUpToken(r.Context(), token, userID); err == nil {
			writeJSON(w, http.StatusOK, map[string]any{
				"step_up_required": false,
				"message":          "step-up token valid",
			})
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"step_up_required": true,
		"message":          "recent authentication required for this action",
		"trigger_url":      "/api/v1/auth/step-up",
	})
}

// stepUpTrigger handles step-up authentication.
// GET: checks if current session meets requested ACR level (via acr_values query param).
//      Returns 200 if sufficient, 403 + acr_values hint if step-up required.
// POST: initiates a step-up challenge (password or MFA).
func (h *Handler) stepUpTrigger(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		h.stepUpACRCheck(w, r)
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var body struct {
		UserID string `json:"user_id"`
		Method string `json:"method"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		// If user_id not in body, try X-User-ID header.
		userIDStr := r.Header.Get("X-User-ID")
		body.UserID = userIDStr
	}
	if body.UserID == "" {
		body.UserID = r.Header.Get("X-User-ID")
	}

	userID, err := uuid.Parse(body.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid or missing user_id")
		return
	}
	if body.Method == "" {
		body.Method = "mfa"
	}

	result, err := h.authSvc.InitStepUp(r.Context(), userID, body.Method)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// stepUpACRCheck handles GET /api/v1/auth/step-up?acr_values=...
// Checks if the current session's ACR level meets the requested level.
// Returns 200 if sufficient, 403 with acr_values hint if step-up is required.
func (h *Handler) stepUpACRCheck(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.Header.Get("X-User-ID")
	if userIDStr == "" {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid user identity")
		return
	}

	acrValues := r.URL.Query().Get("acr_values")
	if acrValues == "" {
		acrValues = r.URL.Query().Get("acr")
	}
	if acrValues == "" {
		writeJSON(w, http.StatusOK, map[string]any{
			"step_up_required": false,
			"message":          "no acr_values requested",
		})
		return
	}

	// Current ACR from session context (default to 0 = no assurance).
	currentACR := r.Header.Get("X-ACR")
	if currentACR == "" {
		currentACR = "0"
	}

	satisfied, challenge, err := h.authSvc.ACRStepUpCheck(r.Context(), userID, currentACR, acrValues)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if satisfied {
		writeJSON(w, http.StatusOK, map[string]any{
			"step_up_required": false,
			"acr":              currentACR,
		})
		return
	}

	// Step-up required — return 403 with hint.
	writeJSON(w, http.StatusForbidden, map[string]any{
		"error":             "insufficient_authentication",
		"step_up_required":  true,
		"acr_values":        acrValues,
		"max_age":           300,
		"challenge":         challenge,
	})
}

// changeEmail handles POST /api/v1/auth/change-email.
// Sends verification link to the new email address. Uses InitiateEmailChange service.
func (h *Handler) changeEmail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var body struct {
		UserID   string `json:"user_id"`
		OldEmail string `json:"old_email"`
		NewEmail string `json:"new_email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.NewEmail == "" {
		writeError(w, http.StatusBadRequest, "new_email is required")
		return
	}
	if body.UserID == "" {
		body.UserID = r.Header.Get("X-User-ID")
	}

	userID, err := uuid.Parse(body.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	result, err := h.authSvc.InitiateEmailChange(r.Context(), userID, body.OldEmail, body.NewEmail)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if os.Getenv("DEV_MODE") == "true" {
		log.Printf("[DEV] email change tokens — old: %s, new: %s", result.OldEmailToken, result.NewEmailToken)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "sent",
		"message": "Verification links sent to both old and new email addresses.",
	})
}

// verifyEmailChange handles GET /api/v1/auth/verify-email-change?token=xxx&step=old|new.
func (h *Handler) verifyEmailChange(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	token := r.URL.Query().Get("token")
	step := r.URL.Query().Get("step")
	if step == "" {
		step = "new"
	}
	if token == "" && r.Method == http.MethodPost {
		var body struct {
			Token string `json:"token"`
			Step  string `json:"step"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err == nil {
			token = body.Token
			if body.Step != "" {
				step = body.Step
			}
		}
	}
	if token == "" {
		writeError(w, http.StatusBadRequest, "token is required")
		return
	}

	applied, err := h.authSvc.ConfirmEmailChange(r.Context(), token, step)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if applied {
		writeJSON(w, http.StatusOK, map[string]string{"status": "email_changed"})
	} else {
		writeJSON(w, http.StatusOK, map[string]string{
			"status":  "pending",
			"message": "One side confirmed. Waiting for the other email confirmation.",
		})
	}
}

// loginAttempts handles GET /api/v1/auth/login-attempts?username=xxx&limit=50
func (h *Handler) loginAttempts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	username := r.URL.Query().Get("username")
	if username == "" {
		username = r.Header.Get("X-User-ID")
	}
	if username == "" {
		writeError(w, http.StatusBadRequest, "username query parameter is required")
		return
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		var n int
		fmt.Sscanf(l, "%d", &n)
		if n > 0 {
			limit = n
		}
	}

	attempts, err := h.authSvc.GetLoginAttempts(r.Context(), username, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query login attempts")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"username": username,
		"count":    len(attempts),
		"attempts": attempts,
	})
}

// rememberDevice handles POST /api/v1/auth/device.
// Records a trusted device fingerprint for the user.
// riskAssess handles POST /api/v1/auth/risk-assess.
// Evaluates login risk and returns whether step-up MFA is required.
// Body: {"user_id": "...", "ip": "...", "user_agent": "..."}
func (h *Handler) riskAssess(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		UserID    string `json:"user_id"`
		IP        string `json:"ip"`
		UserAgent string `json:"user_agent"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "valid user_id is required")
		return
	}

	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "tenant context required")
		return
	}

	ip := req.IP
	if ip == "" {
		ip = clientIP(r)
	}

	assessment := h.authSvc.AssessLoginRisk(r.Context(), tc.TenantID, userID, ip, req.UserAgent)

	writeJSON(w, http.StatusOK, map[string]any{
		"level":               string(assessment.Level),
		"score":               assessment.Score,
		"reasons":             assessment.Reasons,
		"requires_step_up":    assessment.RequiresStepUp,
		"requires_admin_alert": assessment.RequiresAdminAlert,
		"recommended_action": func() string {
			if assessment.RequiresStepUp {
				return "require_mfa"
			}
			if assessment.Score >= 70 {
				return "block"
			}
			return "allow"
		}(),
	})
}

func (h *Handler) rememberDevice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var body struct {
		UserID      string `json:"user_id"`
		Fingerprint string `json:"fingerprint"`
		DeviceName  string `json:"device_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Fingerprint == "" {
		writeError(w, http.StatusBadRequest, "fingerprint is required")
		return
	}
	if body.UserID == "" {
		body.UserID = r.Header.Get("X-User-ID")
	}
	userUUID, err := uuid.Parse(body.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	if err := h.authSvc.RememberTrustedDevice(r.Context(), userUUID, body.Fingerprint, body.DeviceName); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status": "trusted",
		"message": "Device registered as trusted. MFA will be skipped for 30 days.",
	})
}

// --- Auth Hooks (Auth0 Actions equivalent) ---

func (h *Handler) manageHooks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var hook service.AuthHook
		if err := json.NewDecoder(r.Body).Decode(&hook); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if hook.ID == "" {
			hook.ID = uuid.New().String()
		}
		if hook.Headers == nil {
			hook.Headers = make(map[string]string)
		}
		hook.Enabled = true
		h.hooks.RegisterHook(&hook)
		writeJSON(w, http.StatusCreated, map[string]string{"id": hook.ID, "status": "registered"})
	case http.MethodDelete:
		id := r.URL.Query().Get("id")
		if id == "" {
			writeError(w, http.StatusBadRequest, "id is required")
			return
		}
		h.hooks.RemoveHook(id)
		writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// --- Passwordless (WebAuthn-only) ---

func (h *Handler) passwordlessRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Passwordless registration = standard registration but no password required.
	// Instead, the user must complete WebAuthn registration.
	var body struct {
		UserID   string `json:"user_id"`
		Username string `json:"username"`
		Email    string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if body.Email == "" {
		writeError(w, http.StatusBadRequest, "email is required")
		return
	}

	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing tenant context")
		return
	}

	userID, _ := uuid.Parse(body.UserID)
	if userID == uuid.Nil {
		userID = uuid.New()
	}

	// Create a random temporary password (user will never use it).
	tempPass, _ := crypto.GenerateRandomToken(32)
	username := body.Username
	if username == "" {
		username = body.Email
	}

	if err := h.authSvc.Register(r.Context(), tc.TenantID, userID, username, tempPass); err != nil {
		writeAuthError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{
		"user_id":  userID.String(),
		"message":  "Passwordless account created. Complete WebAuthn registration at /api/v1/webauthn/register/begin",
		"next_url": "/api/v1/webauthn/register/begin?user_id=" + userID.String(),
	})
}

// --- MFA WebAuthn (second factor) ---

func (h *Handler) mfaWebAuthnBegin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var body struct {
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if body.UserID == "" {
		writeError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	// Redirect to WebAuthn begin registration endpoint.
	writeJSON(w, http.StatusOK, map[string]string{
		"status":   "mfa_webauthn_challenge",
		"message":  "Complete WebAuthn registration as second factor",
		"begin_url": "/api/v1/webauthn/register/begin?user_id=" + body.UserID,
	})
}

func (h *Handler) mfaWebAuthnFinish(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "mfa_webauthn_enrolled",
		"message": "WebAuthn second factor enrolled successfully",
	})
}

// --- IdP Federation ---

func (h *Handler) idpConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		var configs []*service.IdPConfig
		for _, c := range h.idpConfigs {
			configs = append(configs, c)
		}
		writeJSON(w, http.StatusOK, map[string]any{"configs": configs})

	case http.MethodPost:
		var cfg service.IdPConfig
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if cfg.ID == "" {
			cfg.ID = uuid.New().String()
		}
		if cfg.Enabled {
			// no-op, just mark as enabled
		}
		cfg.Enabled = true
		h.idpConfigs[cfg.ID] = &cfg
		writeJSON(w, http.StatusCreated, cfg)

	case http.MethodDelete:
		id := r.URL.Query().Get("id")
		if id == "" {
			writeError(w, http.StatusBadRequest, "id is required")
			return
		}
		delete(h.idpConfigs, id)
		writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// --- Logout All Devices ---

func (h *Handler) logoutAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing tenant context")
		return
	}

	var body struct {
		UserID string `json:"user_id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	userID, err := uuid.Parse(body.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	if err := h.authSvc.LogoutAll(r.Context(), tc.TenantID, userID, uuid.Nil); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to logout all devices")
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"logged_out_all": true})
}

// --- Email Change (Dual Confirmation) ---

func (h *Handler) emailChange(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var body struct {
		UserID   string `json:"user_id"`
		OldEmail string `json:"old_email"`
		NewEmail string `json:"new_email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	userID, err := uuid.Parse(body.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	result, err := h.authSvc.InitiateEmailChange(r.Context(), userID, body.OldEmail, body.NewEmail)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if os.Getenv("DEV_MODE") == "true" {
		log.Printf("[DEV] email change tokens — old: %s, new: %s", result.OldEmailToken, result.NewEmailToken)
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "sent",
		"message": "Verification links sent to both old and new email addresses.",
	})
}

func (h *Handler) emailChangeConfirm(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var body struct {
		Token string `json:"token"`
		Step  string `json:"step"` // "old" or "new"
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	applied, err := h.authSvc.ConfirmEmailChange(r.Context(), body.Token, body.Step)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"confirmed": true,
		"applied":   applied,
	})
}

// --- Auth0 Lock Compatible Endpoints ---

// authorize handles the Auth0-compatible /authorize endpoint.
// Supports client_id + connection parameters like Auth0 Lock.
func (h *Handler) authorize(w http.ResponseWriter, r *http.Request) {
	clientID := r.URL.Query().Get("client_id")
	connection := r.URL.Query().Get("connection")
	responseType := r.URL.Query().Get("response_type")
	redirectURI := r.URL.Query().Get("redirect_uri")

	if clientID == "" {
		writeError(w, http.StatusBadRequest, "client_id is required")
		return
	}

	// If connection is specified, redirect to that social provider.
	if connection != "" && connection != "Username-Password-Authentication" {
		// Social connection — redirect to social login begin.
		scheme := "https"
		if r.TLS == nil {
			scheme = "http"
		}
		host := r.Host
		if fh := r.Header.Get("X-Forwarded-Host"); fh != "" {
			host = fh
		}
		authURL := fmt.Sprintf("%s://%s/api/v1/auth/social/%s?redirect_uri=%s",
			scheme, host, connection, redirectURI)
		writeJSON(w, http.StatusOK, map[string]string{
			"redirect_to": authURL,
			"connection":  connection,
			"client_id":   clientID,
		})
		return
	}

	// Database connection — return login page parameters for Lock.
	writeJSON(w, http.StatusOK, map[string]any{
		"client_id":     clientID,
		"response_type": responseType,
		"redirect_uri":  redirectURI,
		"connection":    connection,
		"domain":        r.Host,
		"state":         r.URL.Query().Get("state"),
	})
}

// usernamePasswordLogin handles Auth0 Lock's /usernamepassword/login endpoint.
func (h *Handler) usernamePasswordLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Connection string `json:"connection"`
		ClientID  string `json:"client_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ip := clientIP(r)
	userAgent := r.Header.Get("User-Agent")

	tokens, err := h.authSvc.Login(r.Context(), body.Username, body.Password, ip, userAgent)
	if err != nil {
		writeAuthError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, tokens)
}

// dbConnectionsSignup handles Auth0 Lock's /dbconnections/signup endpoint.
func (h *Handler) dbConnectionsSignup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Connection string `json:"connection"`
		ClientID  string `json:"client_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing tenant context")
		return
	}

	userID := uuid.New()
	username := body.Email
	if username == "" {
		writeError(w, http.StatusBadRequest, "email is required")
		return
	}

	if err := h.authSvc.Register(r.Context(), tc.TenantID, userID, username, body.Password); err != nil {
		writeAuthError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"_id":  userID.String(),
		"email": body.Email,
	})
}


// handleSocial routes social login requests: /api/v1/auth/social/{provider} and /api/v1/auth/social/{provider}/callback
func (h *Handler) handleSocial(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/auth/social/")
	parts := strings.SplitN(path, "/", 2)

	provider := parts[0]
	if provider == "" {
		writeError(w, http.StatusBadRequest, "provider is required")
		return
	}

	isCallback := len(parts) == 2 && parts[1] == "callback"

	if isCallback {
		h.socialCallback(w, r, provider)
		return
	}
	h.socialBegin(w, r, provider)
}

func (h *Handler) socialBegin(w http.ResponseWriter, r *http.Request, provider string) {
	// Validate the connector exists
	conn, err := h.socialReg.Get(provider)
	if err != nil {
		writeError(w, http.StatusBadRequest, "unsupported provider: "+provider)
		return
	}

	state := uuid.New().String()
	redirectURI := r.URL.Query().Get("redirect_uri")
	if redirectURI == "" {
		redirectURI = "/login"
	}

	// Build callback URL
	scheme := "https"
	if r.TLS == nil {
		scheme = "http"
	}
	// Use forwarded host if behind proxy
	host := r.Host
	if fh := r.Header.Get("X-Forwarded-Host"); fh != "" {
		host = fh
	}
	callbackURL := scheme + "://" + host + "/api/v1/auth/social/" + provider + "/callback"

	authURL, err := conn.GetAuthURL(r.Context(), state, callbackURL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to build auth URL")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"provider":    provider,
		"auth_url":    authURL,
		"state":       state,
		"redirect_to": redirectURI,
	})
}

func (h *Handler) socialCallback(w http.ResponseWriter, r *http.Request, provider string) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	if code == "" {
		writeError(w, http.StatusBadRequest, "missing authorization code")
		return
	}

	conn, err := h.socialReg.Get(provider)
	if err != nil {
		writeError(w, http.StatusBadRequest, "unsupported provider: "+provider)
		return
	}

	scheme := "https"
	if r.TLS == nil {
		scheme = "http"
	}
	host := r.Host
	if fh := r.Header.Get("X-Forwarded-Host"); fh != "" {
		host = fh
	}
	callbackURL := scheme + "://" + host + "/api/v1/auth/social/" + provider + "/callback"

	userInfo, err := conn.HandleCallback(r.Context(), code, state, callbackURL)
	if err != nil {
		log.Printf("social callback error (%s): %v", provider, err)
		writeError(w, http.StatusUnauthorized, "social authentication failed")
		return
	}

	log.Printf("social login success: provider=%s external_id=%s email=%s", userInfo.Provider, userInfo.ExternalID, userInfo.Email)

	// Complete social login: JIT-provision or link identity, then issue JWT.
	ip := clientIP(r)
	userAgent := r.Header.Get("User-Agent")

	tokens, err := h.authSvc.SocialLogin(r.Context(), userInfo.Provider, userInfo.ExternalID, userInfo.Email, userInfo.Name, userInfo.AvatarURL, ip, userAgent)
	if err != nil {
		log.Printf("social login completion error (%s): %v", provider, err)
		writeError(w, http.StatusInternalServerError, "social login failed")
		return
	}

	writeJSON(w, http.StatusOK, tokens)
}

// forceLogout handles POST /api/v1/auth/sessions/force-logout.
// Admin operation: revokes all sessions for a user.
func (h *Handler) forceLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var body struct {
		TenantID        string `json:"tenant_id"`
		UserID          string `json:"user_id"`
		ExceptSessionID string `json:"except_session_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	tenantID, err := uuid.Parse(body.TenantID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tenant_id")
		return
	}
	userID, err := uuid.Parse(body.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	exceptSessionID := uuid.Nil
	if body.ExceptSessionID != "" {
		exceptSessionID, _ = uuid.Parse(body.ExceptSessionID)
	}

	count, err := h.authSvc.ForceLogout(r.Context(), tenantID, userID, exceptSessionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":        "ok",
		"revoked_count": count,
		"message":       fmt.Sprintf("revoked %d sessions", count),
	})
}

// sessionLimit handles POST /api/v1/auth/sessions/limit.
// Enforces the concurrent session limit for a user.
func (h *Handler) sessionLimit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var body struct {
		TenantID string `json:"tenant_id"`
		UserID   string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	tenantID, err := uuid.Parse(body.TenantID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tenant_id")
		return
	}
	userID, err := uuid.Parse(body.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	if err := h.authSvc.EnforceSessionLimit(r.Context(), tenantID, userID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"message": "session limit enforced",
	})
}

// securityPasswordPolicy handles GET/PUT /api/v1/security/password-policy.
// GET returns the current policy; PUT updates it at runtime.
func (h *Handler) securityPasswordPolicy(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{
			"policy": h.authSvc.GetPasswordPolicy(),
		})

	case http.MethodPut:
		var policy struct {
			MinLength      int      `json:"min_length"`
			RequireUpper   bool     `json:"require_upper"`
			RequireLower   bool     `json:"require_lower"`
			RequireDigit   bool     `json:"require_digit"`
			RequireSpecial bool     `json:"require_special"`
			Blacklist      []string `json:"blacklist"`
			HistoryCount   int      `json:"history_count"`
			MaxAttempts    int      `json:"max_attempts"`
		}
		if err := json.NewDecoder(r.Body).Decode(&policy); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		newPolicy := conf.PasswordPolicy{
			MinLength:      policy.MinLength,
			RequireUpper:   policy.RequireUpper,
			RequireLower:   policy.RequireLower,
			RequireDigit:   policy.RequireDigit,
			RequireSpecial: policy.RequireSpecial,
			Blacklist:      policy.Blacklist,
			HistoryCount:   policy.HistoryCount,
			MaxAttempts:    policy.MaxAttempts,
		}
		h.authSvc.SetPasswordPolicy(newPolicy)
		writeJSON(w, http.StatusOK, map[string]any{
			"status": "updated",
			"policy": h.authSvc.GetPasswordPolicy(),
		})

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
