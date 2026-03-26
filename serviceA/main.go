package main

import (
	"fmt"
	"net/http"
	"strings"
)

func main() {
	http.HandleFunc("/send", func(w http.ResponseWriter, r *http.Request) {
		// Simulate serviceA calling serviceB through the proxy
		body := `{"user_id": "123", "email": "test@test.com", "age": "twenty"}`
		resp, err := http.Post("http://localhost:8080/api/user",
			"application/json", strings.NewReader(body))
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		defer resp.Body.Close()
		fmt.Fprintf(w, "ServiceB responded with status: %d", resp.StatusCode)
	})

	fmt.Println("ServiceA running on :8001")
	http.ListenAndServe(":8001", nil)
}
