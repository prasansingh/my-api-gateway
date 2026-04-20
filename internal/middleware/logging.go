package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode  int
	wroteHeader bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if rw.wroteHeader {
		return
	}
	rw.wroteHeader = true
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Logging returns a middleware that logs each request as structured JSON.
func Logging() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ri := &RouteInfo{}
			aki := &APIKeyInfo{}
			ctx := WithRouteInfo(r.Context(), ri)
			ctx = WithAPIKeyInfo(ctx, aki)
			r = r.WithContext(ctx)

			rw := &responseWriter{ResponseWriter: w, statusCode: 200}

			start := time.Now()
			next.ServeHTTP(rw, r)
			duration := time.Since(start)

			keyPrefix := aki.KeyPrefix

			slog.Info("request",
				"method", r.Method,
				"path", r.URL.Path,
				"upstream", ri.MatchedRoute,
				"duration_ms", duration.Milliseconds(),
				"status", rw.statusCode,
				"client_ip", r.RemoteAddr,
				"key_prefix", keyPrefix,
			)
		})
	}
}
