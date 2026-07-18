package middleware

import (
	"encoding/json"
	"net/http"
)

// GGIDOpenAPISpec is the unified OpenAPI 3.1 spec for all services.
// Generated from swag annotations + manual curation.
type GGIDOpenAPISpec struct {
	OpenAPI    string                    `json:"openapi"`
	Info       OpenAPIInfo               `json:"info"`
	Servers    []OpenAPIServer           `json:"servers"`
	Paths      map[string]OpenAPIPath    `json:"paths"`
	Components OpenAPIComponents         `json:"components"`
}


type OpenAPIServer struct {
	URL         string `json:"url"`
	Description string `json:"description"`
}

type OpenAPIPath struct {
	Get    *OpenAPIOperation `json:"get,omitempty"`
	Post   *OpenAPIOperation `json:"post,omitempty"`
	Put    *OpenAPIOperation `json:"put,omitempty"`
	Delete *OpenAPIOperation `json:"delete,omitempty"`
}

type OpenAPIOperation struct {
	Tags       []string            `json:"tags"`
	Summary    string              `json:"summary"`
	Security   []map[string][]string `json:"security,omitempty"`
	Responses  map[string]OpenAPIResponse `json:"responses"`
}

type OpenAPIResponse struct {
	Description string `json:"description"`
}

// op creates an OpenAPIOperation with tags + summary.
func op(tags []string, summary string) *OpenAPIOperation {
	return &OpenAPIOperation{
		Tags:      tags,
		Summary:   summary,
		Responses: map[string]OpenAPIResponse{
			"200": {Description: "OK"},
			"400": {Description: "Bad Request"},
			"401": {Description: "Unauthorized"},
		},
	}
}

type OpenAPIComponents struct {
	SecuritySchemes map[string]OpenAPISecurityScheme `json:"securitySchemes"`
	Schemas         map[string]SchemaRef            `json:"schemas,omitempty"`
}

type OpenAPISecurityScheme struct {
	Type         string `json:"type"`
	Scheme       string `json:"scheme,omitempty"`
	BearerFormat string `json:"bearerFormat,omitempty"`
	In           string `json:"in,omitempty"`
	Name         string `json:"name,omitempty"`
	Description  string `json:"description,omitempty"`
}

// GenerateOpenAPISpec returns the unified GGID OpenAPI spec.
func GenerateOpenAPISpec() *GGIDOpenAPISpec {
	bearer := []map[string][]string{{"bearerAuth": {}}}
	return &GGIDOpenAPISpec{
		OpenAPI: "3.1.0",
		Info: OpenAPIInfo{
			Title:       "GGID Platform API",
			Description: "Unified Identity, Access Management, and Security platform",
			Version:     "1.0.0",
		},
		Servers: []OpenAPIServer{
			{URL: "https://api.ggid.io", Description: "Production"},
			{URL: "http://localhost:8080", Description: "Local dev"},
		},
		Components: OpenAPIComponents{
			SecuritySchemes: map[string]OpenAPISecurityScheme{
				"bearerAuth": {Type: "http", Scheme: "bearer", BearerFormat: "JWT", Description: "JWT Bearer token"},
				"dpop":      {Type: "http", Scheme: "DPoP", Description: "DPoP proof"},
				"apiKey":    {Type: "apiKey", In: "header", Name: "X-API-Key", Description: "API key auth"},
				"mtls":      {Type: "mutualTLS", Description: "mTLS client cert"},
			},
			Schemas: coreSchemas(),
		},
		Paths: generatePaths(bearer),
	}
}

func generatePaths(sec []map[string][]string) map[string]OpenAPIPath {
	m := make(map[string]OpenAPIPath)
	addAuthPaths(m)
	addIdentityPaths(m)
	addOAuthPaths(m)
	addPolicyPaths(m)
	addAuditPaths(m)
	addOrgPaths(m)
	addAdminPaths(m)
	addGatewayPaths(m)
	return m
}

func addAuthPaths(m map[string]OpenAPIPath) {
	m["/api/v1/auth/access-keys"] = OpenAPIPath{Get: op([]string{"API Keys"}, "Access Keys"), Post: op([]string{"API Keys"}, "Access Keys")}
	m["/api/v1/auth/access-keys/"] = OpenAPIPath{Get: op([]string{"API Keys"}, "Access Keys"), Post: op([]string{"API Keys"}, "Access Keys")}
	m["/api/v1/auth/account-linking"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Account Linking")}
	m["/api/v1/auth/adaptive-auth/config"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Adaptive Auth Config")}
	m["/api/v1/auth/adaptive-mfa/evaluate"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Auth Adaptive Mfa Evaluate")}
	m["/api/v1/auth/anomaly/detect"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Anomaly Detect")}
	m["/api/v1/auth/api-keys"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Api Keys")}
	m["/api/v1/auth/api-keys/"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Api Keys")}
	m["/api/v1/auth/biometric/enroll"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Biometric Enroll")}
	m["/api/v1/auth/biometric/verify"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Auth Biometric Verify")}
	m["/api/v1/auth/breach-warnings"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Breach Warnings")}
	m["/api/v1/auth/break-glass/activate"] = OpenAPIPath{Post: op([]string{"Break Glass"}, "Activate break-glass access")}
	m["/api/v1/auth/break-glass/history"] = OpenAPIPath{Get: op([]string{"Break Glass"}, "Break-glass activation history")}
	m["/api/v1/auth/brute-force/config"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Brute Force Config")}
	m["/api/v1/auth/cae/log"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Cae Log")}
	m["/api/v1/auth/cae/run"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Auth Cae Run")}
	m["/api/v1/auth/cae/status"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Cae Status")}
	m["/api/v1/auth/certificates"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Certificates")}
	m["/api/v1/auth/certificates/"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Certificates")}
	m["/api/v1/auth/change-email"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Change Email")}
	m["/api/v1/auth/conditional-access/"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Conditional Access")}
	m["/api/v1/auth/conditional-access/evaluate"] = OpenAPIPath{Post: op([]string{"Conditional Access"}, "Evaluate conditions against context")}
	m["/api/v1/auth/conditional-access/policies"] = OpenAPIPath{Get: op([]string{"Conditional Access"}, "List CAP policies"), Post: op([]string{"Conditional Access"}, "Create CAP policy")}
	m["/api/v1/auth/consent"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Auth Consent")}
	m["/api/v1/auth/credential-exposure"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Credential Exposure")}
	m["/api/v1/auth/credential-stuffing/block"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Credential Stuffing Block")}
	m["/api/v1/auth/credential-stuffing/blocked"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Credential Stuffing Blocked")}
	m["/api/v1/auth/credentials/"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Credentials")}
	m["/api/v1/auth/credentials/rotation"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Credentials Rotation")}
	m["/api/v1/auth/credentials/rotation/due"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Credentials Rotation Due")}
	m["/api/v1/auth/credentials/rotation/execute"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Credentials Rotation Execute")}
	m["/api/v1/auth/credentials/store"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Credentials Store")}
	m["/api/v1/auth/delegation"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Delegation")}
	m["/api/v1/auth/delegations"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Delegations")}
	m["/api/v1/auth/delegations/"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Delegations")}
	m["/api/v1/auth/detect-credential-stuffing"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Detect Credential Stuffing")}
	m["/api/v1/auth/detect-impossible-travel"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Detect Impossible Travel")}
	m["/api/v1/auth/detect-password-spray"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Detect Password Spray")}
	m["/api/v1/auth/device"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Device")}
	m["/api/v1/auth/device-bindings"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Device Bindings")}
	m["/api/v1/auth/device-fingerprint/analytics"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Device Fingerprint Analytics")}
	m["/api/v1/auth/devices/"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Devices")}
	m["/api/v1/auth/devices/attest"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Auth Devices Attest")}
	m["/api/v1/auth/devices/list"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Devices List")}
	m["/api/v1/auth/devices/register"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Auth Devices Register")}
	m["/api/v1/auth/devices/trusted"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Devices Trusted")}
	m["/api/v1/auth/devices/trusted/"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Devices Trusted")}
	m["/api/v1/auth/dlp/policies"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Dlp Policies")}
	m["/api/v1/auth/dlp/policies/"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Dlp Policies")}
	m["/api/v1/auth/email-otp/send"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Email Otp Send")}
	m["/api/v1/auth/email-otp/verify"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Auth Email Otp Verify")}
	m["/api/v1/auth/email-template/config"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Email Template Config")}
	m["/api/v1/auth/email/change"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Email Change")}
	m["/api/v1/auth/email/change/confirm"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Email Change Confirm")}
	m["/api/v1/auth/email/resend"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Email Resend")}
	m["/api/v1/auth/email/verify"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Auth Email Verify")}
	m["/api/v1/auth/enrollment/dismiss"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Enrollment Dismiss")}
	m["/api/v1/auth/enrollment/nudge/"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Enrollment Nudge")}
	m["/api/v1/auth/expiry-status"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Expiry Status")}
	m["/api/v1/auth/forgot-password"] = OpenAPIPath{Post: op([]string{"Auth"}, "Request password reset email")}
	m["/api/v1/auth/fraud/score"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Fraud Score")}
	m["/api/v1/auth/geo-fencing/config"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Geo Fencing Config")}
	m["/api/v1/auth/geofencing"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Geofencing")}
	m["/api/v1/auth/golden-ticket/detect"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Golden Ticket Detect")}
	m["/api/v1/auth/hijack/timeline"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Hijack Timeline")}
	m["/api/v1/auth/hooks"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Hooks")}
	m["/api/v1/auth/impersonate"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Impersonate")}
	m["/api/v1/auth/impersonate/revoke"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Auth Impersonate Revoke")}
	m["/api/v1/auth/impersonation/config"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Impersonation Config")}
	m["/api/v1/auth/internal/revoke-user"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Auth Internal Revoke User")}
	m["/api/v1/auth/introspection/config"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Introspection Config")}
	m["/api/v1/auth/invalidate-sessions/"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Auth Invalidate Sessions")}
	m["/api/v1/auth/lateral-movement/detect"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Lateral Movement Detect")}
	m["/api/v1/auth/lockout-policy"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Lockout Policy")}
	m["/api/v1/auth/lockout-policy/config"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Lockout Policy Config")}
	m["/api/v1/auth/login"] = OpenAPIPath{Post: op([]string{"Auth"}, "User login — authenticate with username/password, returns JWT tokens")}
	m["/api/v1/auth/login-analytics"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Auth Login Analytics")}
	m["/api/v1/auth/login-attempts"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Auth Login Attempts")}
	m["/api/v1/auth/login-flow/record"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Auth Login Flow Record")}
	m["/api/v1/auth/login-geo/enrich"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Auth Login Geo Enrich")}
	m["/api/v1/auth/login-notify"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Auth Login Notify")}
	m["/api/v1/auth/login-notify/config"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Auth Login Notify Config")}
	m["/api/v1/auth/login-patterns/"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Auth Login Patterns")}
	m["/api/v1/auth/login-policy"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Auth Login Policy")}
	m["/api/v1/auth/login-security"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Auth Login Security")}
	m["/api/v1/auth/login-velocity"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Auth Login Velocity")}
	m["/api/v1/auth/login/orchestrate"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Auth Login Orchestrate")}
	m["/api/v1/auth/logout"] = OpenAPIPath{Post: op([]string{"Auth"}, "Logout and invalidate session")}
	m["/api/v1/auth/logout-all"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Auth Logout All")}
	m["/api/v1/auth/magic-link"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Magic Link")}
	m["/api/v1/auth/magic-link/verify"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Auth Magic Link Verify")}
	m["/api/v1/auth/me"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Me")}
	m["/api/v1/auth/method-policies"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Method Policies")}
	m["/api/v1/auth/method-policies/"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Method Policies")}
	m["/api/v1/auth/mfa/backup-codes"] = OpenAPIPath{Get: op([]string{"MFA"}, "List backup codes"), Post: op([]string{"MFA"}, "Generate new backup codes")}
	m["/api/v1/auth/mfa/backup-codes/generate"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Auth Mfa Backup Codes Generate")}
	m["/api/v1/auth/mfa/backup-codes/remaining"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Auth Mfa Backup Codes Remaining")}
	m["/api/v1/auth/mfa/backup-codes/verify"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Auth Mfa Backup Codes Verify")}
	m["/api/v1/auth/mfa/challenge-config"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Auth Mfa Challenge Config")}
	m["/api/v1/auth/mfa/config"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Auth Mfa Config")}
	m["/api/v1/auth/mfa/disable"] = OpenAPIPath{Post: op([]string{"MFA"}, "Disable MFA")}
	m["/api/v1/auth/mfa/enroll"] = OpenAPIPath{Post: op([]string{"MFA"}, "Enroll MFA (TOTP)")}
	m["/api/v1/auth/mfa/enrollment-stats"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Auth Mfa Enrollment Stats")}
	m["/api/v1/auth/mfa/factors"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Auth Mfa Factors")}
	m["/api/v1/auth/mfa/factors/"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Auth Mfa Factors")}
	m["/api/v1/auth/mfa/jit-enroll"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Auth Mfa Jit Enroll")}
	m["/api/v1/auth/mfa/login"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Auth Mfa Login")}
	m["/api/v1/auth/mfa/setup"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Auth Mfa Setup")}
	m["/api/v1/auth/mfa/status"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Auth Mfa Status")}
	m["/api/v1/auth/mfa/verify"] = OpenAPIPath{Post: op([]string{"MFA"}, "Verify MFA code")}
	m["/api/v1/auth/mfa/webauthn/begin"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Auth Mfa Webauthn Begin")}
	m["/api/v1/auth/mfa/webauthn/finish"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Auth Mfa Webauthn Finish")}
	m["/api/v1/auth/mtls/config"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Mtls Config")}
	m["/api/v1/auth/multi-hash/rehash/"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Auth Multi Hash Rehash")}
	m["/api/v1/auth/multi-hash/verify"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Auth Multi Hash Verify")}
	m["/api/v1/auth/notification-preferences"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Notification Preferences")}
	m["/api/v1/auth/notifications"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Notifications")}
	m["/api/v1/auth/passkey/"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Passkey")}
	m["/api/v1/auth/passkey/auth/begin"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Passkey Auth Begin")}
	m["/api/v1/auth/passkey/auth/finish"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Passkey Auth Finish")}
	m["/api/v1/auth/passkey/register/begin"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Auth Passkey Register Begin")}
	m["/api/v1/auth/passkey/register/finish"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Auth Passkey Register Finish")}
	m["/api/v1/auth/passkeys/status"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Passkeys Status")}
	m["/api/v1/auth/password-breach-check"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Password Breach Check")}
	m["/api/v1/auth/password-breach/notify"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Password Breach Notify")}
	m["/api/v1/auth/password-deprecation"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Password Deprecation")}
	m["/api/v1/auth/password-entropy/check"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Password Entropy Check")}
	m["/api/v1/auth/password-history"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Password History")}
	m["/api/v1/auth/password-history-check"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Password History Check")}
	m["/api/v1/auth/password-history/config"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Password History Config")}
	m["/api/v1/auth/password-pepper/rotate"] = OpenAPIPath{Put: op([]string{"Auth"}, "V1 Auth Password Pepper Rotate")}
	m["/api/v1/auth/password-pepper/status"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Password Pepper Status")}
	m["/api/v1/auth/password-policy"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Password Policy")}
	m["/api/v1/auth/password-policy/audit"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Password Policy Audit")}
	m["/api/v1/auth/password-policy/check"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Password Policy Check")}
	m["/api/v1/auth/password-policy/config"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Password Policy Config")}
	m["/api/v1/auth/password-reset/"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Password Reset")}
	m["/api/v1/auth/password-reset/analytics"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Password Reset Analytics")}
	m["/api/v1/auth/password-strength/distribution"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Password Strength Distribution")}
	m["/api/v1/auth/password/change"] = OpenAPIPath{Post: op([]string{"Auth"}, "Change password (requires current password)")}
	m["/api/v1/auth/password/forgot"] = OpenAPIPath{Post: op([]string{"Auth"}, "Request password reset (v2)")}
	m["/api/v1/auth/password/policy"] = OpenAPIPath{Get: op([]string{"Auth"}, "Get password policy requirements")}
	m["/api/v1/auth/password/reset"] = OpenAPIPath{Post: op([]string{"Auth"}, "Reset password with token")}
	m["/api/v1/auth/password/strength"] = OpenAPIPath{Post: op([]string{"Auth"}, "Evaluate password strength (zxcvbn score 0-4)")}
	m["/api/v1/auth/passwordless/config"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Passwordless Config")}
	m["/api/v1/auth/passwordless/register"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Auth Passwordless Register")}
	m["/api/v1/auth/passwordless/stats"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Passwordless Stats")}
	m["/api/v1/auth/phone/send"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Phone Send")}
	m["/api/v1/auth/phone/verify"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Auth Phone Verify")}
	m["/api/v1/auth/privilege-escalation/detect"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Privilege Escalation Detect")}
	m["/api/v1/auth/profile"] = OpenAPIPath{Get: op([]string{"Auth"}, "Get current user profile"), Put: op([]string{"Auth"}, "Update own profile")}
	m["/api/v1/auth/rate-limits"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Rate Limits")}
	m["/api/v1/auth/refresh"] = OpenAPIPath{Post: op([]string{"Auth"}, "Refresh access token using refresh token")}
	m["/api/v1/auth/register"] = OpenAPIPath{Post: op([]string{"Auth"}, "Self-service user registration")}
	m["/api/v1/auth/replay-check"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Replay Check")}
	m["/api/v1/auth/reset-password"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Reset Password")}
	m["/api/v1/auth/risk-assess"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Risk Assess")}
	m["/api/v1/auth/risk-notify"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Risk Notify")}
	m["/api/v1/auth/risk-scoring/config"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Risk Scoring Config")}
	m["/api/v1/auth/risk/aggregate"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Risk Aggregate")}
	m["/api/v1/auth/rotation-reminders"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Rotation Reminders")}
	m["/api/v1/auth/send-verification"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Send Verification")}
	m["/api/v1/auth/session-binding/config"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Session Binding Config")}
	m["/api/v1/auth/session-timeout"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Session Timeout")}
	m["/api/v1/auth/session-timeout/config"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Session Timeout Config")}
	m["/api/v1/auth/sessions"] = OpenAPIPath{Get: op([]string{"Sessions"}, "List active sessions for current user")}
	m["/api/v1/auth/sessions/"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Sessions")}
	m["/api/v1/auth/sessions/anomaly-score"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Sessions Anomaly Score")}
	m["/api/v1/auth/sessions/bind-device"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Sessions Bind Device")}
	m["/api/v1/auth/sessions/check-device"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Sessions Check Device")}
	m["/api/v1/auth/sessions/device-binding-status"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Sessions Device Binding Status")}
	m["/api/v1/auth/sessions/enforce-limit"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Sessions Enforce Limit")}
	m["/api/v1/auth/sessions/force-logout"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Auth Sessions Force Logout")}
	m["/api/v1/auth/sessions/geo-stats"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Sessions Geo Stats")}
	m["/api/v1/auth/sessions/hijack-check"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Sessions Hijack Check")}
	m["/api/v1/auth/sessions/limit"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Sessions Limit")}
	m["/api/v1/auth/sessions/limits"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Sessions Limits")}
	m["/api/v1/auth/sessions/revoke"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Auth Sessions Revoke")}
	m["/api/v1/auth/sessions/revoke-user"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Auth Sessions Revoke User")}
	m["/api/v1/auth/sessions/stream"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Sessions Stream")}
	m["/api/v1/auth/sessions/termination-reasons"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Sessions Termination Reasons")}
	m["/api/v1/auth/sessions/unbind-device"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Sessions Unbind Device")}
	m["/api/v1/auth/sessions/{id}"] = OpenAPIPath{Delete: op([]string{"Sessions"}, "Revoke a session by ID")}
	m["/api/v1/auth/social/"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Social")}
	m["/api/v1/auth/stats/credential-stuffing"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Stats Credential Stuffing")}
	m["/api/v1/auth/stats/social-providers"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Stats Social Providers")}
	m["/api/v1/auth/step-up"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Step Up")}
	m["/api/v1/auth/step-up-check"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Step Up Check")}
	m["/api/v1/auth/stepup/challenge"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Stepup Challenge")}
	m["/api/v1/auth/stepup/verify"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Auth Stepup Verify")}
	m["/api/v1/auth/synthetic-identity/detect"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Synthetic Identity Detect")}
	m["/api/v1/auth/tap"] = OpenAPIPath{Post: op([]string{"TAP"}, "Issue Temporary Access Pass")}
	m["/api/v1/auth/tap/"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Tap")}
	m["/api/v1/auth/tap/batch"] = OpenAPIPath{Post: op([]string{"TAP"}, "Batch issue TAPs")}
	m["/api/v1/auth/tap/policy"] = OpenAPIPath{Get: op([]string{"TAP"}, "Get TAP policy"), Put: op([]string{"TAP"}, "Update TAP policy")}
	m["/api/v1/auth/threat-intel/feed"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Threat Intel Feed")}
	m["/api/v1/auth/throttle-status"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Throttle Status")}
	m["/api/v1/auth/token-reuse-check"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Auth Token Reuse Check")}
	m["/api/v1/auth/tokens"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Auth Tokens")}
	m["/api/v1/auth/tor-vpn/detect"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Tor Vpn Detect")}
	m["/api/v1/auth/trust-store/cas"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Trust Store Cas")}
	m["/api/v1/auth/trust-store/cas/"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Trust Store Cas")}
	m["/api/v1/auth/trust-store/verify"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Auth Trust Store Verify")}
	m["/api/v1/auth/velocity-rules"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Velocity Rules")}
	m["/api/v1/auth/verify-email"] = OpenAPIPath{Get: op([]string{"Auth"}, "Verify email address with token")}
	m["/api/v1/auth/verify-email-change"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Auth Verify Email Change")}
	m["/api/v1/auth/vpn-check"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Vpn Check")}
	m["/api/v1/auth/webauthn/aaguid"] = OpenAPIPath{Get: op([]string{"WebAuthn"}, "List AAGUID allowlist"), Post: op([]string{"WebAuthn"}, "Add AAGUID to allowlist")}
	m["/api/v1/auth/webauthn/aaguid/"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Webauthn Aaguid")}
	m["/api/v1/auth/webauthn/autofill"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Webauthn Autofill")}
	m["/api/v1/auth/webauthn/begin"] = OpenAPIPath{Post: op([]string{"WebAuthn"}, "Begin WebAuthn registration")}
	m["/api/v1/auth/webauthn/conditional"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Webauthn Conditional")}
	m["/api/v1/auth/webauthn/config"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Webauthn Config")}
	m["/api/v1/auth/webauthn/credentials/valid-ids"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Webauthn Credentials Valid Ids")}
	m["/api/v1/auth/webauthn/finish"] = OpenAPIPath{Post: op([]string{"WebAuthn"}, "Finish WebAuthn registration")}
	m["/api/v1/auth/webauthn/login/begin"] = OpenAPIPath{Post: op([]string{"WebAuthn"}, "Begin WebAuthn login")}
	m["/api/v1/auth/webauthn/login/finish"] = OpenAPIPath{Post: op([]string{"WebAuthn"}, "Finish WebAuthn login")}
	m["/api/v1/auth/webauthn/passwordless/begin"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Webauthn Passwordless Begin")}
	m["/api/v1/auth/webauthn/passwordless/finish"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Auth Webauthn Passwordless Finish")}
	m["/api/v1/auth/webauthn/register/begin"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Auth Webauthn Register Begin")}
	m["/api/v1/auth/webauthn/register/finish"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Auth Webauthn Register Finish")}
}

func addIdentityPaths(m map[string]OpenAPIPath) {
	m["/api/v1/identity/access-review/campaigns"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Access Review Campaigns")}
	m["/api/v1/identity/account-linking/config"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Account Linking Config")}
	m["/api/v1/identity/attribute-governance"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Attribute Governance")}
	m["/api/v1/identity/branding/config"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Branding Config")}
	m["/api/v1/identity/check"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Check")}
	m["/api/v1/identity/ciam/metrics"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Ciam Metrics")}
	m["/api/v1/identity/consent/registry"] = OpenAPIPath{Post: op([]string{"Identity"}, "V1 Identity Consent Registry")}
	m["/api/v1/identity/dashboard/stats"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Dashboard Stats")}
	m["/api/v1/identity/data-governance/classifications"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Data Governance Classifications")}
	m["/api/v1/identity/data-governance/dsr"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Data Governance Dsr")}
	m["/api/v1/identity/data-governance/inventory"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Data Governance Inventory")}
	m["/api/v1/identity/deprovisioning/config"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Deprovisioning Config")}
	m["/api/v1/identity/devices/"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Devices")}
	m["/api/v1/identity/did"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Did")}
	m["/api/v1/identity/did/"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Did")}
	m["/api/v1/identity/directory-health"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Directory Health")}
	m["/api/v1/identity/directory-snapshot"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Directory Snapshot")}
	m["/api/v1/identity/directory/reconcile"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Directory Reconcile")}
	m["/api/v1/identity/dlp/events"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Dlp Events")}
	m["/api/v1/identity/dlp/heatmap"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Dlp Heatmap")}
	m["/api/v1/identity/dlp/policies"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Dlp Policies")}
	m["/api/v1/identity/dlp/policies/"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Dlp Policies")}
	m["/api/v1/identity/entitlement-review/"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Entitlement Review")}
	m["/api/v1/identity/federation/discovery-rules"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Federation Discovery Rules")}
	m["/api/v1/identity/federation/entities"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Federation Entities")}
	m["/api/v1/identity/federation/route-email"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Federation Route Email")}
	m["/api/v1/identity/federation/transform-rules"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Federation Transform Rules")}
	m["/api/v1/identity/federation/transforms"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Federation Transforms")}
	m["/api/v1/identity/federation/trust-relations"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Federation Trust Relations")}
	m["/api/v1/identity/flows"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Flows")}
	m["/api/v1/identity/gdpr/export"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Gdpr Export")}
	m["/api/v1/identity/groups"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Groups")}
	m["/api/v1/identity/groups/"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Groups")}
	m["/api/v1/identity/groups/analytics"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Groups Analytics")}
	m["/api/v1/identity/idp/failover-config"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Idp Failover Config")}
	m["/api/v1/identity/idp/metadata-import"] = OpenAPIPath{Post: op([]string{"Identity"}, "V1 Identity Idp Metadata Import")}
	m["/api/v1/identity/import-validation/config"] = OpenAPIPath{Post: op([]string{"Identity"}, "V1 Identity Import Validation Config")}
	m["/api/v1/identity/jit/dry-run"] = OpenAPIPath{Post: op([]string{"Identity"}, "V1 Identity Jit Dry Run")}
	m["/api/v1/identity/jit/mappings"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Jit Mappings")}
	m["/api/v1/identity/joiner-dashboard"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Joiner Dashboard")}
	m["/api/v1/identity/joiner-flow"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Joiner Flow")}
	m["/api/v1/identity/journeys"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Journeys")}
	m["/api/v1/identity/journeys/"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Journeys")}
	m["/api/v1/identity/ldap/sync"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Ldap Sync")}
	m["/api/v1/identity/ldap/sync-config"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Ldap Sync Config")}
	m["/api/v1/identity/ldap/sync-config/test"] = OpenAPIPath{Post: op([]string{"Identity"}, "V1 Identity Ldap Sync Config Test")}
	m["/api/v1/identity/ldap/sync-history"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Ldap Sync History")}
	m["/api/v1/identity/ldap/sync-status"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Ldap Sync Status")}
	m["/api/v1/identity/lifecycle/events"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Lifecycle Events")}
	m["/api/v1/identity/lifecycle/executions"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Lifecycle Executions")}
	m["/api/v1/identity/lifecycle/rules"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Lifecycle Rules")}
	m["/api/v1/identity/list-objects"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity List Objects")}
	m["/api/v1/identity/list-subjects"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity List Subjects")}
	m["/api/v1/identity/nhi"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Nhi")}
	m["/api/v1/identity/nhi/"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Nhi")}
	m["/api/v1/identity/nhi/orphans"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Nhi Orphans")}
	m["/api/v1/identity/nhi/risk-alerts"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Nhi Risk Alerts")}
	m["/api/v1/identity/nhi/risk/scan"] = OpenAPIPath{Post: op([]string{"Identity"}, "V1 Identity Nhi Risk Scan")}
	m["/api/v1/identity/pii/discover"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Pii Discover")}
	m["/api/v1/identity/pipl/data-inventory"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Pipl Data Inventory")}
	m["/api/v1/identity/privilege-creep"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Privilege Creep")}
	m["/api/v1/identity/privilege-creep/"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Privilege Creep")}
	m["/api/v1/identity/privileged-operations"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Privileged Operations")}
	m["/api/v1/identity/provisioning/log"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Provisioning Log")}
	m["/api/v1/identity/rebac/sync-rbac"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Rebac Sync Rbac")}
	m["/api/v1/identity/review-schedules"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Review Schedules")}
	m["/api/v1/identity/review-schedules/"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Review Schedules")}
	m["/api/v1/identity/risk-scoring/config"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Risk Scoring Config")}
	m["/api/v1/identity/role-mining"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Role Mining")}
	m["/api/v1/identity/saml/attribute-mapping"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Saml Attribute Mapping")}
	m["/api/v1/identity/saml/sp-health"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Saml Sp Health")}
	m["/api/v1/identity/scim/config"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Scim Config")}
	m["/api/v1/identity/scim/config/sync"] = OpenAPIPath{Put: op([]string{"Identity"}, "V1 Identity Scim Config Sync")}
	m["/api/v1/identity/scim/error-recovery"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Scim Error Recovery")}
	m["/api/v1/identity/scim/group-mapping"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Scim Group Mapping")}
	m["/api/v1/identity/scim/provisioning-config"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Scim Provisioning Config")}
	m["/api/v1/identity/scim/sync-health"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Scim Sync Health")}
	m["/api/v1/identity/scim/tokens"] = OpenAPIPath{Post: op([]string{"Identity"}, "V1 Identity Scim Tokens")}
	m["/api/v1/identity/scim/tokens/"] = OpenAPIPath{Post: op([]string{"Identity"}, "V1 Identity Scim Tokens")}
	m["/api/v1/identity/sd-jwt/issue"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Sd Jwt Issue")}
	m["/api/v1/identity/sd-jwt/verify"] = OpenAPIPath{Post: op([]string{"Identity"}, "V1 Identity Sd Jwt Verify")}
	m["/api/v1/identity/secret-broker/active"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Secret Broker Active")}
	m["/api/v1/identity/secret-broker/broker"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Secret Broker Broker")}
	m["/api/v1/identity/secret-broker/revoke"] = OpenAPIPath{Post: op([]string{"Identity"}, "V1 Identity Secret Broker Revoke")}
	m["/api/v1/identity/secret-broker/targets"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Secret Broker Targets")}
	m["/api/v1/identity/secret-broker/targets/"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Secret Broker Targets")}
	m["/api/v1/identity/sync-status"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Sync Status")}
	m["/api/v1/identity/tenants/branding"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Tenants Branding")}
	m["/api/v1/identity/tenants/rate-limits"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Tenants Rate Limits")}
	m["/api/v1/identity/tenants/rate-limits/"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Tenants Rate Limits")}
	m["/api/v1/identity/tenants/self-register"] = OpenAPIPath{Post: op([]string{"Identity"}, "V1 Identity Tenants Self Register")}
	m["/api/v1/identity/tuples"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Tuples")}
	m["/api/v1/identity/user-lifecycle/config"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity User Lifecycle Config")}
	m["/api/v1/identity/user-lifecycle/stages"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity User Lifecycle Stages")}
	m["/api/v1/identity/users"] = OpenAPIPath{Get: op([]string{"Identity"}, "List users (legacy alias)"), Post: op([]string{"Identity"}, "Create user (legacy alias)")}
	m["/api/v1/policy/authorize"] = OpenAPIPath{Post: op([]string{"Policy"}, "Unified PDP authorize")}
	m["/api/v1/identity/users/bulk-import"] = OpenAPIPath{Post: op([]string{"Identity"}, "V1 Identity Users Bulk Import")}
	m["/api/v1/identity/users/import-async"] = OpenAPIPath{Post: op([]string{"Identity"}, "V1 Identity Users Import Async")}
	m["/api/v1/identity/users/import-async/"] = OpenAPIPath{Post: op([]string{"Identity"}, "V1 Identity Users Import Async")}
	m["/api/v1/identity/users/import-async/create"] = OpenAPIPath{Post: op([]string{"Identity"}, "V1 Identity Users Import Async Create")}
	m["/api/v1/identity/vc"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Vc")}
	m["/api/v1/identity/vc/"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Vc")}
	m["/api/v1/identity/vc/issue"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Vc Issue")}
	m["/api/v1/identity/vc/present"] = OpenAPIPath{Get: op([]string{"Identity"}, "V1 Identity Vc Present")}
	m["/api/v1/identity/vc/verify"] = OpenAPIPath{Post: op([]string{"Identity"}, "V1 Identity Vc Verify")}
	m["/api/v1/users"] = OpenAPIPath{Get: op([]string{"Identity"}, "List users"), Post: op([]string{"Identity"}, "Create user")}
	m["/api/v1/users/export"] = OpenAPIPath{Get: op([]string{"Identity"}, "Export users to CSV")}
	m["/api/v1/users/import"] = OpenAPIPath{Post: op([]string{"Identity"}, "Import users from CSV")}
	m["/api/v1/users/{id}"] = OpenAPIPath{Get: op([]string{"Identity"}, "Get user by ID"), Put: op([]string{"Identity"}, "Update user"), Delete: op([]string{"Identity"}, "Delete user")}
}

func addOAuthPaths(m map[string]OpenAPIPath) {
	m["/.well-known/jwks.json"] = OpenAPIPath{Get: op([]string{"OAuth"}, "JWKS — public keys for JWT verification")}
	m["/.well-known/openid-configuration"] = OpenAPIPath{Get: op([]string{"OAuth"}, "OIDC discovery document")}
	m["/api/v1/oauth/.well-known/openid-configuration"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Oauth .Well Known Openid Configuration")}
	m["/api/v1/oauth/agents/"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Oauth Agents")}
	m["/api/v1/oauth/analytics/summary"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Oauth Analytics Summary")}
	m["/api/v1/oauth/audience-mismatches"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Oauth Audience Mismatches")}
	m["/api/v1/oauth/authorize"] = OpenAPIPath{Get: op([]string{"OAuth"}, "OAuth 2.0 authorize endpoint"), Post: op([]string{"OAuth"}, "OAuth 2.0 authorize (POST)")}
	m["/api/v1/oauth/backchannel"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Oauth Backchannel")}
	m["/api/v1/oauth/backchannel-logout"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Oauth Backchannel Logout")}
	m["/api/v1/oauth/ciba/config"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Oauth Ciba Config")}
	m["/api/v1/oauth/client-cert"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Oauth Client Cert")}
	m["/api/v1/oauth/client-events"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Oauth Client Events")}
	m["/api/v1/oauth/client-lifecycle/config"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Oauth Client Lifecycle Config")}
	m["/api/v1/oauth/client-rate-limits"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Oauth Client Rate Limits")}
	m["/api/v1/oauth/clients"] = OpenAPIPath{Get: op([]string{"OAuth"}, "List OAuth clients"), Post: op([]string{"OAuth"}, "Create OAuth client")}
	m["/api/v1/oauth/clients/"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Oauth Clients")}
	m["/api/v1/oauth/clients/dependency-graph"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Oauth Clients Dependency Graph")}
	m["/api/v1/oauth/clients/onboarding"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Oauth Clients Onboarding")}
	m["/api/v1/oauth/clients/{id}"] = OpenAPIPath{Get: op([]string{"OAuth"}, "Get OAuth client"), Put: op([]string{"OAuth"}, "Update OAuth client"), Delete: op([]string{"OAuth"}, "Delete OAuth client")}
	m["/api/v1/oauth/consent/"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Oauth Consent")}
	m["/api/v1/oauth/consent/admin-override"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Oauth Consent Admin Override")}
	m["/api/v1/oauth/consent/analytics"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Oauth Consent Analytics")}
	m["/api/v1/oauth/consent/config"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Oauth Consent Config")}
	m["/api/v1/oauth/consent/list"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Oauth Consent List")}
	m["/api/v1/oauth/consents/dashboard"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Oauth Consents Dashboard")}
	m["/api/v1/oauth/consents/history"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Oauth Consents History")}
	m["/api/v1/oauth/device"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Oauth Device")}
	m["/api/v1/oauth/device/approve"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Oauth Device Approve")}
	m["/api/v1/oauth/device_authorization"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Oauth Device Authorization")}
	m["/api/v1/oauth/dpop/config"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Oauth Dpop Config")}
	m["/api/v1/oauth/dpop/verify"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Oauth Dpop Verify")}
	m["/api/v1/oauth/dynamic-registration/config"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Oauth Dynamic Registration Config")}
	m["/api/v1/oauth/fapi-config"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Oauth Fapi Config")}
	m["/api/v1/oauth/frontchannel-logout"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Oauth Frontchannel Logout")}
	m["/api/v1/oauth/grant-flows"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Oauth Grant Flows")}
	m["/api/v1/oauth/introspect"] = OpenAPIPath{Post: op([]string{"OAuth"}, "Token introspection (RFC 7662)")}
	m["/api/v1/oauth/introspect/batch"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Oauth Introspect Batch")}
	m["/api/v1/oauth/introspection/config"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Oauth Introspection Config")}
	m["/api/v1/oauth/introspection/stats"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Oauth Introspection Stats")}
	m["/api/v1/oauth/issuer/metadata"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Oauth Issuer Metadata")}
	m["/api/v1/oauth/jar/config"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Oauth Jar Config")}
	m["/api/v1/oauth/jwks"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Oauth Jwks")}
	m["/api/v1/oauth/jwks/rotate"] = OpenAPIPath{Put: op([]string{"Auth"}, "V1 Oauth Jwks Rotate")}
	m["/api/v1/oauth/jwks/rotation-status"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Oauth Jwks Rotation Status")}
	m["/api/v1/oauth/oidc-federation/config"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Oauth Oidc Federation Config")}
	m["/api/v1/oauth/oidc/claim-mapping"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Oauth Oidc Claim Mapping")}
	m["/api/v1/oauth/onboarding-checklist"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Oauth Onboarding Checklist")}
	m["/api/v1/oauth/par"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Oauth Par")}
	m["/api/v1/oauth/par/"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Oauth Par")}
	m["/api/v1/oauth/par/config"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Oauth Par Config")}
	m["/api/v1/oauth/rar/consent-preview"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Oauth Rar Consent Preview")}
	m["/api/v1/oauth/redirect-uri-validation/config"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Oauth Redirect Uri Validation Config")}
	m["/api/v1/oauth/register"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Oauth Register")}
	m["/api/v1/oauth/resource-allowed"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Oauth Resource Allowed")}
	m["/api/v1/oauth/resource-indicator"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Oauth Resource Indicator")}
	m["/api/v1/oauth/revoke"] = OpenAPIPath{Post: op([]string{"OAuth"}, "Revoke token (RFC 7009)")}
	m["/api/v1/oauth/revoke-cascade"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Oauth Revoke Cascade")}
	m["/api/v1/oauth/rotation-policy"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Oauth Rotation Policy")}
	m["/api/v1/oauth/scope-delegation"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Oauth Scope Delegation")}
	m["/api/v1/oauth/scope-drift"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Oauth Scope Drift")}
	m["/api/v1/oauth/scope-lifecycle"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Oauth Scope Lifecycle")}
	m["/api/v1/oauth/scopes"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Oauth Scopes")}
	m["/api/v1/oauth/scopes/"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Oauth Scopes")}
	m["/api/v1/oauth/scopes/deprecations"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Oauth Scopes Deprecations")}
	m["/api/v1/oauth/scopes/hierarchy"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Oauth Scopes Hierarchy")}
	m["/api/v1/oauth/scopes/resolve-dependencies"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Oauth Scopes Resolve Dependencies")}
	m["/api/v1/oauth/secret-compare"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Oauth Secret Compare")}
	m["/api/v1/oauth/secret-history"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Oauth Secret History")}
	m["/api/v1/oauth/stats/authorize-flow"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Oauth Stats Authorize Flow")}
	m["/api/v1/oauth/stats/backchannel-logout"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Oauth Stats Backchannel Logout")}
	m["/api/v1/oauth/stats/grant-types"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Oauth Stats Grant Types")}
	m["/api/v1/oauth/stats/oauth-2-1-audit"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Oauth Stats Oauth 2 1 Audit")}
	m["/api/v1/oauth/stats/token-binding"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Oauth Stats Token Binding")}
	m["/api/v1/oauth/stats/token-revocation"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Oauth Stats Token Revocation")}
	m["/api/v1/oauth/token"] = OpenAPIPath{Post: op([]string{"OAuth"}, "Issue token (client_credentials, authorization_code, refresh_token)")}
	m["/api/v1/oauth/token-entropy"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Oauth Token Entropy")}
	m["/api/v1/oauth/token-events/stream"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Oauth Token Events Stream")}
	m["/api/v1/oauth/token-exchange-delegation"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Oauth Token Exchange Delegation")}
	m["/api/v1/oauth/token-families/"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Oauth Token Families")}
	m["/api/v1/oauth/token-lifetime"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Oauth Token Lifetime")}
	m["/api/v1/oauth/token-lifetime/analytics"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Oauth Token Lifetime Analytics")}
	m["/api/v1/oauth/token-rotation/config"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Oauth Token Rotation Config")}
	m["/api/v1/oauth/token-scope-diff"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Oauth Token Scope Diff")}
	m["/api/v1/oauth/token/claims"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Oauth Token Claims")}
	m["/api/v1/oauth/token/downscope"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Oauth Token Downscope")}
	m["/api/v1/oauth/token/dpop-bind"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Oauth Token Dpop Bind")}
	m["/api/v1/oauth/token/dpop-verify"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Oauth Token Dpop Verify")}
	m["/api/v1/oauth/tokens/validate-audience"] = OpenAPIPath{Post: op([]string{"Auth"}, "V1 Oauth Tokens Validate Audience")}
	m["/api/v1/oauth/userinfo"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Oauth Userinfo")}
	m["/api/v1/oauth/validate-client-secret"] = OpenAPIPath{Get: op([]string{"Auth"}, "V1 Oauth Validate Client Secret")}
	m["/oauth/userinfo"] = OpenAPIPath{Get: op([]string{"OAuth"}, "Enhanced UserInfo: profile + roles + groups + permissions + risk_level")}
}

func addPolicyPaths(m map[string]OpenAPIPath) {
	m["/api/v1/roles"] = OpenAPIPath{Get: op([]string{"Policy"}, "List roles"), Post: op([]string{"Policy"}, "Create role")}
	m["/api/v1/roles/{id}"] = OpenAPIPath{Get: op([]string{"Policy"}, "Get role by ID"), Put: op([]string{"Policy"}, "Update role"), Delete: op([]string{"Policy"}, "Delete role")}
}

func addAuditPaths(m map[string]OpenAPIPath) {
	m["/api/v1/audit/access-reviews"] = OpenAPIPath{Get: op([]string{"Audit"}, "Access Reviews"), Post: op([]string{"Audit"}, "Access Reviews")}
	m["/api/v1/audit/access-reviews/pending"] = OpenAPIPath{Get: op([]string{"Audit"}, "Pending"), Post: op([]string{"Audit"}, "Pending")}
	m["/api/v1/audit/activity"] = OpenAPIPath{Get: op([]string{"Audit"}, "Activity"), Post: op([]string{"Audit"}, "Activity")}
	m["/api/v1/audit/aggregations"] = OpenAPIPath{Get: op([]string{"Audit"}, "Aggregations"), Post: op([]string{"Audit"}, "Aggregations")}
	m["/api/v1/audit/aggregations/daily"] = OpenAPIPath{Get: op([]string{"Audit"}, "Daily"), Post: op([]string{"Audit"}, "Daily")}
	m["/api/v1/audit/alert-evaluation/config"] = OpenAPIPath{Get: op([]string{"Audit"}, "Config"), Post: op([]string{"Audit"}, "Config")}
	m["/api/v1/audit/alert-webhooks"] = OpenAPIPath{Get: op([]string{"Audit"}, "Alert Webhooks"), Post: op([]string{"Audit"}, "Alert Webhooks")}
	m["/api/v1/audit/alerts/config"] = OpenAPIPath{Get: op([]string{"Audit"}, "Config"), Post: op([]string{"Audit"}, "Config")}
	m["/api/v1/audit/alerts/evaluate"] = OpenAPIPath{Get: op([]string{"Audit"}, "Evaluate"), Post: op([]string{"Audit"}, "Evaluate")}
	m["/api/v1/audit/alerts/test"] = OpenAPIPath{Get: op([]string{"Audit"}, "Test"), Post: op([]string{"Audit"}, "Test")}
	m["/api/v1/audit/anomalies/detect"] = OpenAPIPath{Get: op([]string{"Audit"}, "Detect"), Post: op([]string{"Audit"}, "Detect")}
	m["/api/v1/audit/anomaly-detection"] = OpenAPIPath{Get: op([]string{"Audit"}, "Anomaly Detection"), Post: op([]string{"Audit"}, "Anomaly Detection")}
	m["/api/v1/audit/anomaly-detection/"] = OpenAPIPath{Get: op([]string{"Audit"}, "Anomaly Detection"), Post: op([]string{"Audit"}, "Anomaly Detection")}
	m["/api/v1/audit/ccm/history"] = OpenAPIPath{Get: op([]string{"CCM"}, "Get compliance history")}
	m["/api/v1/audit/ccm/results"] = OpenAPIPath{Get: op([]string{"CCM"}, "Get compliance monitoring results")}
	m["/api/v1/audit/ccm/run"] = OpenAPIPath{Post: op([]string{"CCM"}, "Trigger compliance scan")}
	m["/api/v1/audit/ccm/summary"] = OpenAPIPath{Get: op([]string{"Audit"}, "Summary"), Post: op([]string{"Audit"}, "Summary")}
	m["/api/v1/audit/compliance-report"] = OpenAPIPath{Get: op([]string{"Audit"}, "Compliance Report"), Post: op([]string{"Audit"}, "Compliance Report")}
	m["/api/v1/audit/compliance-schedules"] = OpenAPIPath{Get: op([]string{"Audit"}, "Compliance Schedules"), Post: op([]string{"Audit"}, "Compliance Schedules")}
	m["/api/v1/audit/compliance/auto-collect"] = OpenAPIPath{Get: op([]string{"Audit"}, "Auto Collect"), Post: op([]string{"Audit"}, "Auto Collect")}
	m["/api/v1/audit/compliance/auto-score"] = OpenAPIPath{Get: op([]string{"Audit"}, "Auto Score"), Post: op([]string{"Audit"}, "Auto Score")}
	m["/api/v1/audit/compliance/cert-export"] = OpenAPIPath{Get: op([]string{"Audit"}, "Cert Export"), Post: op([]string{"Audit"}, "Cert Export")}
	m["/api/v1/audit/compliance/config"] = OpenAPIPath{Get: op([]string{"Audit"}, "Config"), Post: op([]string{"Audit"}, "Config")}
	m["/api/v1/audit/compliance/dashboard"] = OpenAPIPath{Get: op([]string{"Audit"}, "Dashboard"), Post: op([]string{"Audit"}, "Dashboard")}
	m["/api/v1/audit/compliance/drift"] = OpenAPIPath{Get: op([]string{"Audit"}, "Drift"), Post: op([]string{"Audit"}, "Drift")}
	m["/api/v1/audit/compliance/evidence"] = OpenAPIPath{Get: op([]string{"Audit"}, "Evidence"), Post: op([]string{"Audit"}, "Evidence")}
	m["/api/v1/audit/compliance/evidence-attachments"] = OpenAPIPath{Get: op([]string{"Audit"}, "Evidence Attachments"), Post: op([]string{"Audit"}, "Evidence Attachments")}
	m["/api/v1/audit/compliance/evidence-expiry"] = OpenAPIPath{Get: op([]string{"Audit"}, "Evidence Expiry"), Post: op([]string{"Audit"}, "Evidence Expiry")}
	m["/api/v1/audit/compliance/evidence-refresh"] = OpenAPIPath{Get: op([]string{"Audit"}, "Evidence Refresh"), Post: op([]string{"Audit"}, "Evidence Refresh")}
	m["/api/v1/audit/compliance/evidence/"] = OpenAPIPath{Get: op([]string{"Audit"}, "Evidence"), Post: op([]string{"Audit"}, "Evidence")}
	m["/api/v1/audit/compliance/evidence/verify-integrity"] = OpenAPIPath{Get: op([]string{"Audit"}, "Verify Integrity"), Post: op([]string{"Audit"}, "Verify Integrity")}
	m["/api/v1/audit/compliance/gaps"] = OpenAPIPath{Get: op([]string{"Audit"}, "Gaps"), Post: op([]string{"Audit"}, "Gaps")}
	m["/api/v1/audit/compliance/gaps/"] = OpenAPIPath{Get: op([]string{"Audit"}, "Gaps"), Post: op([]string{"Audit"}, "Gaps")}
	m["/api/v1/audit/compliance/heatmap"] = OpenAPIPath{Get: op([]string{"Audit"}, "Heatmap"), Post: op([]string{"Audit"}, "Heatmap")}
	m["/api/v1/audit/compliance/mapping"] = OpenAPIPath{Get: op([]string{"Audit"}, "Mapping"), Post: op([]string{"Audit"}, "Mapping")}
	m["/api/v1/audit/compliance/remediation-progress"] = OpenAPIPath{Get: op([]string{"Audit"}, "Remediation Progress"), Post: op([]string{"Audit"}, "Remediation Progress")}
	m["/api/v1/audit/compliance/schedule-collect"] = OpenAPIPath{Get: op([]string{"Audit"}, "Schedule Collect"), Post: op([]string{"Audit"}, "Schedule Collect")}
	m["/api/v1/audit/compliance/schedules"] = OpenAPIPath{Get: op([]string{"Audit"}, "Schedules"), Post: op([]string{"Audit"}, "Schedules")}
	m["/api/v1/audit/compliance/score-history"] = OpenAPIPath{Get: op([]string{"Audit"}, "Score History"), Post: op([]string{"Audit"}, "Score History")}
	m["/api/v1/audit/compliance/widget-data"] = OpenAPIPath{Get: op([]string{"Audit"}, "Widget Data"), Post: op([]string{"Audit"}, "Widget Data")}
	m["/api/v1/audit/correlate"] = OpenAPIPath{Get: op([]string{"Audit"}, "Correlate"), Post: op([]string{"Audit"}, "Correlate")}
	m["/api/v1/audit/correlation/rules"] = OpenAPIPath{Get: op([]string{"Audit"}, "Rules"), Post: op([]string{"Audit"}, "Rules")}
	m["/api/v1/audit/cross-system-correlate"] = OpenAPIPath{Get: op([]string{"Audit"}, "Cross System Correlate"), Post: op([]string{"Audit"}, "Cross System Correlate")}
	m["/api/v1/audit/dashboards/"] = OpenAPIPath{Get: op([]string{"Audit"}, "V1 Audit Dashboards")}
	m["/api/v1/audit/dsr"] = OpenAPIPath{Get: op([]string{"Audit"}, "Dsr"), Post: op([]string{"Audit"}, "Dsr")}
	m["/api/v1/audit/events"] = OpenAPIPath{Get: op([]string{"Audit"}, "List audit events with filtering")}
	m["/api/v1/audit/events/"] = OpenAPIPath{Get: op([]string{"Audit"}, "V1 Audit Events")}
	m["/api/v1/audit/events/deduplicate"] = OpenAPIPath{Get: op([]string{"Audit"}, "Deduplicate"), Post: op([]string{"Audit"}, "Deduplicate")}
	m["/api/v1/audit/events/subscribe"] = OpenAPIPath{Get: op([]string{"Audit"}, "Subscribe"), Post: op([]string{"Audit"}, "Subscribe")}
	m["/api/v1/audit/events/subscribe/"] = OpenAPIPath{Get: op([]string{"Audit"}, "Subscribe"), Post: op([]string{"Audit"}, "Subscribe")}
	m["/api/v1/audit/events/{id}"] = OpenAPIPath{Get: op([]string{"Audit"}, "Get audit event by ID")}
	m["/api/v1/audit/evidence-collection"] = OpenAPIPath{Get: op([]string{"Audit"}, "Evidence Collection"), Post: op([]string{"Audit"}, "Evidence Collection")}
	m["/api/v1/audit/evidence-collection/"] = OpenAPIPath{Get: op([]string{"Audit"}, "Evidence Collection"), Post: op([]string{"Audit"}, "Evidence Collection")}
	m["/api/v1/audit/evidence/chain"] = OpenAPIPath{Get: op([]string{"Audit"}, "Chain"), Post: op([]string{"Audit"}, "Chain")}
	m["/api/v1/audit/export"] = OpenAPIPath{Get: op([]string{"Audit"}, "Export audit events (JSON/CSV)")}
	m["/api/v1/audit/export/schedule-config"] = OpenAPIPath{Get: op([]string{"Audit"}, "Schedule Config"), Post: op([]string{"Audit"}, "Schedule Config")}
	m["/api/v1/audit/exports"] = OpenAPIPath{Get: op([]string{"Audit"}, "Exports"), Post: op([]string{"Audit"}, "Exports")}
	m["/api/v1/audit/exports/"] = OpenAPIPath{Get: op([]string{"Audit"}, "Exports"), Post: op([]string{"Audit"}, "Exports")}
	m["/api/v1/audit/exports/schedule"] = OpenAPIPath{Get: op([]string{"Audit"}, "Schedule"), Post: op([]string{"Audit"}, "Schedule")}
	m["/api/v1/audit/forensics/timeline"] = OpenAPIPath{Get: op([]string{"Audit"}, "Timeline"), Post: op([]string{"Audit"}, "Timeline")}
	m["/api/v1/audit/framework-coverage"] = OpenAPIPath{Get: op([]string{"Audit"}, "Framework Coverage"), Post: op([]string{"Audit"}, "Framework Coverage")}
	m["/api/v1/audit/gdpr-forget"] = OpenAPIPath{Get: op([]string{"Audit"}, "Gdpr Forget"), Post: op([]string{"Audit"}, "Gdpr Forget")}
	m["/api/v1/audit/gdpr-forget/"] = OpenAPIPath{Get: op([]string{"Audit"}, "Gdpr Forget"), Post: op([]string{"Audit"}, "Gdpr Forget")}
	m["/api/v1/audit/gdpr/forget"] = OpenAPIPath{Get: op([]string{"Audit"}, "Forget"), Post: op([]string{"Audit"}, "Forget")}
	m["/api/v1/audit/hash-chain"] = OpenAPIPath{Get: op([]string{"Audit"}, "Hash Chain"), Post: op([]string{"Audit"}, "Hash Chain")}
	m["/api/v1/audit/hash-chain/config"] = OpenAPIPath{Get: op([]string{"Audit"}, "Config"), Post: op([]string{"Audit"}, "Config")}
	m["/api/v1/audit/impersonation"] = OpenAPIPath{Get: op([]string{"Audit"}, "Impersonation"), Post: op([]string{"Audit"}, "Impersonation")}
	m["/api/v1/audit/incidents"] = OpenAPIPath{Get: op([]string{"Audit"}, "Incidents"), Post: op([]string{"Audit"}, "Incidents")}
	m["/api/v1/audit/incidents/active"] = OpenAPIPath{Get: op([]string{"Audit"}, "Active"), Post: op([]string{"Audit"}, "Active")}
	m["/api/v1/audit/integrity/sign-pqc"] = OpenAPIPath{Get: op([]string{"Audit"}, "Sign Pqc"), Post: op([]string{"Audit"}, "Sign Pqc")}
	m["/api/v1/audit/integrity/verify"] = OpenAPIPath{Get: op([]string{"Audit"}, "Verify"), Post: op([]string{"Audit"}, "Verify")}
	m["/api/v1/audit/integrity/verify-pqc"] = OpenAPIPath{Get: op([]string{"Audit"}, "Verify Pqc"), Post: op([]string{"Audit"}, "Verify Pqc")}
	m["/api/v1/audit/isolation-check"] = OpenAPIPath{Get: op([]string{"Audit"}, "Isolation Check"), Post: op([]string{"Audit"}, "Isolation Check")}
	m["/api/v1/audit/itdr/composite-rules"] = OpenAPIPath{Get: op([]string{"Audit"}, "Composite Rules"), Post: op([]string{"Audit"}, "Composite Rules")}
	m["/api/v1/audit/itdr/composite-rules/"] = OpenAPIPath{Get: op([]string{"Audit"}, "Composite Rules"), Post: op([]string{"Audit"}, "Composite Rules")}
	m["/api/v1/audit/itdr/detections"] = OpenAPIPath{Get: op([]string{"Audit"}, "Detections"), Post: op([]string{"Audit"}, "Detections")}
	m["/api/v1/audit/itdr/detections/"] = OpenAPIPath{Get: op([]string{"Audit"}, "Detections"), Post: op([]string{"Audit"}, "Detections")}
	m["/api/v1/audit/itdr/incidents"] = OpenAPIPath{Get: op([]string{"Audit"}, "Incidents"), Post: op([]string{"Audit"}, "Incidents")}
	m["/api/v1/audit/itdr/kill-chain/"] = OpenAPIPath{Get: op([]string{"Audit"}, "V1 Audit Itdr Kill Chain")}
	m["/api/v1/audit/itdr/playbooks"] = OpenAPIPath{Get: op([]string{"Audit"}, "Playbooks"), Post: op([]string{"Audit"}, "Playbooks")}
	m["/api/v1/audit/itdr/rules"] = OpenAPIPath{Get: op([]string{"Audit"}, "Rules"), Post: op([]string{"Audit"}, "Rules")}
	m["/api/v1/audit/itdr/rules/"] = OpenAPIPath{Get: op([]string{"Audit"}, "Rules"), Post: op([]string{"Audit"}, "Rules")}
	m["/api/v1/audit/itdr/stats"] = OpenAPIPath{Get: op([]string{"Audit"}, "Stats"), Post: op([]string{"Audit"}, "Stats")}
	m["/api/v1/audit/itdr/threat-heatmap"] = OpenAPIPath{Get: op([]string{"Audit"}, "Threat Heatmap"), Post: op([]string{"Audit"}, "Threat Heatmap")}
	m["/api/v1/audit/lineage"] = OpenAPIPath{Get: op([]string{"Audit"}, "Lineage"), Post: op([]string{"Audit"}, "Lineage")}
	m["/api/v1/audit/metrics"] = OpenAPIPath{Get: op([]string{"Audit"}, "Metrics"), Post: op([]string{"Audit"}, "Metrics")}
	m["/api/v1/audit/pii-scan"] = OpenAPIPath{Get: op([]string{"Audit"}, "Pii Scan"), Post: op([]string{"Audit"}, "Pii Scan")}
	m["/api/v1/audit/query-metrics"] = OpenAPIPath{Get: op([]string{"Audit"}, "Query Metrics"), Post: op([]string{"Audit"}, "Query Metrics")}
	m["/api/v1/audit/regulatory/report"] = OpenAPIPath{Get: op([]string{"Audit"}, "Report"), Post: op([]string{"Audit"}, "Report")}
	m["/api/v1/audit/reports"] = OpenAPIPath{Get: op([]string{"Audit"}, "Reports"), Post: op([]string{"Audit"}, "Reports")}
	m["/api/v1/audit/reports/"] = OpenAPIPath{Get: op([]string{"Audit"}, "Reports"), Post: op([]string{"Audit"}, "Reports")}
	m["/api/v1/audit/reports/custom"] = OpenAPIPath{Get: op([]string{"Audit"}, "Custom"), Post: op([]string{"Audit"}, "Custom")}
	m["/api/v1/audit/reports/generate"] = OpenAPIPath{Get: op([]string{"Audit"}, "Generate"), Post: op([]string{"Audit"}, "Generate")}
	m["/api/v1/audit/retention"] = OpenAPIPath{Get: op([]string{"Audit"}, "Retention"), Post: op([]string{"Audit"}, "Retention")}
	m["/api/v1/audit/retention-policies"] = OpenAPIPath{Get: op([]string{"Audit"}, "Retention Policies"), Post: op([]string{"Audit"}, "Retention Policies")}
	m["/api/v1/audit/retention/execute"] = OpenAPIPath{Get: op([]string{"Audit"}, "Execute"), Post: op([]string{"Audit"}, "Execute")}
	m["/api/v1/audit/retention/simulate"] = OpenAPIPath{Get: op([]string{"Audit"}, "Simulate"), Post: op([]string{"Audit"}, "Simulate")}
	m["/api/v1/audit/risk-score"] = OpenAPIPath{Get: op([]string{"Audit"}, "Risk Score"), Post: op([]string{"Audit"}, "Risk Score")}
	m["/api/v1/audit/rules"] = OpenAPIPath{Get: op([]string{"Audit"}, "Rules"), Post: op([]string{"Audit"}, "Rules")}
	m["/api/v1/audit/sbom"] = OpenAPIPath{Get: op([]string{"Audit"}, "Sbom"), Post: op([]string{"Audit"}, "Sbom")}
	m["/api/v1/audit/sbom/"] = OpenAPIPath{Get: op([]string{"Audit"}, "Sbom"), Post: op([]string{"Audit"}, "Sbom")}
	m["/api/v1/audit/search"] = OpenAPIPath{Get: op([]string{"Audit"}, "Search"), Post: op([]string{"Audit"}, "Search")}
	m["/api/v1/audit/security-posture"] = OpenAPIPath{Get: op([]string{"Audit"}, "Security Posture"), Post: op([]string{"Audit"}, "Security Posture")}
	m["/api/v1/audit/siem/forwarder-config"] = OpenAPIPath{Get: op([]string{"Audit"}, "Forwarder Config"), Post: op([]string{"Audit"}, "Forwarder Config")}
	m["/api/v1/audit/siem/health"] = OpenAPIPath{Get: op([]string{"Audit"}, "Health"), Post: op([]string{"Audit"}, "Health")}
	m["/api/v1/audit/siem/health-check"] = OpenAPIPath{Get: op([]string{"Audit"}, "Health Check"), Post: op([]string{"Audit"}, "Health Check")}
	m["/api/v1/audit/siem/metrics"] = OpenAPIPath{Get: op([]string{"Audit"}, "Metrics"), Post: op([]string{"Audit"}, "Metrics")}
	m["/api/v1/audit/stats"] = OpenAPIPath{Get: op([]string{"Audit"}, "Aggregate audit statistics")}
	m["/api/v1/audit/stream"] = OpenAPIPath{Get: op([]string{"Audit"}, "Stream"), Post: op([]string{"Audit"}, "Stream")}
	m["/api/v1/audit/tamper-check"] = OpenAPIPath{Get: op([]string{"Audit"}, "Tamper Check"), Post: op([]string{"Audit"}, "Tamper Check")}
	m["/api/v1/audit/threat-feed"] = OpenAPIPath{Get: op([]string{"Audit"}, "Threat Feed"), Post: op([]string{"Audit"}, "Threat Feed")}
	m["/api/v1/audit/threat-intel/check"] = OpenAPIPath{Get: op([]string{"Audit"}, "Check"), Post: op([]string{"Audit"}, "Check")}
	m["/api/v1/audit/threat-intel/indicators"] = OpenAPIPath{Get: op([]string{"Audit"}, "Indicators"), Post: op([]string{"Audit"}, "Indicators")}
	m["/api/v1/audit/threat-intel/sources"] = OpenAPIPath{Get: op([]string{"Audit"}, "Sources"), Post: op([]string{"Audit"}, "Sources")}
	m["/api/v1/audit/threat-intel/sources/"] = OpenAPIPath{Get: op([]string{"Audit"}, "Sources"), Post: op([]string{"Audit"}, "Sources")}
	m["/api/v1/audit/threat-intel/stats"] = OpenAPIPath{Get: op([]string{"Audit"}, "Stats"), Post: op([]string{"Audit"}, "Stats")}
	m["/api/v1/audit/timeline/reconstruct"] = OpenAPIPath{Get: op([]string{"Audit"}, "Reconstruct"), Post: op([]string{"Audit"}, "Reconstruct")}
	m["/api/v1/audit/verify-integrity"] = OpenAPIPath{Get: op([]string{"Audit"}, "Verify Integrity"), Post: op([]string{"Audit"}, "Verify Integrity")}
	m["/api/v1/audit/webhooks"] = OpenAPIPath{Get: op([]string{"Audit"}, "Webhooks"), Post: op([]string{"Audit"}, "Webhooks")}
	m["/api/v1/audit/webhooks/"] = OpenAPIPath{Get: op([]string{"Audit"}, "Webhooks"), Post: op([]string{"Audit"}, "Webhooks")}
	m["/api/v1/audit/webhooks/delivery-status"] = OpenAPIPath{Get: op([]string{"Audit"}, "Delivery Status"), Post: op([]string{"Audit"}, "Delivery Status")}
	m["/api/v1/audit/ws"] = OpenAPIPath{Get: op([]string{"Audit"}, "Ws"), Post: op([]string{"Audit"}, "Ws")}
	m["/api/v1/compliance/schedules"] = OpenAPIPath{Get: op([]string{"Platform"}, "V1 Compliance Schedules")}
}

func addOrgPaths(m map[string]OpenAPIPath) {
	m["/api/v1/org/budget-tracking"] = OpenAPIPath{Get: op([]string{"Org"}, "V1 Org Budget Tracking")}
	m["/api/v1/org/cost-centers"] = OpenAPIPath{Get: op([]string{"Other"}, "Cost Centers"), Post: op([]string{"Other"}, "Cost Centers")}
	m["/api/v1/org/department-analytics"] = OpenAPIPath{Get: op([]string{"Other"}, "Department Analytics"), Post: op([]string{"Other"}, "Department Analytics")}
	m["/api/v1/org/reporting-structure"] = OpenAPIPath{Get: op([]string{"Other"}, "Reporting Structure"), Post: op([]string{"Other"}, "Reporting Structure")}
	m["/api/v1/org/stats/membership-trends"] = OpenAPIPath{Get: op([]string{"Other"}, "Membership Trends"), Post: op([]string{"Other"}, "Membership Trends")}
	m["/api/v1/org/team-insights"] = OpenAPIPath{Get: op([]string{"Other"}, "Team Insights"), Post: op([]string{"Other"}, "Team Insights")}
	m["/api/v1/org/tenants/migrate"] = OpenAPIPath{Get: op([]string{"Other"}, "Migrate"), Post: op([]string{"Other"}, "Migrate")}
	m["/api/v1/org/vendors"] = OpenAPIPath{Get: op([]string{"Other"}, "Vendors"), Post: op([]string{"Other"}, "Vendors")}
}

func addAdminPaths(m map[string]OpenAPIPath) {
	m["/api/v1/admin/backups"] = OpenAPIPath{Get: op([]string{"Admin"}, "List backups")}
	m["/api/v1/admin/backups/"] = OpenAPIPath{Get: op([]string{"Admin"}, "V1 Admin Backups")}
	m["/api/v1/admin/backups/trigger"] = OpenAPIPath{Post: op([]string{"Admin"}, "Trigger backup")}
	m["/api/v1/admin/config"] = OpenAPIPath{Get: op([]string{"Admin"}, "Config"), Post: op([]string{"Admin"}, "Config")}
	m["/api/v1/admin/config/"] = OpenAPIPath{Get: op([]string{"Admin"}, "Config"), Post: op([]string{"Admin"}, "Config")}
	m["/api/v1/admin/email/config"] = OpenAPIPath{Get: op([]string{"Admin"}, "Config"), Post: op([]string{"Admin"}, "Config")}
	m["/api/v1/admin/email/test"] = OpenAPIPath{Get: op([]string{"Admin"}, "Test"), Post: op([]string{"Admin"}, "Test")}
	m["/api/v1/admin/feature-flags"] = OpenAPIPath{Get: op([]string{"Admin"}, "Feature Flags"), Post: op([]string{"Admin"}, "Feature Flags")}
	m["/api/v1/admin/feature-flags/"] = OpenAPIPath{Get: op([]string{"Admin"}, "Feature Flags"), Post: op([]string{"Admin"}, "Feature Flags")}
	m["/api/v1/admin/keys"] = OpenAPIPath{Get: op([]string{"Admin"}, "List active signing keys")}
	m["/api/v1/admin/keys/history"] = OpenAPIPath{Get: op([]string{"Admin"}, "History"), Post: op([]string{"Admin"}, "History")}
	m["/api/v1/admin/keys/rotate/"] = OpenAPIPath{Put: op([]string{"Admin"}, "V1 Admin Keys Rotate")}
	m["/api/v1/admin/migration/config"] = OpenAPIPath{Get: op([]string{"Admin"}, "Config"), Post: op([]string{"Admin"}, "Config")}
	m["/api/v1/admin/migration/mappings"] = OpenAPIPath{Get: op([]string{"Admin"}, "Mappings"), Post: op([]string{"Admin"}, "Mappings")}
	m["/api/v1/admin/migration/mappings/"] = OpenAPIPath{Get: op([]string{"Admin"}, "Mappings"), Post: op([]string{"Admin"}, "Mappings")}
	m["/api/v1/admin/migration/stats"] = OpenAPIPath{Get: op([]string{"Admin"}, "Stats"), Post: op([]string{"Admin"}, "Stats")}
	m["/api/v1/admin/migration/test"] = OpenAPIPath{Get: op([]string{"Admin"}, "Test"), Post: op([]string{"Admin"}, "Test")}
	m["/api/v1/admin/rls/enable/"] = OpenAPIPath{Get: op([]string{"Admin"}, "V1 Admin Rls Enable")}
	m["/api/v1/admin/rls/status"] = OpenAPIPath{Get: op([]string{"Admin"}, "Status"), Post: op([]string{"Admin"}, "Status")}
	m["/api/v1/admin/rls/test"] = OpenAPIPath{Get: op([]string{"Admin"}, "Test"), Post: op([]string{"Admin"}, "Test")}
	m["/api/v1/admin/secrets"] = OpenAPIPath{Get: op([]string{"Admin"}, "List secret references")}
	m["/api/v1/admin/secrets/health"] = OpenAPIPath{Get: op([]string{"Admin"}, "Health"), Post: op([]string{"Admin"}, "Health")}
	m["/api/v1/admin/secrets/rotate/"] = OpenAPIPath{Put: op([]string{"Admin"}, "V1 Admin Secrets Rotate")}
}

func addGatewayPaths(m map[string]OpenAPIPath) {
	m["/api/v1/.well-known/federation-configuration"] = OpenAPIPath{Get: op([]string{"Other"}, "Federation Configuration"), Post: op([]string{"Other"}, "Federation Configuration")}
	m["/api/v1/access-requests"] = OpenAPIPath{Get: op([]string{"Access"}, "Access Requests"), Post: op([]string{"Access"}, "Access Requests")}
	m["/api/v1/access-requests/"] = OpenAPIPath{Get: op([]string{"Access"}, "Access Requests"), Post: op([]string{"Access"}, "Access Requests")}
	m["/api/v1/agents"] = OpenAPIPath{Get: op([]string{"Other"}, "Agents"), Post: op([]string{"Other"}, "Agents")}
	m["/api/v1/agents/"] = OpenAPIPath{Get: op([]string{"Other"}, "Agents"), Post: op([]string{"Other"}, "Agents")}
	m["/api/v1/agents/drift/report"] = OpenAPIPath{Get: op([]string{"Other"}, "Report"), Post: op([]string{"Other"}, "Report")}
	m["/api/v1/agents/register"] = OpenAPIPath{Get: op([]string{"Other"}, "Register"), Post: op([]string{"Other"}, "Register")}
	m["/api/v1/agents/reviews"] = OpenAPIPath{Get: op([]string{"Other"}, "Reviews"), Post: op([]string{"Other"}, "Reviews")}
	m["/api/v1/agents/reviews/"] = OpenAPIPath{Get: op([]string{"Other"}, "Reviews"), Post: op([]string{"Other"}, "Reviews")}
	m["/api/v1/agents/shadows"] = OpenAPIPath{Get: op([]string{"Other"}, "Shadows"), Post: op([]string{"Other"}, "Shadows")}
	m["/api/v1/agents/token"] = OpenAPIPath{Get: op([]string{"Other"}, "Token"), Post: op([]string{"Other"}, "Token")}
	m["/api/v1/agents/verify"] = OpenAPIPath{Get: op([]string{"Other"}, "Verify"), Post: op([]string{"Other"}, "Verify")}
	m["/api/v1/alerts"] = OpenAPIPath{Get: op([]string{"Other"}, "Alerts"), Post: op([]string{"Other"}, "Alerts")}
	m["/api/v1/audit"] = OpenAPIPath{Get: op([]string{"Audit"}, "Audit"), Post: op([]string{"Audit"}, "Audit")}
	m["/api/v1/authz/check"] = OpenAPIPath{Post: op([]string{"Authz"}, "Simplified permission check: {user_id, resource, action} -> {allowed}")}
	m["/api/v1/certificates"] = OpenAPIPath{Get: op([]string{"Platform"}, "V1 Certificates")}
	m["/api/v1/certificates/"] = OpenAPIPath{Get: op([]string{"Platform"}, "V1 Certificates")}
	m["/api/v1/crypto/fields"] = OpenAPIPath{Get: op([]string{"Crypto"}, "V1 Crypto Fields")}
	m["/api/v1/crypto/fields/"] = OpenAPIPath{Get: op([]string{"Crypto"}, "V1 Crypto Fields")}
	m["/api/v1/departments"] = OpenAPIPath{Get: op([]string{"Org"}, "List departments"), Post: op([]string{"Org"}, "Create department")}
	m["/api/v1/departments/"] = OpenAPIPath{Post: op([]string{"Platform"}, "V1 Departments")}
	m["/api/v1/dlp/policies"] = OpenAPIPath{Get: op([]string{"DLP"}, "V1 Dlp Policies")}
	m["/api/v1/dlp/policies/"] = OpenAPIPath{Get: op([]string{"DLP"}, "V1 Dlp Policies")}
	m["/api/v1/dlp/scan"] = OpenAPIPath{Post: op([]string{"DLP"}, "V1 Dlp Scan")}
	m["/api/v1/event-correlation/rules"] = OpenAPIPath{Get: op([]string{"Platform"}, "V1 Event Correlation Rules")}
	m["/api/v1/groups"] = OpenAPIPath{Get: op([]string{"Identity"}, "List groups"), Post: op([]string{"Identity"}, "Create group")}
	m["/api/v1/groups/{id}"] = OpenAPIPath{Get: op([]string{"Identity"}, "Get group by ID"), Put: op([]string{"Identity"}, "Update group"), Delete: op([]string{"Identity"}, "Delete group")}
	m["/api/v1/groups/{id}/members"] = OpenAPIPath{Get: op([]string{"Identity"}, "List group members"), Post: op([]string{"Identity"}, "Add member to group")}
	m["/api/v1/hr/connectors"] = OpenAPIPath{Get: op([]string{"HR"}, "List HR connectors"), Post: op([]string{"HR"}, "Add HR connector")}
	m["/api/v1/hr/dormant"] = OpenAPIPath{Get: op([]string{"HR"}, "Dormant accounts")}
	m["/api/v1/hr/reconcile"] = OpenAPIPath{Get: op([]string{"Platform"}, "V1 Hr Reconcile")}
	m["/api/v1/hr/sync"] = OpenAPIPath{Post: op([]string{"HR"}, "Trigger HR sync")}
	m["/api/v1/hr/sync/log"] = OpenAPIPath{Get: op([]string{"Platform"}, "V1 Hr Sync Log")}
	m["/api/v1/idp/config"] = OpenAPIPath{Get: op([]string{"Platform"}, "V1 Idp Config")}
	m["/api/v1/mdm/connectors"] = OpenAPIPath{Get: op([]string{"MDM"}, "List MDM connectors"), Post: op([]string{"MDM"}, "Add MDM connector")}
	m["/api/v1/mdm/devices"] = OpenAPIPath{Get: op([]string{"MDM"}, "List MDM devices")}
	m["/api/v1/mdm/devices/"] = OpenAPIPath{Get: op([]string{"Devices"}, "V1 Mdm Devices")}
	m["/api/v1/mdm/sync/"] = OpenAPIPath{Get: op([]string{"Platform"}, "V1 Mdm Sync")}
	m["/api/v1/notifications/log"] = OpenAPIPath{Get: op([]string{"Platform"}, "V1 Notifications Log")}
	m["/api/v1/notifications/rules"] = OpenAPIPath{Get: op([]string{"Notifications"}, "List notification rules"), Post: op([]string{"Notifications"}, "Create notification rule")}
	m["/api/v1/notifications/send"] = OpenAPIPath{Get: op([]string{"Platform"}, "V1 Notifications Send")}
	m["/api/v1/observability/health"] = OpenAPIPath{Get: op([]string{"Observability"}, "Exporter health")}
	m["/api/v1/organizations"] = OpenAPIPath{Get: op([]string{"Other"}, "Organizations"), Post: op([]string{"Other"}, "Organizations")}
	m["/api/v1/organizations/"] = OpenAPIPath{Get: op([]string{"Other"}, "Organizations"), Post: op([]string{"Other"}, "Organizations")}
	m["/api/v1/orgs"] = OpenAPIPath{Get: op([]string{"Org"}, "List organizations"), Post: op([]string{"Org"}, "Create organization")}
	m["/api/v1/orgs/"] = OpenAPIPath{Get: op([]string{"Org"}, "Orgs"), Post: op([]string{"Org"}, "Orgs")}
	m["/api/v1/orgs/tree"] = OpenAPIPath{Get: op([]string{"Org"}, "Tree"), Post: op([]string{"Org"}, "Tree")}
	m["/api/v1/orgs/tree-with-members"] = OpenAPIPath{Get: op([]string{"Org"}, "Tree With Members"), Post: op([]string{"Org"}, "Tree With Members")}
	m["/api/v1/orgs/{id}"] = OpenAPIPath{Get: op([]string{"Org"}, "Get organization by ID"), Put: op([]string{"Org"}, "Update organization"), Delete: op([]string{"Org"}, "Delete organization")}
	m["/api/v1/permissions"] = OpenAPIPath{Get: op([]string{"Policy"}, "List permissions"), Post: op([]string{"Policy"}, "Create permission")}
	m["/api/v1/permissions/tree"] = OpenAPIPath{Get: op([]string{"Policy"}, "Tree"), Post: op([]string{"Policy"}, "Tree")}
	m["/api/v1/plugins"] = OpenAPIPath{Get: op([]string{"Other"}, "Plugins"), Post: op([]string{"Other"}, "Plugins")}
	m["/api/v1/plugins/"] = OpenAPIPath{Get: op([]string{"Other"}, "Plugins"), Post: op([]string{"Other"}, "Plugins")}
	m["/api/v1/policies"] = OpenAPIPath{Get: op([]string{"Policy"}, "List policies"), Post: op([]string{"Policy"}, "Create policy")}
	m["/api/v1/policies/"] = OpenAPIPath{Get: op([]string{"Policy"}, "Policies"), Post: op([]string{"Policy"}, "Policies")}
	m["/api/v1/policies/abac/evaluate"] = OpenAPIPath{Get: op([]string{"Policy"}, "Evaluate"), Post: op([]string{"Policy"}, "Evaluate")}
	m["/api/v1/policies/abac/export"] = OpenAPIPath{Get: op([]string{"Policy"}, "Export"), Post: op([]string{"Policy"}, "Export")}
	m["/api/v1/policies/abac/groups"] = OpenAPIPath{Get: op([]string{"Policy"}, "Groups"), Post: op([]string{"Policy"}, "Groups")}
	m["/api/v1/policies/abac/import"] = OpenAPIPath{Get: op([]string{"Policy"}, "Import"), Post: op([]string{"Policy"}, "Import")}
	m["/api/v1/policies/access-frequency"] = OpenAPIPath{Get: op([]string{"Policy"}, "Access Frequency"), Post: op([]string{"Policy"}, "Access Frequency")}
	m["/api/v1/policies/access-paths"] = OpenAPIPath{Get: op([]string{"Policy"}, "Access Paths"), Post: op([]string{"Policy"}, "Access Paths")}
	m["/api/v1/policies/access-paths/optimization"] = OpenAPIPath{Get: op([]string{"Policy"}, "Optimization"), Post: op([]string{"Policy"}, "Optimization")}
	m["/api/v1/policies/access-requests"] = OpenAPIPath{Get: op([]string{"Policy"}, "Access Requests"), Post: op([]string{"Policy"}, "Access Requests")}
	m["/api/v1/policies/access-requests/"] = OpenAPIPath{Get: op([]string{"Policy"}, "Access Requests"), Post: op([]string{"Policy"}, "Access Requests")}
	m["/api/v1/policies/access-requests/pending"] = OpenAPIPath{Get: op([]string{"Policy"}, "Pending"), Post: op([]string{"Policy"}, "Pending")}
	m["/api/v1/policies/access-review-exemptions"] = OpenAPIPath{Get: op([]string{"Policy"}, "Access Review Exemptions"), Post: op([]string{"Policy"}, "Access Review Exemptions")}
	m["/api/v1/policies/access-review-exemptions/"] = OpenAPIPath{Get: op([]string{"Policy"}, "Access Review Exemptions"), Post: op([]string{"Policy"}, "Access Review Exemptions")}
	m["/api/v1/policies/access-reviews/"] = OpenAPIPath{Get: op([]string{"Policy"}, "Access Reviews"), Post: op([]string{"Policy"}, "Access Reviews")}
	m["/api/v1/policies/access-reviews/auto-assign"] = OpenAPIPath{Get: op([]string{"Policy"}, "Auto Assign"), Post: op([]string{"Policy"}, "Auto Assign")}
	m["/api/v1/policies/access-reviews/campaigns"] = OpenAPIPath{Get: op([]string{"Policy"}, "Campaigns"), Post: op([]string{"Policy"}, "Campaigns")}
	m["/api/v1/policies/access-reviews/campaigns/"] = OpenAPIPath{Get: op([]string{"Policy"}, "Campaigns"), Post: op([]string{"Policy"}, "Campaigns")}
	m["/api/v1/policies/access-reviews/campaigns/active"] = OpenAPIPath{Get: op([]string{"Policy"}, "Active"), Post: op([]string{"Policy"}, "Active")}
	m["/api/v1/policies/access-reviews/delegate"] = OpenAPIPath{Get: op([]string{"Policy"}, "Delegate"), Post: op([]string{"Policy"}, "Delegate")}
	m["/api/v1/policies/access-reviews/delegated"] = OpenAPIPath{Get: op([]string{"Policy"}, "Delegated"), Post: op([]string{"Policy"}, "Delegated")}
	m["/api/v1/policies/access-reviews/escalate"] = OpenAPIPath{Get: op([]string{"Policy"}, "Escalate"), Post: op([]string{"Policy"}, "Escalate")}
	m["/api/v1/policies/access-reviews/escalated"] = OpenAPIPath{Get: op([]string{"Policy"}, "Escalated"), Post: op([]string{"Policy"}, "Escalated")}
	m["/api/v1/policies/access-reviews/metrics"] = OpenAPIPath{Get: op([]string{"Policy"}, "Metrics"), Post: op([]string{"Policy"}, "Metrics")}
	m["/api/v1/policies/analyze"] = OpenAPIPath{Get: op([]string{"Policy"}, "Analyze"), Post: op([]string{"Policy"}, "Analyze")}
	m["/api/v1/policies/approvals"] = OpenAPIPath{Get: op([]string{"Policy"}, "Approvals"), Post: op([]string{"Policy"}, "Approvals")}
	m["/api/v1/policies/approvals/"] = OpenAPIPath{Get: op([]string{"Policy"}, "Approvals"), Post: op([]string{"Policy"}, "Approvals")}
	m["/api/v1/policies/attribute-mapping"] = OpenAPIPath{Get: op([]string{"Policy"}, "Attribute Mapping"), Post: op([]string{"Policy"}, "Attribute Mapping")}
	m["/api/v1/policies/break-glass"] = OpenAPIPath{Get: op([]string{"Policy"}, "Break Glass"), Post: op([]string{"Policy"}, "Break Glass")}
	m["/api/v1/policies/break-glass/active"] = OpenAPIPath{Get: op([]string{"Policy"}, "Active"), Post: op([]string{"Policy"}, "Active")}
	m["/api/v1/policies/bundles"] = OpenAPIPath{Get: op([]string{"Policy"}, "Bundles"), Post: op([]string{"Policy"}, "Bundles")}
	m["/api/v1/policies/check"] = OpenAPIPath{Post: op([]string{"Policy"}, "Check access (principal, resource, action) → allow/deny")}
	m["/api/v1/policies/conditional-access"] = OpenAPIPath{Get: op([]string{"Policy"}, "Conditional Access"), Post: op([]string{"Policy"}, "Conditional Access")}
	m["/api/v1/policies/conflicts/resolve"] = OpenAPIPath{Get: op([]string{"Policy"}, "Resolve"), Post: op([]string{"Policy"}, "Resolve")}
	m["/api/v1/policies/decision-log"] = OpenAPIPath{Get: op([]string{"Policy"}, "Decision Log"), Post: op([]string{"Policy"}, "Decision Log")}
	m["/api/v1/policies/default-action"] = OpenAPIPath{Get: op([]string{"Policy"}, "Default Action"), Post: op([]string{"Policy"}, "Default Action")}
	m["/api/v1/policies/delegate"] = OpenAPIPath{Get: op([]string{"Policy"}, "Delegate"), Post: op([]string{"Policy"}, "Delegate")}
	m["/api/v1/policies/delegated-admin"] = OpenAPIPath{Get: op([]string{"Policy"}, "Delegated Admin"), Post: op([]string{"Policy"}, "Delegated Admin")}
	m["/api/v1/policies/delegated-admin/list"] = OpenAPIPath{Get: op([]string{"Policy"}, "List"), Post: op([]string{"Policy"}, "List")}
	m["/api/v1/policies/delegations"] = OpenAPIPath{Get: op([]string{"Policy"}, "Delegations"), Post: op([]string{"Policy"}, "Delegations")}
	m["/api/v1/policies/diff"] = OpenAPIPath{Get: op([]string{"Policy"}, "Diff"), Post: op([]string{"Policy"}, "Diff")}
	m["/api/v1/policies/dry-run"] = OpenAPIPath{Get: op([]string{"Policy"}, "Dry Run"), Post: op([]string{"Policy"}, "Dry Run")}
	m["/api/v1/policies/dynamic-roles"] = OpenAPIPath{Get: op([]string{"Policy"}, "Dynamic Roles"), Post: op([]string{"Policy"}, "Dynamic Roles")}
	m["/api/v1/policies/dynamic-roles/list"] = OpenAPIPath{Get: op([]string{"Policy"}, "List"), Post: op([]string{"Policy"}, "List")}
	m["/api/v1/policies/effectiveness"] = OpenAPIPath{Get: op([]string{"Policy"}, "Effectiveness"), Post: op([]string{"Policy"}, "Effectiveness")}
	m["/api/v1/policies/emergency-access/audit"] = OpenAPIPath{Get: op([]string{"Policy"}, "Audit"), Post: op([]string{"Policy"}, "Audit")}
	m["/api/v1/policies/evaluate"] = OpenAPIPath{Post: op([]string{"Policy"}, "Evaluate all matching policies with decision trail")}
	m["/api/v1/policies/export"] = OpenAPIPath{Get: op([]string{"Policy"}, "Export"), Post: op([]string{"Policy"}, "Export")}
	m["/api/v1/policies/export-package"] = OpenAPIPath{Get: op([]string{"Policy"}, "Export Package"), Post: op([]string{"Policy"}, "Export Package")}
	m["/api/v1/policies/export-yaml"] = OpenAPIPath{Get: op([]string{"Policy"}, "Export Yaml"), Post: op([]string{"Policy"}, "Export Yaml")}
	m["/api/v1/policies/from-template/"] = OpenAPIPath{Get: op([]string{"Policy"}, "From Template"), Post: op([]string{"Policy"}, "From Template")}
	m["/api/v1/policies/impact-preview"] = OpenAPIPath{Get: op([]string{"Policy"}, "Impact Preview"), Post: op([]string{"Policy"}, "Impact Preview")}
	m["/api/v1/policies/import"] = OpenAPIPath{Get: op([]string{"Policy"}, "Import"), Post: op([]string{"Policy"}, "Import")}
	m["/api/v1/policies/import-package"] = OpenAPIPath{Get: op([]string{"Policy"}, "Import Package"), Post: op([]string{"Policy"}, "Import Package")}
	m["/api/v1/policies/import-yaml"] = OpenAPIPath{Get: op([]string{"Policy"}, "Import Yaml"), Post: op([]string{"Policy"}, "Import Yaml")}
	m["/api/v1/policies/inheritance"] = OpenAPIPath{Get: op([]string{"Policy"}, "Inheritance"), Post: op([]string{"Policy"}, "Inheritance")}
	m["/api/v1/policies/inheritance/"] = OpenAPIPath{Get: op([]string{"Policy"}, "Inheritance"), Post: op([]string{"Policy"}, "Inheritance")}
	m["/api/v1/policies/jit-elevate"] = OpenAPIPath{Get: op([]string{"Policy"}, "Jit Elevate"), Post: op([]string{"Policy"}, "Jit Elevate")}
	m["/api/v1/policies/jit/active"] = OpenAPIPath{Get: op([]string{"Policy"}, "Active"), Post: op([]string{"Policy"}, "Active")}
	m["/api/v1/policies/jit/request"] = OpenAPIPath{Get: op([]string{"Policy"}, "Request"), Post: op([]string{"Policy"}, "Request")}
	m["/api/v1/policies/jit/requests"] = OpenAPIPath{Get: op([]string{"Policy"}, "Requests"), Post: op([]string{"Policy"}, "Requests")}
	m["/api/v1/policies/jit/requests/"] = OpenAPIPath{Get: op([]string{"Policy"}, "Requests"), Post: op([]string{"Policy"}, "Requests")}
	m["/api/v1/policies/merge-conflicts"] = OpenAPIPath{Get: op([]string{"Policy"}, "Merge Conflicts"), Post: op([]string{"Policy"}, "Merge Conflicts")}
	m["/api/v1/policies/permission-boundaries"] = OpenAPIPath{Get: op([]string{"Policy"}, "Permission Boundaries"), Post: op([]string{"Policy"}, "Permission Boundaries")}
	m["/api/v1/policies/permissions/tree"] = OpenAPIPath{Get: op([]string{"Policy"}, "Tree"), Post: op([]string{"Policy"}, "Tree")}
	m["/api/v1/policies/policy-set/evaluate"] = OpenAPIPath{Get: op([]string{"Policy"}, "Evaluate"), Post: op([]string{"Policy"}, "Evaluate")}
	m["/api/v1/policies/privileged-access"] = OpenAPIPath{Get: op([]string{"Policy"}, "Privileged Access"), Post: op([]string{"Policy"}, "Privileged Access")}
	m["/api/v1/policies/privileged-access/revoke"] = OpenAPIPath{Get: op([]string{"Policy"}, "Revoke"), Post: op([]string{"Policy"}, "Revoke")}
	m["/api/v1/policies/sod/check"] = OpenAPIPath{Post: op([]string{"Policy"}, "Check Separation of Duties violations")}
	m["/api/v1/policies/sod/violations"] = OpenAPIPath{Get: op([]string{"Policy"}, "List SoD violations")}
	m["/api/v1/policies/{id}"] = OpenAPIPath{Get: op([]string{"Policy"}, "Get policy by ID"), Put: op([]string{"Policy"}, "Update policy"), Delete: op([]string{"Policy"}, "Delete policy")}
	m["/api/v1/quotas/{tenant_id}"] = OpenAPIPath{Get: op([]string{"Admin"}, "Get tenant quota"), Put: op([]string{"Admin"}, "Update quota")}
	m["/api/v1/risk/evaluate"] = OpenAPIPath{Post: op([]string{"Risk"}, "Evaluate risk score")}
	m["/api/v1/risk/scores/{user_id}"] = OpenAPIPath{Get: op([]string{"Risk"}, "Get risk score for user")}
	m["/api/v1/risk/signals"] = OpenAPIPath{Get: op([]string{"Risk"}, "List risk signals")}
	m["/api/v1/soar/playbooks"] = OpenAPIPath{Get: op([]string{"SOAR"}, "List SOAR playbooks"), Post: op([]string{"SOAR"}, "Create SOAR playbook")}
	m["/api/v1/teams"] = OpenAPIPath{Get: op([]string{"Org"}, "List teams"), Post: op([]string{"Org"}, "Create team")}
	m["/graphql"] = OpenAPIPath{Post: op([]string{"GraphQL"}, "GraphQL endpoint")}
	m["/healthz"] = OpenAPIPath{Get: op([]string{"System"}, "Health check")}
	m["/readyz"] = OpenAPIPath{Get: op([]string{"System"}, "Readiness check")}
}

// SwaggerUIHandler serves the interactive Swagger UI at /docs.
func SwaggerUIHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(swaggerUITemplate))
	}
}

// OpenAPISpecHandler serves the raw JSON spec at /swagger.json.
func OpenAPISpecHandler() http.HandlerFunc {
	spec := GenerateOpenAPISpec()
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(spec)

	}
}

const swaggerUITemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>GGID API Documentation</title>
<link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
<style>
  body { margin: 0; }
  .topbar { background: #1a1a2e; padding: 10px 20px; display: flex; align-items: center; gap: 16px; }
  .topbar a { color: #e0e0e0; text-decoration: none; font-size: 14px; }
  .topbar .brand { font-size: 18px; font-weight: bold; color: #fff; }
  .topbar .badge { background: #4CAF50; color: #fff; padding: 2px 8px; border-radius: 4px; font-size: 12px; }
</style>
</head>
<body>
<div class="topbar">
  <span class="brand">GGID Platform API</span>
  <span class="badge">Interactive</span>
  <a href="/swagger.json" target="_blank">swagger.json</a>
  <a href="https://github.com/topcheer/ggid" target="_blank">GitHub</a>
</div>
<div id="swagger-ui"></div>
<script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
<script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-standalone-preset.js"></script>
<script>
window.onload = function() {
  window.ui = SwaggerUIBundle({
    url: '/swagger.json',
    dom_id: '#swagger-ui',
    deepLinking: true,
    presets: [SwaggerUIBundle.presets.apis, SwaggerUIStandalonePreset],
    plugins: [SwaggerUIBundle.plugins.DownloadUrl],
    layout: 'StandaloneLayout',
    docExpansion: 'list',
    filter: true,
    showRequestHeaders: true,
    showCommonExtensions: true,
    tryItOutEnabled: true,
    supportedSubmitMethods: ['get', 'post', 'put', 'delete', 'patch'],
    requestInterceptor: function(req) {
      // Auto-attach tenant header from localStorage if available.
      var token = localStorage.getItem('ggid_token');
      if (token) {
        req.headers['Authorization'] = 'Bearer ' + token;
      }
      var tenant = localStorage.getItem('ggid_tenant');
      if (tenant) {
        req.headers['X-Tenant-ID'] = tenant;
      }
      return req;
    },
    responseInterceptor: function(res) {
      // Auto-capture token from login responses.
      if (res.url.includes('/auth/login') && res.body) {
        try {
          var body = JSON.parse(res.body);
          if (body.access_token) {
            localStorage.setItem('ggid_token', body.access_token);
          }
        } catch(e) {}
      }
      return res;
    }
  });
};
</script>
</body>
</html>`
