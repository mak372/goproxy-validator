package main

import (
	"bytes"
	"fmt"
	"go_project/config"
	"go_project/validator"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
)

var contract *config.Contract

func main() {
	// Load contract file
	var err error
	contract, err = config.LoadContract("contracts/user-service.json")
	if err != nil {
		fmt.Println("Failed to load contract:", err)
		return
	}
	fmt.Println("Contract loaded for endpoint:", contract.Endpoint)

	target, _ := url.Parse("http://localhost:8002")
	proxy := httputil.NewSingleHostReverseProxy(target)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		// --- Validate REQUEST ---
		reqBody, _ := io.ReadAll(r.Body)
		r.Body = io.NopCloser(bytes.NewBuffer(reqBody))

		fmt.Println("=== INCOMING REQUEST ===")
		fmt.Printf("Endpoint: %s %s\n", r.Method, r.URL.Path)
		fmt.Printf("Body: %s\n", string(reqBody))

		if r.URL.Path == contract.Endpoint && r.Method == contract.Method {
			validator.ValidateJSON(reqBody, contract.Request, "REQUEST", contract)
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

		if r.URL.Path == contract.Endpoint && r.Method == contract.Method {
			validator.ValidateJSON(recorder.body.Bytes(), contract.Response, "RESPONSE", contract)
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
