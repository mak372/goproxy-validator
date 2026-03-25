package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
)

func main() {
	// Proxy forwards all traffic to ServiceB
	target, _ := url.Parse("http://localhost:8002")
	proxy := httputil.NewSingleHostReverseProxy(target)

	// Wrap the proxy to intercept traffic
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		// --- Intercept REQUEST body ---
		reqBody, _ := io.ReadAll(r.Body)
		r.Body = io.NopCloser(bytes.NewBuffer(reqBody)) // restore body after reading

		fmt.Println("=== INCOMING REQUEST ===")
		fmt.Printf("Endpoint: %s %s\n", r.Method, r.URL.Path)
		fmt.Printf("Body: %s\n", string(reqBody))

		// --- Intercept RESPONSE body ---
		recorder := &ResponseRecorder{
			ResponseWriter: w,
			body:           &bytes.Buffer{},
		}

		proxy.ServeHTTP(recorder, r)

		fmt.Println("=== OUTGOING RESPONSE ===")
		fmt.Printf("Status: %d\n", recorder.status)
		fmt.Printf("Body: %s\n", recorder.body.String())
		fmt.Println("========================")
	})

	fmt.Println("Proxy running on :8080")
	http.ListenAndServe(":8080", nil)
}

// ResponseRecorder captures the response so we can inspect it
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
	r.body.Write(b) // capture response body
	return r.ResponseWriter.Write(b)
}
