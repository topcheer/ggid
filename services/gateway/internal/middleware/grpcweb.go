package middleware

import (
	"encoding/base64"
	"io"
	"net/http"
	"strings"
)

// IsGRPCWebRequest detects gRPC-Web requests.
// gRPC-Web sends Content-Type: application/grpc-web or application/grpc-web+proto.
func IsGRPCWebRequest(r *http.Request) bool {
	ct := r.Header.Get("Content-Type")
	return strings.HasPrefix(ct, "application/grpc-web")
}

// GRPCWebTranslator translates between gRPC-Web (browser) and gRPC (backend).
// It handles:
// - Request: decode base64 body, set Content-Type to application/grpc
// - Response: encode body as base64 if the original request was grpc-web
type GRPCWebTranslator struct {
	// BackendAddr is the gRPC backend address (e.g., "localhost:9070")
	BackendAddr string
}

// NewGRPCWebTranslator creates a translator for the given backend.
func NewGRPCWebTranslator(backendAddr string) *GRPCWebTranslator {
	return &GRPCWebTranslator{BackendAddr: backendAddr}
}

// GRPCWebHandler handles gRPC-Web requests by translating them to gRPC.
// For non-gRPC-Web requests, it passes through to the next handler.
func GRPCWebHandler(translator *GRPCWebTranslator, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !IsGRPCWebRequest(r) {
			next.ServeHTTP(w, r)
			return
		}

		// Translate request: grpc-web → grpc
		originalCT := r.Header.Get("Content-Type")
		isText := strings.Contains(originalCT, "+text") || !strings.Contains(originalCT, "+proto")

		// For text-encoded grpc-web, body is base64 encoded
		var grpcBody io.Reader = r.Body
		if isText && r.Body != nil {
			decoded, err := io.ReadAll(base64.NewDecoder(base64.StdEncoding, r.Body))
			if err == nil {
				grpcBody = io.NopCloser(strings.NewReader(string(decoded)))
			}
		}

		// Set Content-Type to application/grpc for backend
		r.Header.Set("Content-Type", "application/grpc")
		r.Body = io.NopCloser(grpcBody)

		// Use a recording response writer to capture the gRPC response
		rec := &grpcWebResponseWriter{
			ResponseWriter: w,
			headers:        make(http.Header),
		}

		// Forward to backend via next handler (which is the gRPC proxy)
		next.ServeHTTP(rec, r)

		// Translate response back: grpc → grpc-web
		w.Header().Set("Content-Type", originalCT)

		// For text mode, encode response as base64
		if isText && len(rec.body) > 0 {
			encoded := base64.StdEncoding.EncodeToString(rec.body)
			// Append grpc-web trailer frame (0x80 + trailers)
			trailer := "grpc-status: 0\r\ngrpc-message: OK\r\n"
			encoded += base64.StdEncoding.EncodeToString([]byte(trailer))
			io.WriteString(w, encoded)
		} else if len(rec.body) > 0 {
			w.Write(rec.body)
			// Append trailer frame
			w.Write([]byte{0x80})
			io.WriteString(w, "grpc-status: 0\r\n")
		}
	})
}

// grpcWebResponseWriter captures the gRPC response for translation.
type grpcWebResponseWriter struct {
	http.ResponseWriter
	body    []byte
	headers http.Header
	status  int
}

func (g *grpcWebResponseWriter) Header() http.Header {
	if g.headers != nil {
		return g.headers
	}
	return g.ResponseWriter.Header()
}

func (g *grpcWebResponseWriter) WriteHeader(code int) {
	g.status = code
	g.ResponseWriter.WriteHeader(code)
}

func (g *grpcWebResponseWriter) Write(b []byte) (int, error) {
	g.body = append(g.body, b...)
	return len(b), nil
}

// GRPCWebTrailers parses grpc-web trailer frames from the end of a response body.
func GRPCWebTrailers(body []byte) (status int, message string) {
	// Trailer frame starts with 0x80 (for text) or 0x00 (for binary)
	if len(body) < 2 {
		return 0, ""
	}
	// Look for the last frame indicator
	for i := len(body) - 1; i >= 0; i-- {
		if body[i] == 0x80 || body[i] == 0x00 {
			trailerStr := string(body[i+1:])
			for _, line := range strings.Split(trailerStr, "\r\n") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					val := strings.TrimSpace(parts[1])
					if key == "grpc-status" {
						var s int
						for _, c := range val {
							s = s*10 + int(c-'0')
						}
						status = s
					}
					if key == "grpc-message" {
						message = val
					}
				}
			}
			break
		}
	}
	return
}
