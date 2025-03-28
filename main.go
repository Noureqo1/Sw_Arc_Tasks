package main

import (
	"circuitbreaker/circuitbreaker"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type Response struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

var cb *circuitbreaker.CircuitBreaker

func init() {
	cb = circuitbreaker.NewCircuitBreaker(circuitbreaker.Options{
		FailureThreshold:      3,
		ResetTimeout:          5 * time.Second,
		HalfOpenSuccessNeeded: 2,
		MaxRetries:            3,
		BaseRetryDelay:        100 * time.Millisecond,
		MaxRetryDelay:         2 * time.Second,
	})
}

// Simulate random failures (for testing) made following a tetorial

func simulateExternalService() error {

	if time.Now().UnixNano()%3 == 0 {
		return fmt.Errorf("external service error")
	}
	return nil
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	err := cb.Execute(func() error {
		return simulateExternalService()
	})

	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(Response{
			Status:  "error",
			Message: fmt.Sprintf("Service unavailable: %v", err),
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(Response{
		Status:  "success",
		Message: "Request processed successfully",
	})
}

func getCircuitState(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	state := cb.GetState()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"state": state,
	})
}

func main() {
	http.HandleFunc("/api/request", handleRequest)
	http.HandleFunc("/api/state", getCircuitState)

	port := ":8080"
	fmt.Printf("Server starting on port %s...\n", port)
	log.Fatal(http.ListenAndServe(port, nil))
}
