package middleware

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var authFailures = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "gateway_auth_failures_total",
	Help: "Total number of authentication failures.",
}, []string{"reason"})

func Auth(db *sql.DB) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				authFailures.WithLabelValues("missing_key").Inc()
				writeJSON401(w, "missing api key")
				return
			}

			if !strings.HasPrefix(authHeader, "Bearer ") {
				authFailures.WithLabelValues("missing_key").Inc()
				writeJSON401(w, "invalid authorization format")
				return
			}

			token := strings.TrimPrefix(authHeader, "Bearer ")
			if token == "" {
				authFailures.WithLabelValues("missing_key").Inc()
				writeJSON401(w, "missing api key")
				return
			}

			keyHash := hashAPIKey(token)

			var keyPrefix string
			var isActive bool
			err := db.QueryRowContext(r.Context(),
				"SELECT key_prefix, is_active FROM api_keys WHERE key_hash = $1",
				keyHash,
			).Scan(&keyPrefix, &isActive)

			if err == sql.ErrNoRows {
				authFailures.WithLabelValues("invalid_key").Inc()
				writeJSON401(w, "invalid api key")
				return
			}
			if err != nil {
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}

			if !isActive {
				authFailures.WithLabelValues("inactive_key").Inc()
				writeJSON401(w, "api key is inactive")
				return
			}

			if aki := GetAPIKeyInfo(r.Context()); aki != nil {
				aki.KeyPrefix = keyPrefix
			}
			next.ServeHTTP(w, r)
		})
	}
}

func writeJSON401(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func hashAPIKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}
