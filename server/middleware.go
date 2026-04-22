package server

import (
	"net/http"
	"time"

	closdelog "closde-sync/internal/logging"
)

func logger() *closdelog.Logger {
	return closdelog.With("server")
}

func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		next.ServeHTTP(w, r)
		logger().Debug(
			"handled request",
			"method", r.Method,
			"path", r.URL.Path,
			"duration", time.Since(started).Round(time.Millisecond).String(),
		)
	})
}
