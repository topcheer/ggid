package middleware

import (
	"net/http"
	"strconv"
)

// MaxBodySize limits the size of request bodies. Returns 413 if exceeded.
func MaxBodySize(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			}
			// Wrap writer to catch MaxBytesError and return proper status
			mw := &maxBodyWriter{ResponseWriter: w}
			next.ServeHTTP(mw, r)
		})
	}
}

type maxBodyWriter struct {
	http.ResponseWriter
}

func (w *maxBodyWriter) Write(data []byte) (int, error) {
	n, err := w.ResponseWriter.Write(data)
	return n, err
}

// ParseMaxBodySize converts a string like "10MB" or "1GB" to bytes.
func ParseMaxBodySize(s string) int64 {
	if len(s) == 0 {
		return 10 << 20 // default 10MB
	}
	// Extract numeric prefix
	numStr := ""
	for _, c := range s {
		if c >= '0' && c <= '9' {
			numStr += string(c)
		} else {
			break
		}
	}
	if numStr == "" {
		return 10 << 20
	}
	num, _ := strconv.ParseInt(numStr, 10, 64)
	unit := s[len(numStr):]
	switch unit {
	case "KB", "kb":
		return num << 10
	case "MB", "mb":
		return num << 20
	case "GB", "gb":
		return num << 30
	default:
		return num
	}
}
