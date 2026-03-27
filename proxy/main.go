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

		// --- Validate REQUEST before forwarding to Service B ---
		reqBody, _ := io.ReadAll(r.Body)
		r.Body = io.NopCloser(bytes.NewBuffer(reqBody))

		fmt.Println("=== INCOMING REQUEST ===")
		fmt.Printf("Endpoint: %s %s\n", r.Method, r.URL.Path)
		fmt.Printf("Body: %s\n", string(reqBody))

		mu.RLock()
		c := contract
		mu.RUnlock()

		if c != nil && r.URL.Path == c.Endpoint && r.Method == c.Method {
			violations := validator.ValidateJSON(reqBody, c.Request, "REQUEST", c)
			if len(violations) > 0 {
				fmt.Println("REQUEST blocked — contract violations found, not forwarding to Service B")
				fmt.Println("========================")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error":      "request violates contract",
					"violations": violations,
				})
				return
			}
		}

		// --- Forward to Service B and capture response ---
		recorder := newResponseRecorder()
		proxy.ServeHTTP(recorder, r)

		// --- Validate RESPONSE before sending back to Service A ---
		fmt.Println("=== OUTGOING RESPONSE ===")
		fmt.Printf("Status: %d\n", recorder.status)
		fmt.Printf("Body: %s\n", recorder.body.String())

		if c != nil && r.URL.Path == c.Endpoint && r.Method == c.Method {
			violations := validator.ValidateJSON(recorder.body.Bytes(), c.Response, "RESPONSE", c)
			if len(violations) > 0 {
				fmt.Println("RESPONSE blocked — contract violations found, not sending to Service A")
				fmt.Println("========================")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadGateway)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error":      "response from Service B violates contract",
					"violations": violations,
				})
				return
			}
		}

		// Validation passed — send the response to Service A
		for k, v := range recorder.header {
			for _, vv := range v {
				w.Header().Add(k, vv)
			}
		}
		w.WriteHeader(recorder.status)
		w.Write(recorder.body.Bytes())

		fmt.Println("========================")
	})

	fmt.Println("Proxy running on :8080")
	http.ListenAndServe(":8080", nil)
}

// ResponseRecorder buffers the response from Service B without sending it to the client.
// This allows the proxy to validate the response before deciding to send it.
type ResponseRecorder struct {
	header http.Header
	status int
	body   *bytes.Buffer
}

func newResponseRecorder() *ResponseRecorder {
	return &ResponseRecorder{
		header: make(http.Header),
		body:   &bytes.Buffer{},
		status: http.StatusOK,
	}
}

func (r *ResponseRecorder) Header() http.Header {
	return r.header
}

func (r *ResponseRecorder) WriteHeader(status int) {
	r.status = status
}

func (r *ResponseRecorder) Write(b []byte) (int, error) {
	return r.body.Write(b)
}
