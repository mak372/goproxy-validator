package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"time"
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

type KYCResponse struct {
	CustomerID     string  `json:"customerId"`
	VerificationID string  `json:"verificationId"`
	Status         string  `json:"status"`
	RiskScore      float64 `json:"riskScore"`
	VerifiedAt     string  `json:"verifiedAt"`
}

// In-memory identity registry: documentNumber -> fullName
var registry = map[string]string{
	"DL1234567":  "Amit Sharma",
	"PAN9876543": "Priya Mehta",
	"PASS112233": "Rahul Verma",
}

func main() {
	http.HandleFunc("/api/kyc/verify", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req KYCRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		status := "rejected"
		riskScore := 85.0 + rand.Float64()*15 // high risk 85-100

		registeredName, exists := registry[req.DocumentNumber]
		if exists && registeredName == req.FullName {
			status = "verified"
			riskScore = rand.Float64() * 30 // low risk 0-30
		} else if exists {
			status = "pending"
			riskScore = 40 + rand.Float64()*30 // medium risk 40-70
		}

		resp := KYCResponse{
			CustomerID:     req.CustomerID,
			VerificationID: fmt.Sprintf("VER-%d", time.Now().UnixNano()),
			Status:         status,
			RiskScore:      riskScore,
			VerifiedAt:     time.Now().Format(time.RFC3339),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	fmt.Println("Identity Registry Service running on :8002")
	http.ListenAndServe(":8002", nil)
}
