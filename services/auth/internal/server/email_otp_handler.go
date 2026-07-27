// Legacy email OTP handler — superseded by otp_handler.go.
// Kept for backward compatibility; routes are redirected to new handler.
package server

import (
	"net/http"
)

func (h *Handler) handleEmailOTPSendLegacy(w http.ResponseWriter, r *http.Request) {
	// Deprecated: use handleOTPSend instead
	h.handleOTPSend(w, r)
}

func (h *Handler) handleEmailOTPVerifyLegacy(w http.ResponseWriter, r *http.Request) {
	// Deprecated: use handleOTPVerify instead
	h.handleOTPVerify(w, r)
}
