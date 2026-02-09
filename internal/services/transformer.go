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

// XMLFIDASM represents the root XML element for FIDASM1 format
type XMLFIDASM struct {
	XMLName xml.Name       `xml:"XML"`
	Header  HeaderXML      `xml:"Header"`
	PayLoad []PayLoadXML   `xml:"PayLoad"`
}

// HeaderXML represents the message header
type HeaderXML struct {
	MsgCode     string `xml:"msgCode"`
	MsgSubtype  string `xml:"msgSubtype"`
	MsgVersion  string `xml:"msgVersion"`
	RefID       string `xml:"refID"`
	MsgLength   int    `xml:"msgLength"`
	MsgCount    int    `xml:"msgCount"`
	EndOfChain  int    `xml:"endOfChain"`
	MsgTimeSent string `xml:"msgTimeSent"`
}

// PayLoadXML represents a single flight payload
type PayLoadXML struct {
	Header              string `xml:"Header"`
	ActionCode          string `xml:"ActionCode"`
	AFSkey              string `xml:"AFSkey"`
	FlightNo            string `xml:"FlightNo"`
	Leg                 string `xml:"Leg"`
	STAD                string `xml:"STAD"`
	OfficialFlightDate  string `xml:"OfficialFlightDate"`
	AircraftType        string `xml:"AircraftType"`
	ServiceClass        string `xml:"ServiceClass"`
	AircraftOperator    string `xml:"AircraftOperator"`
	ServiceTypeCode     string `xml:"ServiceTypeCode"`
	CodeShareFlight     string `xml:"CodeShareFlight"`
	FlightMode          string `xml:"FlightMode"`
	ModeSequence        string `xml:"ModeSequence"`
	CategoryCode        string `xml:"CategoryCode"`
	Station1            string `xml:"Station1"`
	Station2            string `xml:"Station2"`
	Station3            string `xml:"Station3"`
	Station4            string `xml:"Station4"`
	Station5            string `xml:"Station5"`
	Station6            string `xml:"Station6"`
	STD1                string `xml:"STD1"`
	STD2                string `xml:"STD2"`
	STD3                string `xml:"STD3"`
	STD4                string `xml:"STD4"`
	STD5                string `xml:"STD5"`
	STA2                string `xml:"STA2"`
	STA3                string `xml:"STA3"`
	STA4                string `xml:"STA4"`
	STA5                string `xml:"STA5"`
	STA6                string `xml:"STA6"`
	SpFIndicator        string `xml:"SpFIndicator"`
	SchOpenTimeC        string `xml:"SchOpenTimeC"`
	SchCloseTimeC       string `xml:"SchCloseTimeC"`
	SchOpenTimeL        string `xml:"SchOpenTimeL"`
	SchCloseTimeL       string `xml:"SchCloseTimeL"`
	SchBoardTimeL       string `xml:"SchBoardTimeL"`
	SchFCTimeL          string `xml:"SchFCTimeL"`
	StandCode           string `xml:"StandCode"`
	LoungeCode          string `xml:"LoungeCode"`
	AcftRegnNo          string `xml:"AcftRegnNo"`
	Memo                string `xml:"Memo"`
	TerminalID          string `xml:"TerminalID"`
	SuffixDisp          string `xml:"SuffixDisp"`
	CheckInType         string `xml:"CheckInType"`
	IslandsAlloc        string `xml:"IslandsAlloc"`
	DeskAlloc           string `xml:"DeskAlloc"`
	IslandStatus        string `xml:"IslandStatus"`
	ActIslandOpenTime   string `xml:"ActIslandOpenTime"`
	ActIslandCloseTime  string `xml:"ActIslandCloseTime"`
}

// TransformToXML transforms AFS records to FIDASM1 XML format
func (t *XMLTransformer) TransformToXML(afsRecords []models.ActiveFlight, batchID string) (string, error) {
	log.WithFields(log.Fields{
		"recordCount": len(afsRecords),
		"batchId":     batchID,
	}).Info("Transforming AFS records to FIDASM1 XML")

	payloads := make([]PayLoadXML, len(afsRecords))
	for i, afs := range afsRecords {
		payloads[i] = t.transformFlight(afs)
	}

	// Format timestamp as YYYYMMDDHHmmss
	msgTimeSent := time.Now().Format("20060102150405")

	xmlDoc := XMLFIDASM{
		Header: HeaderXML{
			MsgCode:     "FIDASM1",
			MsgSubtype:  "",
			MsgVersion:  "1",
			RefID:       "",
			MsgLength:   0,
			MsgCount:    len(afsRecords),
			EndOfChain:  0,
			MsgTimeSent: msgTimeSent,
		},
		PayLoad: payloads,
	}

	// Marshal to XML with indentation
	output, err := xml.MarshalIndent(xmlDoc, "", "")
	if err != nil {
		return "", fmt.Errorf("failed to marshal XML: %w", err)
	}

	// Add XML declaration
	xmlString := xml.Header + string(output)

	log.WithField("batchId", batchID).Debug("FIDASM1 XML transformation completed")
	return xmlString, nil
}

// transformFlight transforms single AFS record to PayLoadXML element
func (t *XMLTransformer) transformFlight(afs models.ActiveFlight) PayLoadXML {
	// Format timestamps as YYYYMMDDHHmm
	formatTimestamp := func(dateStr, timeStr string) string {
		if dateStr == "" || timeStr == "" {
			return ""
		}
		// Assuming date format is YYYY-MM-DD and time format is HH:mm
		// Combine and format to YYYYMMDDHHmm
		combined := dateStr + timeStr
		// Remove any separators
		formatted := ""
		for _, char := range combined {
			if char >= '0' && char <= '9' {
				formatted += string(char)
			}
		}
		return formatted
	}

	// Helper to format flight date/time
	// flightDateTime := utils.FormatDate(afs.FlightDate) + afs.STD
	stad := formatTimestamp(utils.FormatDate(afs.FlightDate), afs.STD)
	std1 := formatTimestamp(utils.FormatDate(afs.FlightDate), afs.STD)
	sta2 := formatTimestamp(utils.FormatDate(afs.FlightDate), afs.STA)

	return PayLoadXML{
		Header:             "AFS",
		ActionCode:         "REV", // Default action code, adjust based on your logic
		AFSkey:             afs.ID.Hex(),
		FlightNo:           afs.FlightNo,
		Leg:                string(afs.LegSequence + 64), // Convert 1->A, 2->B, etc.
		STAD:               stad,
		OfficialFlightDate: stad,
		AircraftType:       afs.AircraftType,
		ServiceClass:       "", // Map from your model if available
		AircraftOperator:   afs.FlightOwner,
		ServiceTypeCode:    afs.ServiceType,
		CodeShareFlight:    "",
		FlightMode:         "0",
		ModeSequence:       "0",
		CategoryCode:       "I", // Default to International, adjust based on your logic
		Station1:           afs.DepartureStation,
		Station2:           afs.ArrivalStation,
		Station3:           "",
		Station4:           "",
		Station5:           "",
		Station6:           "",
		STD1:               std1,
		STD2:               "",
		STD3:               "",
		STD4:               "",
		STD5:               "",
		STA2:               sta2,
		STA3:               "",
		STA4:               "",
		STA5:               "",
		STA6:               "",
		SpFIndicator:       "N",
		SchOpenTimeC:       "", // Calculate based on STD - offset
		SchCloseTimeC:      "", // Calculate based on STD - offset
		SchOpenTimeL:       "", // Calculate based on STD - offset
		SchCloseTimeL:      "", // Calculate based on STD - offset
		SchBoardTimeL:      "", // Calculate based on STD - offset
		SchFCTimeL:         "", // Calculate based on STD - offset
		StandCode:          "",
		LoungeCode:         "",
		AcftRegnNo:         afs.TailNo,
		Memo:               "",
		TerminalID:         afs.PassengerTerminalDep,
		SuffixDisp:         "N",
		CheckInType:        "C",
		IslandsAlloc:       "", // Map from your configuration
		DeskAlloc:          "", // Map from your configuration
		IslandStatus:       "",
		ActIslandOpenTime:  "",
		ActIslandCloseTime: "",
	}
}

// CreateManifest creates batch manifest metadata
func (t *XMLTransformer) CreateManifest(batchID string, afsRecords []models.ActiveFlight, apiResponse *models.APIResponse) map[string]interface{} {
	flightIDs := make([]string, len(afsRecords))
	for i, afs := range afsRecords {
		flightIDs[i] = afs.ID.Hex()
	}

	manifest := map[string]interface{}{
		"batchId":     batchID,
		"timestamp":   time.Now().Format(time.RFC3339),
		"flightCount": len(afsRecords),
		"flightIds":   flightIDs,
		"apiStatus":   "pending",
		"format":      "FIDASM1",
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