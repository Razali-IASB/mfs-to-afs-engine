package services

import (
	"encoding/xml"
	"fmt"
	"time"

	"github.com/mh-airlines/afs-engine/internal/models"
	"github.com/mh-airlines/afs-engine/internal/utils"
	log "github.com/sirupsen/logrus"
)

// XMLTransformer transforms AFS records to XML
type XMLTransformer struct{}

// NewXMLTransformer creates a new XML transformer
func NewXMLTransformer() *XMLTransformer {
	return &XMLTransformer{}
}

// FlightScheduleXML represents the root XML element
type FlightScheduleXML struct {
	XMLName     xml.Name   `xml:"FlightSchedule"`
	Version     string     `xml:"version,attr"`
	BatchID     string     `xml:"batchId,attr"`
	GeneratedAt string     `xml:"generatedAt,attr"`
	RecordCount int        `xml:"recordCount,attr"`
	Flights     FlightsXML `xml:"Flights"`
}

// FlightsXML contains the flight array
type FlightsXML struct {
	Flight []FlightXML `xml:"Flight"`
}

// FlightXML represents a single flight in XML
type FlightXML struct {
	ID                   string               `xml:"id,attr"`
	FlightIdentification FlightIdentification `xml:"FlightIdentification"`
	Route                RouteXML             `xml:"Route"`
	Aircraft             AircraftXML          `xml:"Aircraft"`
	ServiceDetails       ServiceDetailsXML    `xml:"ServiceDetails"`
	SourceTracking       SourceTrackingXML    `xml:"SourceTracking"`
}

// FlightIdentification contains flight ID info
type FlightIdentification struct {
	FlightNumber      string `xml:"FlightNumber"`
	FlightOwner       string `xml:"FlightOwner"`
	OperationalSuffix string `xml:"OperationalSuffix,omitempty"`
	FlightDate        string `xml:"FlightDate"`
	LegSequence       int    `xml:"LegSequence"`
}

// RouteXML contains departure and arrival info
type RouteXML struct {
	DepartureStation StationXML `xml:"DepartureStation"`
	ArrivalStation   StationXML `xml:"ArrivalStation"`
}

// StationXML represents a station (departure or arrival)
type StationXML struct {
	Airport       string `xml:"Airport"`
	Terminal      string `xml:"Terminal,omitempty"`
	ScheduledTime string `xml:"ScheduledTime"`
	UTCVariation  string `xml:"UTCVariation"`
	DayChange     int    `xml:"DayChange"`
}

// AircraftXML contains aircraft info
type AircraftXML struct {
	Type          string `xml:"Type"`
	Owner         string `xml:"Owner"`
	Registration  string `xml:"Registration"`
	Configuration string `xml:"Configuration"`
}

// ServiceDetailsXML contains service details
type ServiceDetailsXML struct {
	ServiceType  string `xml:"ServiceType"`
	OnwardFlight string `xml:"OnwardFlight,omitempty"`
}

// SourceTrackingXML contains source traceability
type SourceTrackingXML struct {
	SeasonID         string `xml:"SeasonId"`
	ItineraryVariant int    `xml:"ItineraryVariant"`
	SourceMFSID      string `xml:"SourceMFSId"`
}

// TransformToXML transforms AFS records to XML string
func (t *XMLTransformer) TransformToXML(afsRecords []models.ActiveFlight, batchID string) (string, error) {
	log.WithFields(log.Fields{
		"recordCount": len(afsRecords),
		"batchId":     batchID,
	}).Info("Transforming AFS records to XML")

	flights := make([]FlightXML, len(afsRecords))
	for i, afs := range afsRecords {
		flights[i] = t.transformFlight(afs)
	}

	xmlDoc := FlightScheduleXML{
		Version:     "1.0",
		BatchID:     batchID,
		GeneratedAt: time.Now().Format(time.RFC3339),
		RecordCount: len(afsRecords),
		Flights: FlightsXML{
			Flight: flights,
		},
	}

	// Marshal to XML with indentation
	output, err := xml.MarshalIndent(xmlDoc, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal XML: %w", err)
	}

	// Add XML declaration
	xmlString := xml.Header + string(output)

	log.WithField("batchId", batchID).Debug("XML transformation completed")
	return xmlString, nil
}

// transformFlight transforms single AFS record to XML flight element
func (t *XMLTransformer) transformFlight(afs models.ActiveFlight) FlightXML {
	return FlightXML{
		ID: afs.ID.Hex(),
		FlightIdentification: FlightIdentification{
			FlightNumber:      afs.FlightNo,
			FlightOwner:       afs.FlightOwner,
			OperationalSuffix: afs.OperationalSuffix,
			FlightDate:        utils.FormatDate(afs.FlightDate),
			LegSequence:       afs.LegSequence,
		},
		Route: RouteXML{
			DepartureStation: StationXML{
				Airport:       afs.DepartureStation,
				Terminal:      afs.PassengerTerminalDep,
				ScheduledTime: afs.STD,
				UTCVariation:  afs.UTCLocalTimeVariationDep,
				DayChange:     afs.DayChangeDeparture,
			},
			ArrivalStation: StationXML{
				Airport:       afs.ArrivalStation,
				Terminal:      afs.PassengerTerminalArr,
				ScheduledTime: afs.STA,
				UTCVariation:  afs.UTCLocalTimeVariationArr,
				DayChange:     afs.DayChangeArrival,
			},
		},
		Aircraft: AircraftXML{
			Type:          afs.AircraftType,
			Owner:         afs.AircraftOwner,
			Registration:  afs.TailNo,
			Configuration: afs.AircraftConfiguration,
		},
		ServiceDetails: ServiceDetailsXML{
			ServiceType:  afs.ServiceType,
			OnwardFlight: afs.OnwardFlight,
		},
		SourceTracking: SourceTrackingXML{
			SeasonID:         afs.SeasonID,
			ItineraryVariant: afs.ItineraryVarID,
			SourceMFSID:      afs.SourceMFSID.Hex(),
		},
	}
}

// CreateManifest creates batch manifest metadata
func (t *XMLTransformer) CreateManifest(batchID string, afsRecords []models.ActiveFlight, apiResponse *models.APIResponse) map[string]interface{} {
	flightIDs := make([]string, len(afsRecords))
	for i, afs := range afsRecords {
		flightIDs[i] = afs.ID.Hex() // FIXED: Convert ObjectID to string using .Hex()
	}

	manifest := map[string]interface{}{
		"batchId":     batchID,
		"timestamp":   time.Now().Format(time.RFC3339),
		"flightCount": len(afsRecords),
		"flightIds":   flightIDs,
		"apiStatus":   "pending",
	}

	if apiResponse != nil {
		manifest["apiStatus"] = "completed"
		manifest["apiResponse"] = map[string]interface{}{
			"statusCode": apiResponse.StatusCode,
			"message":    apiResponse.Message,
			"timestamp":  apiResponse.Timestamp.Format(time.RFC3339),
			"accepted":   apiResponse.Accepted,
			"rejected":   apiResponse.Rejected,
		}
	}

	return manifest
}