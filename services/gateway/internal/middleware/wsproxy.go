// Package middleware implements HTTP middleware for the API Gateway.
package middleware

import (
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// IsWebSocketRequest detects whether the request is a WebSocket upgrade.
func IsWebSocketRequest(r *http.Request) bool {
	if !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		return false
	}
	hasConnection := false
	for _, v := range r.Header.Values("Connection") {
		if strings.Contains(strings.ToLower(v), "upgrade") {
			hasConnection = true
			break
		}
	}
	return hasConnection
}

// WebSocketProxy returns an http.HandlerFunc that tunnels a WebSocket
// connection from the client to the target backend.
//
// The target must be a "host:port" style address (e.g. "localhost:9001").
// The function performs an HTTP connection hijack, dials the backend, and
// then copies bytes bidirectionally until either side closes.
func WebSocketProxy(target string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !IsWebSocketRequest(r) {
			WriteError(w, r, http.StatusBadRequest, "bad_request", "not a websocket request")
			return
		}

		hj, ok := w.(http.Hijacker)
		if !ok {
			WriteError(w, r, http.StatusInternalServerError, "internal_error", "hijacking not supported")
			return
		}

		clientConn, clientBuf, err := hj.Hijack()
		if err != nil {
			slog.Error("wsproxy: hijack error", "err", err)
			WriteError(w, r, http.StatusInternalServerError, "internal_error", "hijack failed")
			return
		}
		defer clientConn.Close()

		// Dial backend
		backendConn, err := net.DialTimeout("tcp", target, 10*time.Second)
		if err != nil {
			slog.Error("wsproxy: dial backend error", "target", target, "err", err)
			// Write a simple HTTP error response to the client.
			resp := "HTTP/1.1 502 Bad Gateway\r\nContent-Length: 0\r\n\r\n"
			_, _ = clientConn.Write([]byte(resp))
			return
		}
		defer backendConn.Close()

		// Forward the original HTTP upgrade request to the backend so that
		// the backend WebSocket handshake completes correctly.
		if err := r.Write(backendConn); err != nil {
			slog.Error("wsproxy: write request to backend error", "err", err)
			return
		}

		// Bidirectional copy
		var wg sync.WaitGroup
		wg.Add(2)

		// client → backend
		go func() {
			defer wg.Done()
			io.Copy(backendConn, clientBuf) // clientBuf includes any buffered bytes from the hijacked reader
			// Half-close the backend write side so the backend knows the
			// client direction is done. TCPConn supports CloseWrite.
			if tcp, ok := backendConn.(*net.TCPConn); ok {
				_ = tcp.CloseWrite()
			}
		}()

		// backend → client
		go func() {
			defer wg.Done()
			io.Copy(clientConn, backendConn)
			if tcp, ok := clientConn.(*net.TCPConn); ok {
				_ = tcp.CloseWrite()
			}
		}()

		wg.Wait()
	}
}

// WebSocketInterceptor wraps an http.Handler so that WebSocket upgrade
// requests matching the provided checker are handled by the WebSocket proxy
// (tunnelling directly to the backend), while all other requests fall
// through to the normal HTTP handler.
func WebSocketInterceptor(target string, next http.Handler) http.Handler {
	wsHandler := WebSocketProxy(target)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if IsWebSocketRequest(r) {
			wsHandler.ServeHTTP(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}
