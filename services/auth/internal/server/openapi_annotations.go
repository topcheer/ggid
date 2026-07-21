// Package server OpenAPI annotations for the auth service.
// These comments are consumed by swaggo/swag to generate OpenAPI documentation.
// To regenerate: swag init -g services/auth/internal/server/http.go
package server

// --- Auth: Authentication Endpoints ---

// Login godoc
// @Summary User login
// @Description Authenticate a user with username/password and return JWT tokens. Supports tenant resolution via tenant_id or tenant_slug. Includes brute-force protection and account lockout checks.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body loginRequest true "Login credentials"
// @Success 200 {object} domain.TokenSet "JWT access + refresh tokens"
// @Failure 400 {object} map[string]string "invalid request body"
// @Failure 401 {object} map[string]string "invalid credentials"
// @Failure 423 {object} map[string]string "account locked"
// @Failure 429 {object} map[string]string "too many login attempts"
// @Router /api/v1/auth/verify [post]

// Register godoc
// @Summary Register new user
// @Description Create a new user credential with username, email, and password. Password must meet the configured policy requirements.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body registerRequest true "Registration details"
// @Success 201 {object} map[string]bool "registration successful"
// @Failure 400 {object} map[string]string "invalid request or password too weak"
// @Failure 409 {object} map[string]string "credential already exists"
// @Router /api/v1/auth/register [post]

// Logout godoc
// @Summary User logout
// @Description Revoke the session associated with the provided refresh token.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body object true "Logout request" example("{\"refresh_token\":\"string\"}")
// @Success 200 {object} map[string]bool "logged out"
// @Failure 400 {object} map[string]string "invalid request"
// @Router /api/v1/auth/logout [post]

// Refresh godoc
// @Summary Refresh access token
// @Description Validate and rotate the refresh token, issuing a new access token and refresh token pair.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body object true "Refresh request" example("{\"refresh_token\":\"string\"}")
// @Success 200 {object} domain.TokenSet "New JWT tokens"
// @Failure 401 {object} map[string]string "invalid or expired refresh token"
// @Router /api/v1/auth/refresh [post]

// ForgotPassword godoc
// @Summary Request password reset
// @Description Initiate the password reset flow. Sends a reset token to the user's email if the account exists. Does not reveal whether the email exists.
// @Tags auth,password
// @Accept json
// @Produce json
// @Param request body object true "Forgot password request" example("{\"email\":\"user@example.com\",\"tenant_id\":\"uuid\"}")
// @Success 200 {object} map[string]bool "reset email sent if account exists"
// @Router /api/v1/auth/forgot-password [post]

// ResetPassword godoc
// @Summary Reset password with token
// @Description Complete the password reset flow using a one-time reset token received via email.
// @Tags auth,password
// @Accept json
// @Produce json
// @Param request body object true "Reset request" example("{\"token\":\"string\",\"new_password\":\"string\"}")
// @Success 200 {object} map[string]bool "password reset successful"
// @Failure 400 {object} map[string]string "invalid or expired token"
// @Router /api/v1/auth/reset-password [post]

// ChangePassword godoc
// @Summary Change password
// @Description Change the password for an authenticated user. Requires the old password for verification. Invalidates all other sessions after successful change.
// @Tags auth,password
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body object true "Change password request" example("{\"old_password\":\"string\",\"new_password\":\"string\"}")
// @Success 200 {object} map[string]bool "password changed"
// @Failure 400 {object} map[string]string "invalid request or password too weak"
// @Failure 401 {object} map[string]string "old password incorrect"
// @Router /api/v1/auth/password/change [post]

// HandleSessions godoc
// @Summary List user sessions
// @Description List all active sessions for the authenticated user.
// @Tags auth,sessions
// @Produce json
// @Security BearerAuth
// @Success 200 {array} domain.Session "Active sessions"
// @Failure 401 {object} map[string]string "unauthorized"
// @Router /api/v1/auth/sessions [get]

// HandleInvalidateSessions godoc
// @Summary Invalidate user sessions
// @Description Invalidate all sessions for a user. Used for password change, MFA enrollment, or posture drop events. Supports except_session_id to preserve the current session.
// @Tags auth,sessions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param user_id path string true "User ID (UUID)"
// @Param request body InvalidationRequest false "Invalidation options"
// @Success 200 {object} map[string]any "invalidation result with counts"
// @Failure 400 {object} map[string]string "invalid user_id or reason"
// @Router /api/v1/auth/invalidate-sessions/{user_id} [post]

// MfaSetup godoc
// @Summary Setup MFA device
// @Description Generate a new TOTP secret for the user. Returns a QR code URI for authenticator app setup. Device is created in disabled state until verified.
// @Tags auth,mfa
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body mfaSetupRequest true "MFA setup request"
// @Success 200 {object} service.SetupResponse "Device ID, secret, and QR code URI"
// @Failure 400 {object} map[string]string "invalid request"
// @Failure 409 {object} map[string]string "MFA already enabled"
// @Router /api/v1/auth/mfa/setup [post]

// MfaVerify godoc
// @Summary Verify MFA code
// @Description Verify a TOTP code for an MFA device. On first verification, the device is enabled. On first enrollment, all non-MFA sessions are invalidated.
// @Tags auth,mfa
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body mfaVerifyRequest true "MFA verify request"
// @Success 200 {object} map[string]bool "verified"
// @Failure 400 {object} map[string]string "invalid device_id"
// @Failure 401 {object} map[string]string "invalid MFA code"
// @Router /api/v1/auth/mfa/verify [post]

// MfaDisable godoc
// @Summary Disable MFA device
// @Description Remove an MFA device for the user.
// @Tags auth,mfa
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body mfaDisableRequest true "MFA disable request"
// @Success 200 {object} map[string]bool "disabled"
// @Failure 400 {object} map[string]string "invalid device_id"
// @Router /api/v1/auth/mfa/disable [post]

// MfaLogin godoc
// @Summary Complete MFA login
// @Description Complete the MFA challenge during login by verifying the TOTP code. Issues full JWT tokens on success.
// @Tags auth,mfa
// @Accept json
// @Produce json
// @Param request body object true "MFA login request" example("{\"username\":\"string\",\"password\":\"string\",\"mfa_code\":\"123456\",\"ip\":\"string\",\"user_agent\":\"string\"}")
// @Success 200 {object} domain.TokenSet "JWT tokens"
// @Failure 401 {object} map[string]string "invalid credentials or MFA code"
// @Router /api/v1/auth/mfa/login [post]

// VerifyEmail godoc
// @Summary Verify email address
// @Description Verify a user's email address using a verification token.
// @Tags auth,email
// @Accept json
// @Produce json
// @Param request body object true "Verify email request" example("{\"token\":\"string\"}")
// @Success 200 {object} map[string]bool "email verified"
// @Failure 400 {object} map[string]string "invalid or expired token"
// @Router /api/v1/auth/verify-email [post]

// --- Auth: WebAuthn / Passkey Endpoints ---

// HandleWebAuthnPasswordlessBegin godoc
// @Summary Begin WebAuthn passwordless login
// @Description Initiate a WebAuthn passwordless authentication flow. Returns the challenge for the browser's credential picker.
// @Tags auth,webauthn
// @Accept json
// @Produce json
// @Param request body object true "Begin request" example("{\"user_id\":\"uuid\"}")
// @Success 200 {object} map[string]any "Challenge options"
// @Router /api/v1/auth/webauthn/passwordless/begin [post]

// HandleWebAuthnPasswordlessFinish godoc
// @Summary Finish WebAuthn passwordless login
// @Description Complete the WebAuthn passwordless authentication by verifying the assertion from the browser.
// @Tags auth,webauthn
// @Accept json
// @Produce json
// @Param request body object true "Finish assertion"
// @Success 200 {object} domain.TokenSet "JWT tokens"
// @Failure 401 {object} map[string]string "authentication failed"
// @Router /api/v1/auth/webauthn/passwordless/finish [post]

// WebAuthnRegisterBegin godoc
// @Summary Begin WebAuthn registration
// @Description Start the WebAuthn credential registration flow. Returns creation options for navigator.credentials.create().
// @Tags auth,webauthn
// @Produce json
// @Security BearerAuth
// @Param user_id query string true "User ID (UUID)"
// @Success 200 {object} map[string]any "Credential creation options"
// @Router /api/v1/auth/webauthn/register/begin [get]

// WebAuthnRegisterFinish godoc
// @Summary Finish WebAuthn registration
// @Description Complete the WebAuthn credential registration by verifying the attestation from the browser.
// @Tags auth,webauthn
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]bool "registration successful"
// @Router /api/v1/auth/webauthn/register/finish [post]

// WebAuthnLoginBegin godoc
// @Summary Begin WebAuthn login
// @Description Start the WebAuthn authentication flow. Returns assertion options for navigator.credentials.get().
// @Tags auth,webauthn
// @Produce json
// @Success 200 {object} map[string]any "Assertion options"
// @Router /api/v1/auth/webauthn/login/begin [get]

// WebAuthnLoginFinish godoc
// @Summary Finish WebAuthn login
// @Description Complete the WebAuthn authentication by verifying the assertion response.
// @Tags auth,webauthn
// @Accept json
// @Produce json
// @Success 200 {object} domain.TokenSet "JWT tokens"
// @Failure 401 {object} map[string]string "authentication failed"
// @Router /api/v1/auth/webauthn/login/finish [post]
