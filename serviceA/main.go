package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

type KYCRequest struct {
	CustomerID     string  `json:"customerId"`
	FullName       string  `json:"fullName"`
	DateOfBirth    string  `json:"dateOfBirth"`
	DocumentType   string  `json:"documentType"`
	DocumentNumber string  `json:"documentNumber"`
	Address        Address `json:"address"`
}

type Address struct {
	Street  string `json:"street"`
	City    string `json:"city"`
	Pincode string `json:"pincode"`
}

func main() {
	// POST /verify — accepts a KYC request, forwards to proxy for contract validation
	http.HandleFunc("/verify", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req KYCRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		proxyBase := os.Getenv("PROXY_URL")
		if proxyBase == "" {
			proxyBase = "http://localhost:8080"
		}

		payload, _ := json.Marshal(req)
		resp, err := http.Post(proxyBase+"/api/kyc/verify", "application/json", bytes.NewReader(payload))
		if err != nil {
			http.Error(w, "failed to reach proxy: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			http.Error(w, "failed to read proxy response", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		w.Write(respBody)
	})

	fmt.Println("KYC Verification Service running on :8001")
	http.ListenAndServe(":8001", nil)
}
