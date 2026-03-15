package main

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	mux := http.NewServeMux()

	mux.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
		slog.Info("upstream request", "method", r.Method, "path", r.URL.Path)

		body, _ := io.ReadAll(r.Body)
		defer r.Body.Close()

		headers := make(map[string][]string)
		for k, v := range r.Header {
			headers[k] = v
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"headers": headers,
			"body":    string(body),
		})
	})

	mux.HandleFunc("/slow", func(w http.ResponseWriter, r *http.Request) {
		slog.Info("upstream request", "method", r.Method, "path", r.URL.Path)
		time.Sleep(10 * time.Second)
		slog.Info("slow request done")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "done"})
	})

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		slog.Info("upstream request", "method", r.Method, "path", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	slog.Info("upstream listening", "port", 8081)
	if err := http.ListenAndServe(":8081", mux); err != nil {
		slog.Error("upstream error", "error", err)
		os.Exit(1)
	}
}
