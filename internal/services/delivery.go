package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/mh-airlines/afs-engine/internal/config"
	"github.com/mh-airlines/afs-engine/internal/models"
	"github.com/mh-airlines/afs-engine/internal/utils"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// APIDelivery handles XML delivery to downstream API
type APIDelivery struct {
	config      *config.Config
	generator   *AFSGenerator
	transformer *XMLTransformer
	httpClient  *http.Client
}

// NewAPIDelivery creates a new API delivery service
func NewAPIDelivery(cfg *config.Config, gen *AFSGenerator, trans *XMLTransformer) *APIDelivery {
	return &APIDelivery{
		config:      cfg,
		generator:   gen,
		transformer: trans,
		httpClient: &http.Client{
			Timeout: cfg.API.Timeout,
		},
	}
}

// SendBatch sends a batch of AFS records to the API
func (d *APIDelivery) SendBatch(ctx context.Context, afsRecords []models.ActiveFlight, batchID string) (*models.DeliveryResult, error) {
	result := &models.DeliveryResult{
		BatchID:      batchID,
		Success:      false,
		TotalRecords: len(afsRecords),
	}

	log.WithFields(log.Fields{
		"batchId": batchID,
		"records": len(afsRecords),
	}).Info("Sending batch to API")

	// Transform to XML
	xmlPayload, err := d.transformer.TransformToXML(afsRecords, batchID)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("XML transformation failed: %v", err))
		return result, err
	}

	// Send with retry logic
	apiResponse, err := d.sendWithRetry(ctx, xmlPayload, batchID)
	if err != nil {
		result.Errors = append(result.Errors, err.Error())

		// Update as failed - FIXED: extractIDs now returns []primitive.ObjectID
		afsIDs := extractIDs(afsRecords)
		_ = d.generator.UpdateDeliveryStatus(ctx, afsIDs, "FAILED", bson.M{
			"lastErrorMessage": err.Error(),
			"lastErrorAt":      time.Now(),
		})

		return result, err
	}

	// Parse API response
	result.APIResponse = apiResponse
	result.Success = apiResponse.StatusCode == 200
	result.AcceptedRecords = apiResponse.Accepted
	result.RejectedRecords = apiResponse.Rejected

	afsIDs := extractIDs(afsRecords)
	now := time.Now()

	updateData := bson.M{
		"deliveredAt":    now,
		"sentXMLBatchId": batchID,
		"apiResponse":    apiResponse,
	}

	if err := d.generator.UpdateDeliveryStatus(ctx, afsIDs, "SENT", updateData); err != nil {
		log.WithError(err).Error("Failed to update delivery status")
	}

	log.WithFields(log.Fields{
		"batchId":  batchID,
		"accepted": result.AcceptedRecords,
		"rejected": result.RejectedRecords,
	}).Info("Batch delivered successfully")

	// Archive XML if enabled
	if d.config.Storage.EnableXMLArchive {
		if err := d.archiveXML(batchID, xmlPayload, result); err != nil {
			log.WithError(err).Warn("Failed to archive XML")
		}
	}

	return result, nil
}

// sendWithRetry sends XML with exponential backoff retry
func (d *APIDelivery) sendWithRetry(ctx context.Context, xmlPayload, batchID string) (*models.APIResponse, error) {
	var lastErr error

	for attempt := 1; attempt <= d.config.API.RetryAttempts; attempt++ {
		log.WithFields(log.Fields{
			"batchId":     batchID,
			"attempt":     attempt,
			"maxAttempts": d.config.API.RetryAttempts,
		}).Info("Sending to API")

		resp, err := d.sendRequest(ctx, xmlPayload)
		if err == nil {
			return resp, nil
		}

		lastErr = err
		log.WithError(err).WithField("attempt", attempt).Warn("API request failed")

		// Don't retry on client errors (4xx)
		if resp != nil && resp.StatusCode >= 400 && resp.StatusCode < 500 {
			log.WithField("statusCode", resp.StatusCode).Error("Client error, not retrying")
			return resp, err
		}

		// Wait before retry (exponential backoff)
		if attempt < d.config.API.RetryAttempts {
			delay := d.calculateBackoff(attempt)
			log.WithField("delay", delay).Info("Waiting before retry")

			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}

	return nil, fmt.Errorf("all retry attempts failed: %w", lastErr)
}

// sendRequest sends HTTP POST request
func (d *APIDelivery) sendRequest(ctx context.Context, xmlPayload string) (*models.APIResponse, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		d.config.API.Endpoint,
		bytes.NewBufferString(xmlPayload),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/xml")
	req.Header.Set("User-Agent", "AFS-Engine-Go/1.0")

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response
	var apiResp struct {
		Message  string   `json:"message"`
		Accepted int      `json:"accepted"`
		Rejected int      `json:"rejected"`
		Errors   []string `json:"errors"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		// If not JSON, use simple response
		apiResp.Message = string(body)
	}

	response := &models.APIResponse{
		StatusCode: resp.StatusCode,
		Message:    apiResp.Message,
		Timestamp:  time.Now(),
		Accepted:   apiResp.Accepted,
		Rejected:   apiResp.Rejected,
		Errors:     apiResp.Errors,
	}

	if resp.StatusCode != http.StatusOK {
		return response, fmt.Errorf("API returned status %d: %s", resp.StatusCode, apiResp.Message)
	}

	return response, nil
}

// calculateBackoff calculates exponential backoff delay
func (d *APIDelivery) calculateBackoff(attempt int) time.Duration {
	return time.Duration(math.Pow(2, float64(attempt-1))) * d.config.API.RetryDelay
}

// archiveXML archives XML and manifest to file system
func (d *APIDelivery) archiveXML(batchID, xmlPayload string, result *models.DeliveryResult) error {
	now := time.Now()
	archiveDir := filepath.Join(
		d.config.Storage.ArchivePath,
		fmt.Sprintf("%d", now.Year()),
		fmt.Sprintf("%02d", now.Month()),
		fmt.Sprintf("%02d", now.Day()),
	)

	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		return fmt.Errorf("failed to create archive directory: %w", err)
	}

	// Save XML file
	xmlPath := filepath.Join(archiveDir, fmt.Sprintf("%s.xml", batchID))
	if err := os.WriteFile(xmlPath, []byte(xmlPayload), 0644); err != nil {
		return fmt.Errorf("failed to write XML file: %w", err)
	}

	// Save manifest
	manifestPath := filepath.Join(archiveDir, fmt.Sprintf("%s_manifest.json", batchID))
	manifestData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	if err := os.WriteFile(manifestPath, manifestData, 0644); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	log.WithField("path", archiveDir).Info("Batch archived successfully")
	return nil
}

// ProcessAllPending processes all pending AFS records
func (d *APIDelivery) ProcessAllPending(ctx context.Context, flightDate time.Time) (*models.DeliveryStats, error) {
	stats := &models.DeliveryStats{}

	log.Info("Starting API delivery process")

	// Get pending records
	pendingRecords, err := d.generator.GetAFSForDelivery(ctx, flightDate, "PENDING")
	if err != nil {
		return stats, fmt.Errorf("failed to get pending records: %w", err)
	}

	stats.TotalFlights = len(pendingRecords)

	if len(pendingRecords) == 0 {
		log.Info("No pending AFS records to deliver")
		return stats, nil
	}

	// Split into batches
	batches := createBatches(pendingRecords, d.config.Processing.BatchSize)
	stats.TotalBatches = len(batches)

	log.WithFields(log.Fields{
		"totalFlights": stats.TotalFlights,
		"totalBatches": stats.TotalBatches,
	}).Info("Processing batches")

	// Process each batch
	for i, batch := range batches {
		batchID := utils.GenerateBatchID(i + 1)

		result, err := d.SendBatch(ctx, batch, batchID)
		if err != nil {
			log.WithError(err).WithField("batchId", batchID).Error("Batch delivery failed")
			stats.FailedBatches++
			stats.FailedFlights += len(batch)
		} else {
			stats.SuccessfulBatches++
			stats.DeliveredFlights += result.AcceptedRecords
			stats.FailedFlights += result.RejectedRecords
		}

		// Small delay between batches
		if i < len(batches)-1 {
			time.Sleep(1 * time.Second)
		}
	}

	log.WithFields(log.Fields{
		"successfulBatches": stats.SuccessfulBatches,
		"failedBatches":     stats.FailedBatches,
		"deliveredFlights":  stats.DeliveredFlights,
		"failedFlights":     stats.FailedFlights,
	}).Info("API delivery completed")

	return stats, nil
}

// Helper functions

func extractIDs(records []models.ActiveFlight) []primitive.ObjectID {
	ids := make([]primitive.ObjectID, len(records))
	for i, record := range records {
		ids[i] = record.ID
	}
	return ids
}

func createBatches(records []models.ActiveFlight, batchSize int) [][]models.ActiveFlight {
	var batches [][]models.ActiveFlight

	for i := 0; i < len(records); i += batchSize {
		end := i + batchSize
		if end > len(records) {
			end = len(records)
		}
		batches = append(batches, records[i:end])
	}

	return batches
}