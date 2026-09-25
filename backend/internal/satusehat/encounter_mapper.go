package satusehat

import (
	"errors"
	"fmt"
	"strings"
	"time"

	dbgen "github.com/danisetiawan31/klinik-rme/internal/db/generated"
)

var (
	ErrMissingIHSPatientID = errors.New("IHS Patient ID wajib diisi untuk pemetaan Encounter SATUSEHAT")
	ErrMissingIHSOrgID     = errors.New("IHS Organization ID wajib diisi untuk pemetaan Encounter SATUSEHAT")
	ErrInvalidStatus       = errors.New("status kunjungan tidak valid")
	ErrMissingSelesaiAt    = errors.New("kunjungan berstatus selesai wajib memiliki timestamp selesai_at")
	ErrInvalidPeriod       = errors.New("waktu selesai (period.end) tidak boleh mendahului waktu mulai (period.start)")
)

// EncounterInput adalah struct input untuk mapper Encounter.
// Karena Encounter memerlukan IHS ID dari SATUSEHAT (bukan ID integer lokal),
// data IHS ID dioper dari luar sebagai parameter — didapat setelah GET /Patient dan GET /Practitioner.
type EncounterInput struct {
	Kunjungan     dbgen.Kunjungan // Data dari tabel kunjungan lokal
	Pasien        dbgen.Pasien    // Data pasien (untuk display name di subject)
	NamaDokter    string          // Nama dokter (display name di participant)
	NamaKlinik    string          // Nama klinik (display name di location)
	IHSPatientID  string          // IHS Patient ID dari SATUSEHAT (contoh: "P02280547535")
	IHSDokterID   string          // IHS Practitioner ID dari SATUSEHAT (contoh: "N10002345")
	IHSLocationID string          // IHS Location ID klinik dari SATUSEHAT (UUID)
	IHSOrgID      string          // IHS Organization ID faskes dari SATUSEHAT (contoh: "100028456")
}

// statusToFHIR memetakan status lokal sistem klinik_rme ke status resmi HL7 FHIR R4 EncounterStatus.
//
// Penjelasan mapping:
//   - menunggu  → arrived     : Pasien sudah tiba & terdaftar di antrean faskes
//   - dipanggil → in-progress : Pasien sedang dalam sesi pemeriksaan dokter di poli
//   - selesai   → finished    : Seluruh sesi pemeriksaan dan asuhan klinis selesai
//   - tidak_hadir → cancelled : Pasien tidak merespons panggilan, antrean dibatalkan
func statusToFHIR(statusLokal string) (string, error) {
	switch statusLokal {
	case "menunggu":
		return "arrived", nil
	case "dipanggil":
		return "in-progress", nil
	case "selesai":
		return "finished", nil
	case "tidak_hadir":
		return "cancelled", nil
	default:
		return "", fmt.Errorf("%w: %s", ErrInvalidStatus, statusLokal)
	}
}

// MapKunjunganToFHIR mengubah data kunjungan lokal dan IHS ID-nya menjadi resource HL7 FHIR R4 Encounter
// yang 100% patuh pada spesifikasi interoperabilitas SATUSEHAT Kemenkes RI.
func MapKunjunganToFHIR(input EncounterInput) (*FHIREncounter, error) {
	k := input.Kunjungan

	// 1. Validasi IHS Patient ID wajib ada
	if strings.TrimSpace(input.IHSPatientID) == "" {
		return nil, ErrMissingIHSPatientID
	}

	// 2. Validasi IHS Organization ID wajib ada
	if strings.TrimSpace(input.IHSOrgID) == "" {
		return nil, ErrMissingIHSOrgID
	}

	// 3. Value Mapping: Status Lokal → FHIR EncounterStatus
	fhirStatus, err := statusToFHIR(k.Status)
	if err != nil {
		return nil, err
	}

	// 4. Kalkulasi Waktu (Period):
	// Sesuai Juknis SATUSEHAT SSRME V2.0:
	// - period.start adalah waktu kedatangan/registrasi pasien di klinik (created_at).
	// - period.end adalah waktu selesai pemeriksaan (selesai_at). Hanya terisi jika status finished/cancelled.
	var createdAtTime time.Time
	if k.CreatedAt.Valid {
		createdAtTime = k.CreatedAt.Time
	} else {
		createdAtTime = time.Now()
	}
	periodStart := createdAtTime.Format("2006-01-02T15:04:05+07:00")

	var periodEnd string
	if k.Status == "selesai" {
		if !k.SelesaiAt.Valid {
			return nil, ErrMissingSelesaiAt
		}
		if k.SelesaiAt.Time.Before(createdAtTime) {
			return nil, ErrInvalidPeriod
		}
		periodEnd = k.SelesaiAt.Time.Format("2006-01-02T15:04:05+07:00")
	} else if k.Status == "tidak_hadir" {
		if k.SelesaiAt.Valid {
			periodEnd = k.SelesaiAt.Time.Format("2006-01-02T15:04:05+07:00")
		} else {
			periodEnd = periodStart
		}
	}

	// 5. Rekam Kronologis statusHistory (Standar Wajib Siklus Hidup Encounter SATUSEHAT)
	var statusHistory []FHIREncounterStatusHistory
	switch k.Status {
	case "menunggu":
		// Fase 1: Pasien baru tiba di faskes dan mengantre
		statusHistory = []FHIREncounterStatusHistory{
			{
				Status: "arrived",
				Period: FHIRPeriod{
					Start: periodStart,
				},
			},
		}
	case "dipanggil":
		// Fase 2: Pasien dipanggil masuk ke ruang periksa poli
		var dipanggilTimeStr string
		if k.DipanggilAt.Valid {
			dipanggilTimeStr = k.DipanggilAt.Time.Format("2006-01-02T15:04:05+07:00")
		} else {
			dipanggilTimeStr = periodStart
		}
		statusHistory = []FHIREncounterStatusHistory{
			{
				Status: "arrived",
				Period: FHIRPeriod{
					Start: periodStart,
					End:   dipanggilTimeStr,
				},
			},
			{
				Status: "in-progress",
				Period: FHIRPeriod{
					Start: dipanggilTimeStr,
				},
			},
		}
	case "selesai":
		// Fase 3: Pasien telah selesai diperiksa dan RME ditutup
		var dipanggilTimeStr string
		if k.DipanggilAt.Valid {
			dipanggilTimeStr = k.DipanggilAt.Time.Format("2006-01-02T15:04:05+07:00")
		} else {
			dipanggilTimeStr = periodStart
		}
		statusHistory = []FHIREncounterStatusHistory{
			{
				Status: "arrived",
				Period: FHIRPeriod{
					Start: periodStart,
					End:   dipanggilTimeStr,
				},
			},
			{
				Status: "in-progress",
				Period: FHIRPeriod{
					Start: dipanggilTimeStr,
					End:   periodEnd,
				},
			},
			{
				Status: "finished",
				Period: FHIRPeriod{
					Start: periodEnd,
					End:   periodEnd,
				},
			},
		}
	case "tidak_hadir":
		statusHistory = []FHIREncounterStatusHistory{
			{
				Status: "arrived",
				Period: FHIRPeriod{
					Start: periodStart,
					End:   periodEnd,
				},
			},
			{
				Status: "cancelled",
				Period: FHIRPeriod{
					Start: periodEnd,
					End:   periodEnd,
				},
			},
		}
	}

	// 6. Identifier Faskes (Standar Namespace Kemenkes RI)
	var identifiers []FHIRIdentifier
	if k.ID > 0 {
		identifiers = []FHIRIdentifier{
			{
				System: fmt.Sprintf("http://sys-ids.kemkes.go.id/encounter/%s", strings.TrimSpace(input.IHSOrgID)),
				Value:  fmt.Sprintf("%d", k.ID),
			},
		}
	}

	// 7. Konstruksi Struct FHIREncounter
	encounter := &FHIREncounter{
		ResourceType:  "Encounter",
		Identifier:    identifiers,
		Status:        fhirStatus,
		StatusHistory: statusHistory,
		// class selalu AMB (ambulatory) untuk rawat jalan. Sesuai standar HL7 v3 ActCode.
		Class: FHIRCoding{
			System:  "http://terminology.hl7.org/CodeSystem/v3-ActCode",
			Code:    "AMB",
			Display: "ambulatory",
		},
		Subject: FHIRReference{
			Reference: fmt.Sprintf("Patient/%s", input.IHSPatientID),
			Display:   strings.TrimSpace(input.Pasien.Nama),
		},
		Period: FHIRPeriod{
			Start: periodStart,
			End:   periodEnd,
		},
		ServiceProvider: FHIRReference{
			Reference: fmt.Sprintf("Organization/%s", input.IHSOrgID),
		},
	}

	// 8. Participant (Dokter) — ditambahkan jika dokter sudah terassign dan IHS ID-nya tersedia
	if strings.TrimSpace(input.IHSDokterID) != "" {
		encounter.Participant = []FHIREncounterParticipantType{
			{
				Type: []FHIRCodeableConcept{
					{
						Coding: []FHIRCoding{
							{
								System:  "http://terminology.hl7.org/CodeSystem/v3-ParticipationType",
								Code:    "ATND",
								Display: "attender",
							},
						},
					},
				},
				Individual: FHIRReference{
					Reference: fmt.Sprintf("Practitioner/%s", input.IHSDokterID),
					Display:   strings.TrimSpace(input.NamaDokter),
				},
			},
		}
	}

	// 9. Location (Klinik) — ditambahkan jika IHS Location ID tersedia
	if strings.TrimSpace(input.IHSLocationID) != "" {
		encounter.Location = []FHIREncounterLocation{
			{
				Location: FHIRReference{
					Reference: fmt.Sprintf("Location/%s", input.IHSLocationID),
					Display:   strings.TrimSpace(input.NamaKlinik),
				},
			},
		}
	}

	return encounter, nil
}
