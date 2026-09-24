package satusehat

// FHIRIdentifier represents a FHIR Identifier datatype (e.g. NIK or IHS Number)
type FHIRIdentifier struct {
	Use    string `json:"use,omitempty"`
	System string `json:"system"`
	Value  string `json:"value"`
}

// FHIRHumanName represents a FHIR HumanName datatype
type FHIRHumanName struct {
	Use  string `json:"use,omitempty"`
	Text string `json:"text"`
}

// FHIRContactPoint represents a FHIR ContactPoint (telecom) datatype
type FHIRContactPoint struct {
	System string `json:"system"`
	Value  string `json:"value"`
	Use    string `json:"use,omitempty"`
}

// FHIRAddress represents a FHIR Address datatype
type FHIRAddress struct {
	Use  string   `json:"use,omitempty"`
	Line []string `json:"line"`
}

// FHIRPatient represents the HL7 FHIR R4 Patient Resource according to SATUSEHAT specification
type FHIRPatient struct {
	ResourceType string             `json:"resourceType"`
	Identifier   []FHIRIdentifier   `json:"identifier"`
	Active       bool               `json:"active"`
	Name         []FHIRHumanName    `json:"name"`
	Telecom      []FHIRContactPoint `json:"telecom,omitempty"`
	Gender       string             `json:"gender"`
	BirthDate    string             `json:"birthDate"`
	Address      []FHIRAddress      `json:"address,omitempty"`
}
