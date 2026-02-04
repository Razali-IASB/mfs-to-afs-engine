package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type Response struct {
	Message  string   `json:"message"`
	Accepted int      `json:"accepted"`
	Rejected int      `json:"rejected"`
	Errors   []string `json:"errors"`
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3001"
	}

	http.HandleFunc("/api/schedules", handleSchedules)
	http.HandleFunc("/health", handleHealth)

	log.Printf("Mock API Receiver starting on port %s...\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func handleSchedules(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read XML payload
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	xmlContent := string(body)

	// Count flights in XML (simple approach)
	flightCount := strings.Count(xmlContent, "<Flight ")

	log.Printf("Received batch with %d flights\n", flightCount)
	log.Printf("XML size: %d bytes\n", len(xmlContent))

	// Simulate processing time
	time.Sleep(100 * time.Millisecond)

	// Simulate 100% success (in production, may have failures)
	response := Response{
		Message:  "Batch processed successfully",
		Accepted: flightCount,
		Rejected: 0,
		Errors:   []string{},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)

	log.Printf("Responded with: Accepted=%d, Rejected=%d\n", response.Accepted, response.Rejected)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status":"healthy","service":"mock-api-receiver"}`)
}
