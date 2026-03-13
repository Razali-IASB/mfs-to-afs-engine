package services

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/mh-airlines/afs-engine/internal/models"
	log "github.com/sirupsen/logrus"
)

type JSONTransformer struct {
	rng *rand.Rand
}

func NewJSONTransformer() *JSONTransformer {
	return &JSONTransformer{
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (t *JSONTransformer) TransformToJSON(afsRecords []models.ActiveFlight, batchID string) (string, error) {
	log.WithFields(log.Fields{
		"recordCount": len(afsRecords),
		"batchId":     batchID,
	}).Info("Transforming AFS records to FIDASM1 JSON")

	payloads := make([]PayLoadJSON, len(afsRecords))
	for i, afs := range afsRecords {
		payloads[i] = t.transformFlight(afs)
	}

	msgTimeSent := time.Now().Format("20060102150405")

	jsonDoc := JSONFIDASM{
		Header: HeaderJSON{
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

	output, err := json.MarshalIndent(jsonDoc, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}

	log.WithField("batchId", batchID).Debug("FIDASM1 JSON transformation completed")
	return string(output), nil
}

// transformFlight transforms single AFS record to PayLoadJSON element
func (t *JSONTransformer) transformFlight(afs models.ActiveFlight) PayLoadJSON {
	formatTimestamp := func(flightDate time.Time, timeStr string, dayOffset int) string {
		if timeStr == "" {
			return ""
		}
		adjusted := flightDate.AddDate(0, 0, dayOffset)
		dateStr := adjusted.Format("20060102")

		combined := dateStr + timeStr
		formatted := ""
		for _, char := range combined {
			if char >= '0' && char <= '9' {
				formatted += string(char)
			}
		}
		return formatted
	}

	legValue := ""
	switch afs.MovementType {
	case "DEPARTURE":
		legValue = "D"
	case "ARRIVAL":
		legValue = "A"
	default:
		legValue = string(rune(afs.LegSequence + 64))
	}

	var stad string
	if afs.MovementType == "ARRIVAL" {
		stad = formatTimestamp(afs.FlightDate, afs.STA, afs.DayChangeArrival)
	} else {
		stad = formatTimestamp(afs.FlightDate, afs.STD, afs.DayChangeDeparture)
	}

	officialFlightDate := formatTimestamp(afs.FlightDate, afs.STD, afs.DayChangeDeparture)
	std1 := formatTimestamp(afs.FlightDate, afs.STD, afs.DayChangeDeparture)

	sta2 := formatTimestamp(afs.FlightDate, afs.STA, afs.DayChangeArrival)

	codeshareFlights := []string{}
	if len(afs.CodeshareFlights) > 0 {
		codeshareFlights = afs.CodeshareFlights
	}

	categoryCode := afs.CategoryCode
	timings := afs.OperationalTimings

	hasSuffix := afs.OperationalSuffix != ""

	combinedFlightNo := afs.FlightNo
	if hasSuffix {
		combinedFlightNo += afs.OperationalSuffix
	}

	suffixDisp := "N"
	if hasSuffix && afs.ShowSuffix {
		suffixDisp = "Y"
	}

	return PayLoadJSON{
		Header:             "AFS",
		ActionCode:         "NEW",
		AFSkey:             fmt.Sprintf("%09d", t.rng.Intn(900000000)+100000000),
		FlightNo:           combinedFlightNo,
		Leg:                legValue,
		STAD:               stad,
		OfficialFlightDate: officialFlightDate,
		AircraftType:       afs.AircraftType,
		ServiceClass:       "",
		AircraftOperator:   afs.FlightOwner,
		ServiceTypeCode:    afs.ServiceType,
		CodeShareFlight:    codeshareFlights,
		FlightMode:         "0",
		ModeSequence:       "0",
		CategoryCode:       categoryCode,
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
		SchOpenTimeC:       timings.SchOpenTimeC,
		SchCloseTimeC:      timings.SchCloseTimeC,
		SchOpenTimeL:       timings.SchOpenTimeL,
		SchCloseTimeL:      timings.SchCloseTimeL,
		SchBoardTimeL:      timings.SchBoardTimeL,
		SchFCTimeL:         timings.SchFCTimeL,
		StandCode:          "",
		LoungeCode:         "",
		AcftRegnNo:         afs.TailNo,
		Memo:               "",
		TerminalID:         afs.PassengerTerminalDep,
		SuffixDisp:         suffixDisp,
		CheckInType:        "",
		IslandsAlloc:       "",
		DeskAlloc:          "",
		IslandStatus:       "",
		ActIslandOpenTime:  "",
		ActIslandCloseTime: "",
	}
}

func (t *JSONTransformer) CreateManifest(batchID string, afsRecords []models.ActiveFlight, apiResponse *models.APIResponse) map[string]interface{} {
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