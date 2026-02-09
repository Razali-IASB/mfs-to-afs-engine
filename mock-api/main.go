package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
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

	// Create downloads directory if it doesn't exist
	downloadsDir := os.Getenv("DOWNLOADS_DIR")
	if downloadsDir == "" {
		downloadsDir = "./downloads"
	}
	
	if err := os.MkdirAll(downloadsDir, 0755); err != nil {
		log.Fatalf("Failed to create downloads directory: %v", err)
	}

	http.HandleFunc("/api/schedules", func(w http.ResponseWriter, r *http.Request) {
		handleSchedules(w, r, downloadsDir)
	})
	http.HandleFunc("/health", handleHealth)

	log.Printf("Mock API Receiver starting on port %s...\n", port)
	log.Printf("XML files will be saved to: %s\n", downloadsDir)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func handleSchedules(w http.ResponseWriter, r *http.Request, downloadsDir string) {
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

	// Generate filename with timestamp
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("fidasm_%s.xml", timestamp)
	filepath := filepath.Join(downloadsDir, filename)

	// Save XML to file
	err = os.WriteFile(filepath, body, 0644)
	if err != nil {
		log.Printf("ERROR: Failed to save XML file: %v\n", err)
		http.Error(w, "Failed to save XML file", http.StatusInternalServerError)
		return
	}

	log.Printf("✓ XML file saved: %s\n", filepath)


	flightCount := strings.Count(xmlContent, "<PayLoad>")
	if flightCount == 0 {
		flightCount = strings.Count(xmlContent, "<Flight ")
	}

	log.Printf("Received batch with %d flights\n", flightCount)
	log.Printf("XML size: %d bytes\n", len(xmlContent))

	time.Sleep(100 * time.Millisecond)

	response := Response{
		Message:  fmt.Sprintf("Batch processed successfully. File saved as %s", filename),
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