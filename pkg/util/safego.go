// Package util provides shared utility helpers.
package util

import "log/slog"

// SafeGo launches a goroutine with panic recovery.
// The recovered panic is logged with the provided context label.
//
// Usage:
//
//	util.SafeGo("update last_used_at", func() { repo.Update(ctx, id) })
func SafeGo(label string, fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("goroutine panic recovered",
					"label", label,
					"panic", r,
				)
			}
		}()
		fn()
	}()
}
