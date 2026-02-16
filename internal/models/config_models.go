package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// DefaultConfig represents the system-wide default configuration
type DefaultConfig struct {
	ID            primitive.ObjectID `bson:"_id,omitempty"`
	Gate          GateConfig         `bson:"gate"`
	CheckIn       CheckInConfig      `bson:"checkIn"`
	ShowGate      ShowGateConfig     `bson:"showGate"`
	CreatedBy     string             `bson:"createdBy"`
	CreatedAt     time.Time          `bson:"createdAt"`
	UpdatedBy     string             `bson:"updatedBy"`
	UpdatedAt     time.Time          `bson:"updatedAt"`
	Source        string             `bson:"source"`
	Description   string             `bson:"description"`
	Version       int                `bson:"version"`
	IsActive      bool               `bson:"isActive"`
	EffectiveFrom time.Time          `bson:"effectiveFrom"`
	EffectiveTo   *time.Time         `bson:"effectiveTo"`
}

// AirlineConfig represents airline-specific configuration overrides
type AirlineConfig struct {
	ID            primitive.ObjectID `bson:"_id,omitempty"`
	AirlineID     primitive.ObjectID `bson:"airlineId"`
	Gate          GateConfig         `bson:"gate"`
	CheckIn       CheckInConfig      `bson:"checkIn"`
	ShowGate      ShowGateConfig     `bson:"showGate"`
	CreatedBy     string             `bson:"createdBy"`
	CreatedAt     time.Time          `bson:"createdAt"`
	UpdatedBy     string             `bson:"updatedBy"`
	UpdatedAt     time.Time          `bson:"updatedAt"`
	Source        string             `bson:"source"`
	Description   string             `bson:"description"`
	Version       int                `bson:"version"`
	IsActive      bool               `bson:"isActive"`
	EffectiveFrom time.Time          `bson:"effectiveFrom"`
	EffectiveTo   *time.Time         `bson:"effectiveTo"`
}

// GateConfig represents gate timing configuration
type GateConfig struct {
	DomesticOpen           int `bson:"domesticOpen"`           // Minutes before STD
	DomesticBoarding       int `bson:"domesticBoarding"`       // Minutes before STD
	DomesticFinalCall      int `bson:"domesticFinalCall"`      // Minutes before STD
	DomesticClose          int `bson:"domesticClose"`          // Minutes before STD
	InternationalOpen      int `bson:"internationalOpen"`      // Minutes before STD
	InternationalBoarding  int `bson:"internationalBoarding"`  // Minutes before STD
	InternationalFinalCall int `bson:"internationalFinalCall"` // Minutes before STD
	InternationalClose     int `bson:"internationalClose"`     // Minutes before STD
}

// CheckInConfig represents check-in timing configuration
type CheckInConfig struct {
	DomesticOpen              int  `bson:"domesticOpen"`              // Minutes before STD
	DomesticClose             int  `bson:"domesticClose"`             // Minutes before STD
	InternationalOpen         int  `bson:"internationalOpen"`         // Minutes before STD
	InternationalClose        int  `bson:"internationalClose"`        // Minutes before STD
	AutoCommonCheckInOpen     bool `bson:"autoCommonCheckInOpen"`
	AutoCommonCheckInClose    bool `bson:"autoCommonCheckInClose"`
	AutoDedicatedCheckInOpen  bool `bson:"autoDedicatedCheckInOpen"`
	AutoDedicatedCheckInClose bool `bson:"autoDedicatedCheckInClose"`
}

// ShowGateConfig represents gate display configuration
type ShowGateConfig struct {
	DomesticOffset     int `bson:"domesticOffset"`     // Minutes before STD
	InternationalOffset int `bson:"internationalOffset"` // Minutes before STD
	WindowHours        int `bson:"windowHours"`
	LongDelayThreshold int `bson:"longDelayThreshold"`
}

// Airline represents airline information
type Airline struct {
	ID          primitive.ObjectID   `bson:"_id,omitempty"`
	Code        string               `bson:"code"`        // IATA code (e.g., "MH")
	ICAOCode    string               `bson:"icaoCode"`    // ICAO code (e.g., "MAS")
	Name        string               `bson:"name"`
	TerminalIDs []primitive.ObjectID `bson:"terminalIds"`
	Alliance    string               `bson:"alliance"`
	Display     bool                 `bson:"display"`
	CarrierType string               `bson:"carrierType"`
	IsActive    bool                 `bson:"isActive"`
	ShowSuffix  bool				 `bson:"showSuffix"`
}