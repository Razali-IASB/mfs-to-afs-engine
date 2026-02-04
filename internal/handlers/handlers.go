package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mh-airlines/afs-engine/internal/services"
	"github.com/mh-airlines/afs-engine/internal/utils"
	log "github.com/sirupsen/logrus"
)

// Handler manages HTTP handlers
type Handler struct {
	db        *services.Database
	generator *services.AFSGenerator
	delivery  *services.APIDelivery
	scheduler *services.Scheduler
}

// NewHandler creates a new handler
func NewHandler(db *services.Database, gen *services.AFSGenerator, del *services.APIDelivery, sched *services.Scheduler) *Handler {
	return &Handler{
		db:        db,
		generator: gen,
		delivery:  del,
		scheduler: sched,
	}
}

// HealthCheck handles health check endpoint
func (h *Handler) HealthCheck(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := h.db.HealthCheck(ctx)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "unhealthy",
			"error":   err.Error(),
			"service": "afs-engine",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "afs-engine",
		"time":    time.Now().Format(time.RFC3339),
	})
}

// GenerateAFS handles manual AFS generation request
func (h *Handler) GenerateAFS(c *gin.Context) {
	var req struct {
		Date string `json:"date" binding:"required"` // Format: YYYY-MM-DD
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Parse date
	targetDate, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format (expected YYYY-MM-DD)"})
		return
	}

	log.WithField("date", req.Date).Info("Manual AFS generation requested")

	// Trigger generation
	ctx := c.Request.Context()
	if err := h.scheduler.TriggerManualGeneration(ctx, targetDate); err != nil {
		log.WithError(err).Error("Manual generation failed")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Generation failed",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "AFS generation completed",
		"date":    req.Date,
	})
}

// GetAFSRecords retrieves AFS records for a specific date
func (h *Handler) GetAFSRecords(c *gin.Context) {
	dateStr := c.Query("date")
	if dateStr == "" {
		dateStr = utils.FormatDate(utils.GetTodayDate())
	}

	targetDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format"})
		return
	}

	status := c.DefaultQuery("status", "ALL")

	ctx := c.Request.Context()
	var records interface{}

	if status == "ALL" {
		// Get all records for date
		collection := h.db.GetCollection("active_flights")
		cursor, err := collection.Find(ctx, gin.H{"flightDate": targetDate})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer cursor.Close(ctx)

		var allRecords []interface{}
		if err := cursor.All(ctx, &allRecords); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		records = allRecords
	} else {
		// Get records by status
		afsRecords, err := h.generator.GetAFSForDelivery(ctx, targetDate, status)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		records = afsRecords
	}

	c.JSON(http.StatusOK, gin.H{
		"date":    dateStr,
		"status":  status,
		"records": records,
	})
}

// GetStats retrieves processing statistics
func (h *Handler) GetStats(c *gin.Context) {
	ctx := c.Request.Context()

	// Get counts from database
	collection := h.db.GetCollection("active_flights")
	today := utils.GetTodayDate()

	totalCount, _ := collection.CountDocuments(ctx, gin.H{"flightDate": today})
	sentCount, _ := collection.CountDocuments(ctx, gin.H{
		"flightDate":     today,
		"deliveryStatus": "SENT",
	})
	pendingCount, _ := collection.CountDocuments(ctx, gin.H{
		"flightDate":     today,
		"deliveryStatus": "PENDING",
	})
	failedCount, _ := collection.CountDocuments(ctx, gin.H{
		"flightDate":     today,
		"deliveryStatus": "FAILED",
	})

	c.JSON(http.StatusOK, gin.H{
		"date":    utils.FormatDate(today),
		"total":   totalCount,
		"sent":    sentCount,
		"pending": pendingCount,
		"failed":  failedCount,
		"successRate": func() float64 {
			if totalCount > 0 {
				return float64(sentCount) / float64(totalCount) * 100
			}
			return 0
		}(),
	})
}

// RetryFailed manually triggers retry for failed deliveries
func (h *Handler) RetryFailed(c *gin.Context) {
	ctx := c.Request.Context()
	targetDate := utils.GetTodayDate()

	log.Info("Manual retry triggered")

	deliveryStats, err := h.delivery.ProcessAllPending(ctx, targetDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Retry failed",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"stats":  deliveryStats,
	})
}
