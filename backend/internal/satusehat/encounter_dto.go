package satusehat

// FHIRCoding represents a single terminology coding (system + code + display)
type FHIRCoding struct {
	System  string `json:"system"`
	Code    string `json:"code"`
	Display string `json:"display"`
}

// FHIRCodeableConcept represents a coded concept with one or more codings
type FHIRCodeableConcept struct {
	Coding []FHIRCoding `json:"coding"`
}

// FHIRReference represents a reference to another FHIR resource (e.g. Patient, Practitioner)
type FHIRReference struct {
	Reference string `json:"reference"`
	Display   string `json:"display,omitempty"`
}

// FHIREncounterParticipantType represents the role of a participant in an encounter
type FHIREncounterParticipantType struct {
	Type       []FHIRCodeableConcept `json:"type"`
	Individual FHIRReference         `json:"individual"`
}

// FHIRPeriod represents a time period with start and optional end
type FHIRPeriod struct {
	Start string `json:"start"`
	End   string `json:"end,omitempty"`
}

// FHIREncounterStatusHistory represents a history entry of an encounter status change
type FHIREncounterStatusHistory struct {
	Status string     `json:"status"`
	Period FHIRPeriod `json:"period"`
}

// FHIREncounterLocation represents the location of an encounter
type FHIREncounterLocation struct {
	Location FHIRReference `json:"location"`
}

// FHIREncounter represents the HL7 FHIR R4 Encounter Resource (Kunjungan Pelayanan)
// sesuai spesifikasi SATUSEHAT Kemenkes RI
type FHIREncounter struct {
	ResourceType    string                         `json:"resourceType"`
	Identifier      []FHIRIdentifier               `json:"identifier,omitempty"`
	Status          string                         `json:"status"`
	StatusHistory   []FHIREncounterStatusHistory   `json:"statusHistory,omitempty"`
	Class           FHIRCoding                     `json:"class"`
	Subject         FHIRReference                  `json:"subject"`
	Participant     []FHIREncounterParticipantType `json:"participant,omitempty"`
	Period          FHIRPeriod                     `json:"period"`
	Location        []FHIREncounterLocation        `json:"location,omitempty"`
	ServiceProvider FHIRReference                  `json:"serviceProvider"`
}
