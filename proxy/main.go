package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go_project/config"
	"go_project/logger"
	"go_project/validator"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
)

var (
	contract *config.Contract
	mu       sync.RWMutex
)

func main() {
	if err := logger.Init(); err != nil {
		fmt.Println("Failed to initialize logger:", err)
		return
	}
	defer logger.Log.Sync()

	// Load contract file as initial default (optional — can be overridden via POST /contract)
	if c, err := config.LoadContract("contracts/user-service.json"); err == nil {
		contract = c
		fmt.Println("Contract loaded from file for endpoint:", contract.Endpoint)
	} else {
		fmt.Println("No contract file found — POST to /contract to load one")
	}

	target, _ := url.Parse("http://localhost:8002")
	proxy := httputil.NewSingleHostReverseProxy(target)

	// POST /contract — dynamically update the contract without restarting
	http.HandleFunc("/contract", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed to read body", http.StatusBadRequest)
			return
		}
		var c config.Contract
		if err := json.Unmarshal(body, &c); err != nil {
			http.Error(w, "invalid JSON contract: "+err.Error(), http.StatusBadRequest)
			return
		}
		if c.Endpoint == "" || c.Method == "" || len(c.Request) == 0 {
			http.Error(w, "contract must have endpoint, method, and request fields", http.StatusBadRequest)
			return
		}
		mu.Lock()
		contract = &c
		mu.Unlock()
		fmt.Printf("Contract updated: %s %s\n", c.Method, c.Endpoint)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message":  "contract updated",
			"endpoint": c.Method + " " + c.Endpoint,
		})
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		// --- Validate REQUEST ---
		reqBody, _ := io.ReadAll(r.Body)
		r.Body = io.NopCloser(bytes.NewBuffer(reqBody))

		fmt.Println("=== INCOMING REQUEST ===")
		fmt.Printf("Endpoint: %s %s\n", r.Method, r.URL.Path)
		fmt.Printf("Body: %s\n", string(reqBody))

		mu.RLock()
		c := contract
		mu.RUnlock()

		if c != nil && r.URL.Path == c.Endpoint && r.Method == c.Method {
			validator.ValidateJSON(reqBody, c.Request, "REQUEST", c)
		}

		// --- Validate RESPONSE ---
		recorder := &ResponseRecorder{
			ResponseWriter: w,
			body:           &bytes.Buffer{},
		}

		proxy.ServeHTTP(recorder, r)

		fmt.Println("=== OUTGOING RESPONSE ===")
		fmt.Printf("Status: %d\n", recorder.status)
		fmt.Printf("Body: %s\n", recorder.body.String())

		if c != nil && r.URL.Path == c.Endpoint && r.Method == c.Method {
			validator.ValidateJSON(recorder.body.Bytes(), c.Response, "RESPONSE", c)
		}

		fmt.Println("========================")
	})

	fmt.Println("Proxy running on :8080")
	http.ListenAndServe(":8080", nil)
}

type ResponseRecorder struct {
	http.ResponseWriter
	status int
	body   *bytes.Buffer
}

func (r *ResponseRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *ResponseRecorder) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}
