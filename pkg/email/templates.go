package email

import (
	"html"
	"strings"
)

// safeURL sanitises a URL for use in an href attribute. It rejects dangerous
// protocols (javascript:, data:, vbscript:) by returning a safe placeholder.
// This prevents XSS via href injection even if html.EscapeString is bypassed.
func safeURL(link string) string {
	lower := strings.ToLower(strings.TrimSpace(link))
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		return html.EscapeString(link)
	}
	// Relative URLs (starting with /) are also safe.
	if strings.HasPrefix(link, "/") {
		return html.EscapeString(link)
	}
	// Block javascript:, data:, vbscript:, and any other non-http scheme.
	return "#"
}

// Templates provides HTML email templates for common IAM notifications.
// All user-supplied fields are HTML-escaped to prevent injection attacks.

// PasswordResetData holds data for the password reset email.
type PasswordResetData struct {
	UserName string
	Link     string
	Expiry   string // Human-readable expiry (e.g., "30 minutes")
	AppName  string
}

// PasswordResetHTML returns an HTML email body for password reset.
func PasswordResetHTML(d PasswordResetData) string {
	appName := d.AppName
	if appName == "" {
		appName = "GGID"
	}
	return `<html><body style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
<h2 style="color: #1a1a1a;">Password Reset Request</h2>
<p style="color: #4a4a4a; font-size: 16px; line-height: 1.5;">Hi ` + html.EscapeString(d.UserName) + `,</p>
<p style="color: #4a4a4a; font-size: 16px; line-height: 1.5;">
We received a request to reset your password. Click the button below to set a new password:
</p>
<p style="text-align: center; margin: 30px 0;">
<a href="` + safeURL(d.Link) + `" style="background-color: #4F46E5; color: white; padding: 12px 32px; text-decoration: none; border-radius: 6px; font-size: 16px; font-weight: bold;">Reset Password</a>
</p>
<p style="color: #6b7280; font-size: 14px;">
This link will expire in ` + html.EscapeString(d.Expiry) + `.
</p>
<p style="color: #6b7280; font-size: 14px;">
If you didn't request a password reset, you can safely ignore this email. Your password has not been changed.
</p>
<hr style="border: none; border-top: 1px solid #e5e7eb; margin: 30px 0;">
<p style="color: #9ca3af; font-size: 12px;">
This is an automated message from ` + html.EscapeString(appName) + `. Please do not reply to this email.
</p>
</body></html>`
}

// PasswordResetText returns a plain text email body for password reset.
func PasswordResetText(d PasswordResetData) string {
	return "Hi " + d.UserName + ",\n\n" +
		"We received a request to reset your password. Click the link below to set a new password:\n\n" +
		d.Link + "\n\n" +
		"This link will expire in " + d.Expiry + ".\n\n" +
		"If you didn't request a password reset, you can safely ignore this email.\n"
}

// EmailVerificationData holds data for the email verification email.
type EmailVerificationData struct {
	UserName string
	Link     string
	AppName  string
}

// EmailVerificationHTML returns an HTML email body for email verification.
func EmailVerificationHTML(d EmailVerificationData) string {
	appName := d.AppName
	if appName == "" {
		appName = "GGID"
	}
	return `<html><body style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
<h2 style="color: #1a1a1a;">Verify Your Email</h2>
<p style="color: #4a4a4a; font-size: 16px;">Hi ` + html.EscapeString(d.UserName) + `,</p>
<p style="color: #4a4a4a; font-size: 16px;">Please verify your email address by clicking the button below:</p>
<p style="text-align: center; margin: 30px 0;">
<a href="` + safeURL(d.Link) + `" style="background-color: #10B981; color: white; padding: 12px 32px; text-decoration: none; border-radius: 6px; font-weight: bold;">Verify Email</a>
</p>
<hr style="border: none; border-top: 1px solid #e5e7eb; margin: 30px 0;">
<p style="color: #9ca3af; font-size: 12px;">This is an automated message from ` + html.EscapeString(appName) + `.</p>
</body></html>`
}

// WelcomeData holds data for the welcome email.
type WelcomeData struct {
	UserName string
	AppName  string
	Link     string
}

// WelcomeHTML returns an HTML email body for new user welcome.
func WelcomeHTML(d WelcomeData) string {
	appName := d.AppName
	if appName == "" {
		appName = "GGID"
	}
	return `<html><body style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
<h2 style="color: #1a1a1a;">Welcome to ` + html.EscapeString(appName) + `!</h2>
<p style="color: #4a4a4a; font-size: 16px;">Hi ` + html.EscapeString(d.UserName) + `,</p>
<p style="color: #4a4a4a; font-size: 16px;">Your account has been created successfully. Welcome aboard!</p>
<p style="text-align: center; margin: 30px 0;">
<a href="` + safeURL(d.Link) + `" style="background-color: #4F46E5; color: white; padding: 12px 32px; text-decoration: none; border-radius: 6px; font-weight: bold;">Get Started</a>
</p>
<hr style="border: none; border-top: 1px solid #e5e7eb; margin: 30px 0;">
<p style="color: #9ca3af; font-size: 12px;">This is an automated message from ` + html.EscapeString(appName) + `.</p>
</body></html>`
}

// MFACodeData holds data for the MFA code email.
type MFACodeData struct {
	UserName string
	Code     string
}

// MFACodeHTML returns an HTML email body for MFA OTP code.
func MFACodeHTML(d MFACodeData) string {
	return `<html><body style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
<h2 style="color: #1a1a1a;">Your Verification Code</h2>
<p style="color: #4a4a4a; font-size: 16px;">Hi ` + html.EscapeString(d.UserName) + `,</p>
<p style="color: #4a4a4a; font-size: 16px;">Use the following code to complete your login:</p>
<p style="text-align: center; margin: 30px 0;">
<span style="font-size: 36px; font-weight: bold; letter-spacing: 8px; color: #4F46E5;">` + html.EscapeString(d.Code) + `</span>
</p>
<p style="color: #6b7280; font-size: 14px;">This code will expire in 10 minutes.</p>
<hr style="border: none; border-top: 1px solid #e5e7eb; margin: 30px 0;">
<p style="color: #9ca3af; font-size: 12px;">If you didn't request this code, please ignore this email.</p>
</body></html>`
}
