package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/api/user", func(w http.ResponseWriter, r *http.Request) {
		// Simulate serviceB responding
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "success",
			"message": "user processed",
		})
	})

	fmt.Println("ServiceB running on :8002")
	http.ListenAndServe(":8002", nil)
}
