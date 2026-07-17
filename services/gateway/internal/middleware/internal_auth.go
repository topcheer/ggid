package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strconv"
	"time"
)

// SignInternalRequest injects HMAC signature headers into an outbound request.
// Called by the gateway proxy Director to authenticate to backend services.
func SignInternalRequest(req *http.Request, serviceName string, secret []byte) {
	if len(secret) == 0 || req == nil {
		return
	}
	ts := time.Now().Unix()
	tsStr := strconv.FormatInt(ts, 10)
	reqID := req.Header.Get("X-Request-ID")
	payload := serviceName + "|" + tsStr + "|" + reqID
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(payload))
	sig := hex.EncodeToString(mac.Sum(nil))

	req.Header.Set("X-Internal-Service", serviceName)
	req.Header.Set("X-Internal-Timestamp", tsStr)
	req.Header.Set("X-Internal-Signature", sig)
}
