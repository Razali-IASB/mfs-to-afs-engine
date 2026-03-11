package services

import (
	"context"
	"time"

	"github.com/mh-airlines/afs-engine/internal/config"
	"github.com/mh-airlines/afs-engine/internal/utils"
	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
)

// Scheduler handles cron-based job scheduling
type Scheduler struct {
	config    *config.Config
	cron      *cron.Cron
	generator *AFSGenerator
	delivery  *APIDelivery
	db        *Database
}

// NewScheduler creates a new scheduler
func NewScheduler(cfg *config.Config, gen *AFSGenerator, del *APIDelivery, db *Database) *Scheduler {
	return &Scheduler{
		config:    cfg,
		cron:      cron.New(),
		generator: gen,
		delivery:  del,
		db:        db,
	}
}

// Start starts the scheduler
func (s *Scheduler) Start() error {
	log.Info("Initializing job scheduler...")

	// Schedule daily AFS generation
	_, err := s.cron.AddFunc(s.config.Scheduler.CronSchedule, s.dailyAFSJob)
	if err != nil {
		return err
	}

	// Schedule retry job (every 15 minutes)
	_, err = s.cron.AddFunc("*/15 * * * *", s.retryFailedJob)
	if err != nil {
		return err
	}

	s.cron.Start()

	log.WithField("schedule", s.config.Scheduler.CronSchedule).Info("Job scheduler started")
	log.Info("Retry job scheduled: every 15 minutes")

	return nil
}

// dailyAFSJob is the main daily job
func (s *Scheduler) dailyAFSJob() {
	ctx := context.Background()
	startTime := time.Now()

	log.Info("========================================")
	log.Info("Starting daily AFS generation job")
	log.Info("========================================")

	defer func() {
		if r := recover(); r != nil {
			log.WithField("panic", r).Error("Daily job panicked")
		}
	}()

	targetDate := utils.GetTodayDate()

	// Phase 1 & 2: Generate AFS from MFS
	generationStats, err := s.generator.GenerateAFS(ctx, &targetDate)
	if err != nil {
		log.WithError(err).Error("AFS generation failed")
		return
	}

	log.WithFields(log.Fields{
		"mfsRecords":   generationStats.MFSRecordsQueried,
		"afsGenerated": generationStats.AFSRecordsGenerated,
		"errors":       generationStats.Errors,
		"duration":     generationStats.Duration.Seconds(),
	}).Info("Phase 1-2 completed (AFS Generation)")

	// Phase 3 & 4: Transform to XML and send to API
	deliveryStats, err := s.delivery.ProcessAllPending(ctx, targetDate)
	if err != nil {
		log.WithError(err).Error("API delivery failed")
		return
	}

	log.WithFields(log.Fields{
		"totalBatches":      deliveryStats.TotalBatches,
		"successfulBatches": deliveryStats.SuccessfulBatches,
		"failedBatches":     deliveryStats.FailedBatches,
		"deliveredFlights":  deliveryStats.DeliveredFlights,
		"failedFlights":     deliveryStats.FailedFlights,
	}).Info("Phase 3-4 completed (API Delivery)")

	// Calculate success rate
	duration := time.Since(startTime)
	successRate := 0.0
	if deliveryStats.TotalFlights > 0 {
		successRate = float64(deliveryStats.DeliveredFlights) / float64(deliveryStats.TotalFlights) * 100
	}

	log.Info("========================================")
	log.Info("Daily AFS generation job completed")
	log.WithFields(log.Fields{
		"duration":     duration.Seconds(),
		"mfsRecords":   generationStats.MFSRecordsQueried,
		"afsGenerated": generationStats.AFSRecordsGenerated,
		"delivered":    deliveryStats.DeliveredFlights,
		"total":        deliveryStats.TotalFlights,
		"successRate":  successRate,
	}).Info("Job Summary")
	log.Info("========================================")

	// Alert if success rate is low
	if deliveryStats.TotalFlights > 0 && successRate < 95 {
		log.WithField("successRate", successRate).Error("WARNING: Low delivery success rate")
		// TODO: Send alert to operations team
	}
}

// retryFailedJob retries failed deliveries
func (s *Scheduler) retryFailedJob() {
	ctx := context.Background()

	log.Info("Starting retry job for failed deliveries")

	defer func() {
		if r := recover(); r != nil {
			log.WithField("panic", r).Error("Retry job panicked")
		}
	}()

	targetDate := utils.GetTodayDate()

	// Find failed records (widen to [targetDate-1, targetDate] for local-date-aware flights)
	collection := s.db.GetCollection("active_flights")
	filter := bson.M{
		"flightDate": bson.M{
			"$gte": targetDate.AddDate(0, 0, -1),
			"$lte": targetDate,
		},
		"deliveryStatus":   "FAILED",
		"deliveryAttempts": bson.M{"$lt": s.config.API.RetryAttempts},
	}

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		log.WithError(err).Error("Failed to query failed records")
		return
	}
	defer cursor.Close(ctx)

	var failedRecords []struct {
		ID string `bson:"_id"`
	}
	if err := cursor.All(ctx, &failedRecords); err != nil {
		log.WithError(err).Error("Failed to decode failed records")
		return
	}

	if len(failedRecords) == 0 {
		log.Info("No failed records to retry")
		return
	}

	log.WithField("count", len(failedRecords)).Info("Found failed records to retry")

	// Reset status to PENDING
	failedIDs := make([]string, len(failedRecords))
	for i, record := range failedRecords {
		failedIDs[i] = record.ID
	}

	update := bson.M{
		"$set": bson.M{"deliveryStatus": "PENDING"},
	}

	_, err = collection.UpdateMany(ctx, bson.M{"_id": bson.M{"$in": failedIDs}}, update)
	if err != nil {
		log.WithError(err).Error("Failed to reset status")
		return
	}

	// Process retries
	retryStats, err := s.delivery.ProcessAllPending(ctx, targetDate)
	if err != nil {
		log.WithError(err).Error("Retry processing failed")
		return
	}

	log.WithFields(log.Fields{
		"delivered": retryStats.DeliveredFlights,
		"failed":    retryStats.FailedFlights,
	}).Info("Retry job completed")
}

// TriggerManualGeneration manually triggers AFS generation for a specific date
func (s *Scheduler) TriggerManualGeneration(ctx context.Context, targetDate time.Time) error {
	log.WithField("date", utils.FormatDate(targetDate)).Info("Manual AFS generation triggered")

	// Generate AFS
	generationStats, err := s.generator.GenerateAFS(ctx, &targetDate)
	if err != nil {
		return err
	}

	// Deliver to API
	deliveryStats, err := s.delivery.ProcessAllPending(ctx, targetDate)
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"mfsRecords":       generationStats.MFSRecordsQueried,
		"afsGenerated":     generationStats.AFSRecordsGenerated,
		"deliveredFlights": deliveryStats.DeliveredFlights,
	}).Info("Manual generation completed")

	return nil
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	if s.cron != nil {
		s.cron.Stop()
		log.Info("Job scheduler stopped")
	}
}
