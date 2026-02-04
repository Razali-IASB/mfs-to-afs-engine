package main

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Station struct {
	DepartureStation         string    `bson:"DepartureStation"`
	PassengerTerminalDep     string    `bson:"passengerTerminalDep"`
	STD                      string    `bson:"std"`
	UTCLocalTimeVariationDep string    `bson:"utcLocalTimeVariationDep"`
	CD                       int       `bson:"cd"`
	ArrivalStation           string    `bson:"ArrivalStation"`
	PassengerTerminalArr     string    `bson:"passengerTerminalArr"`
	STA                      string    `bson:"sta"`
	CA                       int       `bson:"ca"`
	UTCLocalTimeVariationArr string    `bson:"utcLocalTimeVariationArr"`
	IATASubTypeCode          string    `bson:"iataSubTypeCode"`
	AircraftOwner            string    `bson:"aircraftOwner"`
	TailNo                   string    `bson:"TailNo"`
	AircraftConfiguration    string    `bson:"aircraftConfiguration"`
	OnwardFlight             string    `bson:"onwardFlight"`
	CreatedAt                time.Time `bson:"createdAt"`
	UpdatedAt                time.Time `bson:"updatedAt"`
}

type MasterFlight struct {
	ID                primitive.ObjectID `bson:"_id,omitempty"`
	CreationDate      time.Time          `bson:"creationDate"`
	FileName          string             `bson:"fileName"`
	BucketPath        string             `bson:"bucketPath"`
	FlightOwner       string             `bson:"flightOwner"`
	OperationalSuffix string             `bson:"operationalSuffix"`
	FlightNo          string             `bson:"flightNo"`
	SeasonID          string             `bson:"seasonId"`
	ItineraryVarID    int                `bson:"itineraryVarId"`
	StartDate         time.Time          `bson:"startDate"`
	EndDate           time.Time          `bson:"endDate"`
	Frequency         string             `bson:"frequency"`
	FreqRate          string             `bson:"freqRate"`
	IATAServiceType   string             `bson:"iataServiceTypeCode"`
	ScheduleStatus    string             `bson:"scheduleStatus"`
	Stations          []Station          `bson:"stations"`
	MessageType       string             `bson:"MessageType"`
	IsAdhoc           bool               `bson:"isAdhoc"`
}

func main() {
	// Connect to MongoDB
	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://admin:afs_secure_pass_2026@localhost:27017/?authSource=admin"))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(ctx)

	db := client.Database("afs_db")
	collection := db.Collection("master_flights")

	// Clear existing data
	collection.Drop(ctx)

	log.Println("Seeding MFS collection...")

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	// Sample flight schedules
	flights := []MasterFlight{
		{
			CreationDate:      now,
			FileName:          "MH0085_S26_VAR1",
			FlightOwner:       "MH",
			OperationalSuffix: "",
			FlightNo:          "MH 0085",
			SeasonID:          "S26",
			ItineraryVarID:    1,
			StartDate:         today.AddDate(0, 0, -7), // Started 7 days ago
			EndDate:           today.AddDate(0, 3, 0),  // Ends in 3 months
			Frequency:         "1234567",               // Daily
			IATAServiceType:   "J",
			ScheduleStatus:    "ACTIVE",
			MessageType:       "ASM",
			IsAdhoc:           false,
			Stations: []Station{
				{
					DepartureStation:         "KUL",
					PassengerTerminalDep:     "M",
					STD:                      "0855",
					UTCLocalTimeVariationDep: "+0800",
					CD:                       0,
					ArrivalStation:           "SIN",
					PassengerTerminalArr:     "3",
					STA:                      "0955",
					CA:                       0,
					UTCLocalTimeVariationArr: "+0800",
					IATASubTypeCode:          "73H",
					AircraftOwner:            "MH",
					TailNo:                   "9MMTK",
					AircraftConfiguration:    "12F150Y",
					CreatedAt:                now,
					UpdatedAt:                now,
				},
				{
					DepartureStation:         "SIN",
					PassengerTerminalDep:     "3",
					STD:                      "1055",
					UTCLocalTimeVariationDep: "+0800",
					CD:                       0,
					ArrivalStation:           "LHR",
					PassengerTerminalArr:     "4",
					STA:                      "1735",
					CA:                       0,
					UTCLocalTimeVariationArr: "+0000",
					IATASubTypeCode:          "359",
					AircraftOwner:            "MH",
					TailNo:                   "9MMAG",
					AircraftConfiguration:    "35J28W229Y",
					CreatedAt:                now,
					UpdatedAt:                now,
				},
			},
		},
		{
			CreationDate:      now,
			FileName:          "MH0001_S26_VAR1",
			FlightOwner:       "MH",
			OperationalSuffix: "F",
			FlightNo:          "MH 0001",
			SeasonID:          "S26",
			ItineraryVarID:    1,
			StartDate:         today.AddDate(0, 0, -7),
			EndDate:           today.AddDate(0, 3, 0),
			Frequency:         "1357", // Mon, Wed, Fri, Sun
			IATAServiceType:   "J",
			ScheduleStatus:    "ACTIVE",
			MessageType:       "ASM",
			IsAdhoc:           false,
			Stations: []Station{
				{
					DepartureStation:         "KUL",
					PassengerTerminalDep:     "M",
					STD:                      "0020",
					UTCLocalTimeVariationDep: "+0800",
					CD:                       0,
					ArrivalStation:           "DXB",
					PassengerTerminalArr:     "1",
					STA:                      "0345",
					CA:                       0,
					UTCLocalTimeVariationArr: "+0400",
					IATASubTypeCode:          "388",
					AircraftOwner:            "MH",
					TailNo:                   "9MMNA",
					AircraftConfiguration:    "8F66J420Y",
					CreatedAt:                now,
					UpdatedAt:                now,
				},
				{
					DepartureStation:         "DXB",
					PassengerTerminalDep:     "1",
					STD:                      "0445",
					UTCLocalTimeVariationDep: "+0400",
					CD:                       0,
					ArrivalStation:           "LHR",
					PassengerTerminalArr:     "4",
					STA:                      "0910",
					CA:                       0,
					UTCLocalTimeVariationArr: "+0000",
					IATASubTypeCode:          "388",
					AircraftOwner:            "MH",
					TailNo:                   "9MMNA",
					AircraftConfiguration:    "8F66J420Y",
					CreatedAt:                now,
					UpdatedAt:                now,
				},
			},
		},
		{
			CreationDate:      now,
			FileName:          "MH1111_S26_VAR1",
			FlightOwner:       "MH",
			OperationalSuffix: "Z",
			FlightNo:          "MH 1111",
			SeasonID:          "S26",
			ItineraryVarID:    1,
			StartDate:         today,
			EndDate:           today.AddDate(0, 1, 0), // 1 month
			Frequency:         "123",                  // Mon, Tue, Wed
			IATAServiceType:   "J",
			ScheduleStatus:    "ACTIVE",
			MessageType:       "ASM",
			IsAdhoc:           false,
			Stations: []Station{
				{
					DepartureStation:         "KUL",
					PassengerTerminalDep:     "1",
					STD:                      "0145",
					UTCLocalTimeVariationDep: "+0800",
					CD:                       0,
					ArrivalStation:           "PEN",
					PassengerTerminalArr:     "1",
					STA:                      "0245",
					CA:                       0,
					UTCLocalTimeVariationArr: "+0800",
					IATASubTypeCode:          "B11",
					AircraftOwner:            "MH",
					TailNo:                   "9MMA",
					AircraftConfiguration:    "28C100Y",
					CreatedAt:                now,
					UpdatedAt:                now,
				},
			},
		},
	}

	// Insert flights
	for _, flight := range flights {
		_, err := collection.InsertOne(ctx, flight)
		if err != nil {
			log.Printf("Failed to insert %s: %v\n", flight.FlightNo, err)
		} else {
			log.Printf("Inserted %s (%s)\n", flight.FlightNo, flight.Frequency)
		}
	}

	log.Printf("Seeded %d MFS records successfully\n", len(flights))
}
