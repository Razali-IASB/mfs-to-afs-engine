package services

type JSONFIDASM struct {
	Header  HeaderJSON    `json:"header"`
	PayLoad []PayLoadJSON `json:"payload"`
}

type HeaderJSON struct {
	MsgCode     string `json:"msgCode"`
	MsgSubtype  string `json:"msgSubtype"`
	MsgVersion  string `json:"msgVersion"`
	RefID       string `json:"refID"`
	MsgLength   int    `json:"msgLength"`
	MsgCount    int    `json:"msgCount"`
	EndOfChain  int    `json:"endOfChain"`
	MsgTimeSent string `json:"msgTimeSent"`
}

type PayLoadJSON struct {
	Header             string   `json:"header"`
	ActionCode         string   `json:"actionCode"`
	AFSkey             int32    `json:"afsKey"`
	FlightNo           string   `json:"flightNo"`
	Leg                string   `json:"leg"`
	STAD               string   `json:"stad"`
	OfficialFlightDate string   `json:"officialFlightDate"`
	AircraftType       string   `json:"aircraftType"`
	ServiceClass       string   `json:"serviceClass"`
	AircraftOperator   string   `json:"aircraftOperator"`
	ServiceTypeCode    string   `json:"serviceTypeCode"`
	CodeShareFlight    []string `json:"codeShareFlight"`
	FlightMode         string   `json:"flightMode"`
	ModeSequence       string   `json:"modeSequence"`
	CategoryCode       string   `json:"categoryCode"`
	Station1           string   `json:"station1"`
	Station2           string   `json:"station2"`
	Station3           string   `json:"station3"`
	Station4           string   `json:"station4"`
	Station5           string   `json:"station5"`
	Station6           string   `json:"station6"`
	STD1               string   `json:"std1"`
	STD2               string   `json:"std2"`
	STD3               string   `json:"std3"`
	STD4               string   `json:"std4"`
	STD5               string   `json:"std5"`
	STA2               string   `json:"sta2"`
	STA3               string   `json:"sta3"`
	STA4               string   `json:"sta4"`
	STA5               string   `json:"sta5"`
	STA6               string   `json:"sta6"`
	SpFIndicator       string   `json:"spfIndicator"`
	SchOpenTimeC       string   `json:"schOpenTimeC"`
	SchCloseTimeC      string   `json:"schCloseTimeC"`
	SchOpenTimeL       string   `json:"schOpenTimeL"`
	SchCloseTimeL      string   `json:"schCloseTimeL"`
	SchBoardTimeL      string   `json:"schBoardTimeL"`
	SchFCTimeL         string   `json:"schFcTimeL"`
	StandCode          string   `json:"standCode"`
	LoungeCode         string   `json:"loungeCode"`
	AcftRegnNo         string   `json:"acftRegnNo"`
	Memo               string   `json:"memo"`
	TerminalID         string   `json:"terminalId"`
	SuffixDisp         string   `json:"suffixDisp"`
	CheckInType        string   `json:"checkInType"`
	IslandsAlloc       string   `json:"islandsAlloc"`
	DeskAlloc          string   `json:"deskAlloc"`
	IslandStatus       string   `json:"islandStatus"`
	ActIslandOpenTime  string   `json:"actIslandOpenTime"`
	ActIslandCloseTime string   `json:"actIslandCloseTime"`
}