package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

func main() {
	// POST /send — forward dynamic data to the proxy
	// Body: any JSON payload
	// Query param: endpoint (e.g. /send?endpoint=/api/user)
	http.HandleFunc("/send", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		endpoint := r.URL.Query().Get("endpoint")
		if endpoint == "" {
			http.Error(w, "missing query param: endpoint", http.StatusBadRequest)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed to read body", http.StatusBadRequest)
			return
		}
		proxyURL := "http://localhost:8080" + endpoint
		resp, err := http.Post(proxyURL, "application/json", bytes.NewReader(body))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()
		fmt.Fprintf(w, "ServiceB responded with status: %d", resp.StatusCode)
	})

	fmt.Println("ServiceA running on :8001")
	http.ListenAndServe(":8001", nil)
}
