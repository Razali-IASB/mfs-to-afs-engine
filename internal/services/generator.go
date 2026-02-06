package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mh-airlines/afs-engine/internal/config"
	"github.com/mh-airlines/afs-engine/internal/models"
	"github.com/mh-airlines/afs-engine/internal/utils"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// AFSGenerator handles AFS generation from MFS
type AFSGenerator struct {
	db     *Database
	config *config.Config
}

// NewAFSGenerator creates a new AFS generator
func NewAFSGenerator(db *Database, cfg *config.Config) *AFSGenerator {
	return &AFSGenerator{
		db:     db,
		config: cfg,
	}
}

func formatTimeToHHMM(timeStr string) string {
	return strings.ReplaceAll(timeStr, ":", "")
}

// GenerateAFS generates AFS records for target date
func (g *AFSGenerator) GenerateAFS(ctx context.Context, targetDate *time.Time) (*models.GenerationStats, error) {
	flightDate := utils.GetTodayDate()
	if targetDate != nil {
		flightDate = utils.NormalizeDate(*targetDate)
	}

	stats := &models.GenerationStats{
		StartTime: time.Now(),
	}

	log.WithField("date", utils.FormatDate(flightDate)).Info("Starting AFS generation")

	// Phase 1: Query valid MFS records
	mfsRecords, err := g.queryValidMFS(ctx, flightDate)
	if err != nil {
		return stats, fmt.Errorf("failed to query MFS: %w", err)
	}

	stats.MFSRecordsQueried = len(mfsRecords)
	log.WithField("count", len(mfsRecords)).Info("Found valid MFS records")

	// Phase 2: Process each MFS record
	for _, mfs := range mfsRecords {
		afsRecords := g.expandMFSToAFS(mfs, flightDate)

		for _, afs := range afsRecords {
			if err := g.upsertAFS(ctx, afs); err != nil {
				log.WithError(err).WithField("flightNo", mfs.FlightNo).Error("Failed to upsert AFS")
				stats.Errors++
				continue
			}
			stats.AFSRecordsGenerated++
		}
	}

	stats.EndTime = time.Now()
	stats.Duration = stats.EndTime.Sub(stats.StartTime)

	log.WithFields(log.Fields{
		"mfsRecords":   stats.MFSRecordsQueried,
		"afsGenerated": stats.AFSRecordsGenerated,
		"errors":       stats.Errors,
		"duration":     stats.Duration.Seconds(),
	}).Info("AFS generation completed")

	return stats, nil
}

// queryValidMFS queries MFS records valid for target date
func (g *AFSGenerator) queryValidMFS(ctx context.Context, targetDate time.Time) ([]models.MasterFlight, error) {
	collection := g.db.GetCollection("master_flights")

	filter := bson.M{
		"startDate":      bson.M{"$lte": targetDate},
		"endDate":        bson.M{"$gte": targetDate},
		"scheduleStatus": "ACTIVE",
	}

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	// Decode directly into MasterFlight structs - BSON tags now match MongoDB schema
	var mfsRecords []models.MasterFlight
	if err := cursor.All(ctx, &mfsRecords); err != nil {
		return nil, err
	}

	// Filter by frequency pattern
	var validRecords []models.MasterFlight
	for _, mfs := range mfsRecords {
		if utils.MatchesFrequency(targetDate, mfs.Frequency, mfs.StartDate) {
			validRecords = append(validRecords, mfs)
		}
	}

	return validRecords, nil
}

// expandMFSToAFS expands MFS record into AFS records (one per leg)
func (g *AFSGenerator) expandMFSToAFS(mfs models.MasterFlight, flightDate time.Time) []models.ActiveFlight {
	var afsRecords []models.ActiveFlight

	for i, station := range mfs.Stations {
		afsObjectID := primitive.NewObjectID()

		expiryDate := utils.CalculateExpiryDate(flightDate, g.config.Storage.AFSTTLDays)
		now := time.Now()

		afs := models.ActiveFlight{
			ID:                       afsObjectID,
			FlightNo:                 mfs.FlightNo,
			FlightOwner:              mfs.FlightOwner,
			OperationalSuffix:        mfs.OperationalSuffix,
			FlightDate:               flightDate,
			LegSequence:              i + 1,
			DepartureStation:         station.DepartureStation,
			ArrivalStation:           station.ArrivalStation,
			PassengerTerminalDep:     station.PassengerTerminalDep,
			PassengerTerminalArr:     station.PassengerTerminalArr,
			STD:                      station.STD,
			STA:                      station.STA,
			UTCLocalTimeVariationDep: station.UTCLocalTimeVariationDep,
			UTCLocalTimeVariationArr: station.UTCLocalTimeVariationArr,
			DayChangeDeparture:       station.CD,
			DayChangeArrival:         station.CA,
			AircraftType:             station.IATASubTypeCode,
			AircraftOwner:            station.AircraftOwner,
			TailNo:                   station.TailNo,
			AircraftConfiguration:    station.AircraftConfiguration,
			ServiceType:              mfs.IATAServiceType,
			OnwardFlight:             station.OnwardFlight,
			SourceMFSID:              mfs.ID,
			SeasonID:                 mfs.SeasonID,
			ItineraryVarID:           mfs.ItineraryVarID,
			DeliveryStatus:           "PENDING",
			DeliveryAttempts:         0,
			ExpiresAt:                expiryDate,
			CreatedAt:                now,
			UpdatedAt:                now,
		}

		afsRecords = append(afsRecords, afs)
	}

	return afsRecords
}

// upsertAFS inserts or updates AFS record (idempotent)
func (g *AFSGenerator) upsertAFS(ctx context.Context, afs models.ActiveFlight) error {
	collection := g.db.GetCollection("active_flights")

	filter := bson.M{"_id": afs.ID}
	
	stdFormatted := formatTimeToHHMM(afs.STD)
	staFormatted := formatTimeToHHMM(afs.STA)
	
	update := bson.M{
		"$set": bson.M{
			"flightNo":                 afs.FlightNo,
			"flightOwner":              afs.FlightOwner,
			"operationalSuffix":        afs.OperationalSuffix,
			"flightDate":               afs.FlightDate,
			"legSequence":              afs.LegSequence,
			"departureStation":         afs.DepartureStation,
			"arrivalStation":           afs.ArrivalStation,
			"passengerTerminalDep":     afs.PassengerTerminalDep,
			"passengerTerminalArr":     afs.PassengerTerminalArr,
			"std":                      stdFormatted,
			"sta":                      staFormatted,
			"utcLocalTimeVariationDep": afs.UTCLocalTimeVariationDep,
			"utcLocalTimeVariationArr": afs.UTCLocalTimeVariationArr,
			"dayChangeDeparture":       afs.DayChangeDeparture,
			"dayChangeArrival":         afs.DayChangeArrival,
			"aircraftType":             afs.AircraftType,
			"aircraftOwner":            afs.AircraftOwner,
			"tailNo":                   afs.TailNo,
			"aircraftConfiguration":    afs.AircraftConfiguration,
			"serviceType":              afs.ServiceType,
			"onwardFlight":             afs.OnwardFlight,
			"sourceMFSId":              afs.SourceMFSID,
			"seasonId":                 afs.SeasonID,
			"itineraryVarId":           afs.ItineraryVarID,
			"deliveryStatus":           afs.DeliveryStatus,
			"deliveryAttempts":         afs.DeliveryAttempts,
			"expiresAt":                afs.ExpiresAt,
			"updatedAt":                time.Now(),
		},
		"$setOnInsert": bson.M{
			"createdAt": time.Now(),
		},
	}

	opts := options.Update().SetUpsert(true)
	_, err := collection.UpdateOne(ctx, filter, update, opts)
	return err
}

// GetAFSForDelivery retrieves AFS records ready for delivery
func (g *AFSGenerator) GetAFSForDelivery(ctx context.Context, flightDate time.Time, status string) ([]models.ActiveFlight, error) {
	collection := g.db.GetCollection("active_flights")

	filter := bson.M{
		"flightDate":     flightDate,
		"deliveryStatus": status,
	}

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var records []models.ActiveFlight
	if err := cursor.All(ctx, &records); err != nil {
		return nil, err
	}

	return records, nil
}

// UpdateDeliveryStatus updates delivery status for AFS records
func (g *AFSGenerator) UpdateDeliveryStatus(ctx context.Context, afsIDs []primitive.ObjectID, status string, additionalData bson.M) error {
	collection := g.db.GetCollection("active_flights")

	filter := bson.M{"_id": bson.M{"$in": afsIDs}}

	update := bson.M{
		"$set": bson.M{
			"deliveryStatus": status,
			"updatedAt":      time.Now(),
		},
		"$inc": bson.M{"deliveryAttempts": 1},
	}

	// Merge additional data
	if additionalData != nil {
		for k, v := range additionalData {
			update["$set"].(bson.M)[k] = v
		}
	}

	_, err := collection.UpdateMany(ctx, filter, update)
	return err
}