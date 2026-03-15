package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("upstream: %s %s\n", r.Method, r.URL.Path)

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
		fmt.Printf("upstream: %s %s\n", r.Method, r.URL.Path)
		time.Sleep(10 * time.Second)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "done"})
	})

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("upstream: %s %s\n", r.Method, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	fmt.Println("upstream listening on :8081")
	if err := http.ListenAndServe(":8081", mux); err != nil {
		fmt.Printf("upstream error: %v\n", err)
		os.Exit(1)
	}
}
