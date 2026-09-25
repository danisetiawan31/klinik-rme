package satusehat_test

import (
	"encoding/json"
	"testing"
	"time"

	dbgen "github.com/danisetiawan31/klinik-rme/internal/db/generated"
	"github.com/danisetiawan31/klinik-rme/internal/satusehat"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// helper: membangun EncounterInput dasar yang valid untuk dipakai di berbagai skenario test
func buildBaseEncounterInput(status string, dipanggilAt *time.Time, selesaiAt *time.Time) satusehat.EncounterInput {
	wib := time.FixedZone("WIB", 7*3600)
	createdAt := time.Date(2026, 9, 23, 8, 0, 0, 0, wib) // Pasien mendaftar jam 08:00 WIB

	pasien := dbgen.Pasien{
		ID:   1,
		Nama: "Ahmad Dhani Setiawan",
	}
	kunjungan := dbgen.Kunjungan{
		ID:               10,
		PasienID:         1,
		KlinikID:         1,
		Status:           status,
		TanggalKunjungan: pgtype.Date{Time: createdAt, Valid: true},
		CreatedAt:        pgtype.Timestamptz{Time: createdAt, Valid: true},
	}

	if dipanggilAt != nil {
		kunjungan.DipanggilAt = pgtype.Timestamptz{Time: *dipanggilAt, Valid: true}
	}
	if selesaiAt != nil {
		kunjungan.SelesaiAt = pgtype.Timestamptz{Time: *selesaiAt, Valid: true}
	} else if status == "selesai" {
		// Default selesai 08:45 WIB (30 menit setelah dipanggil 08:15)
		defaultSelesai := time.Date(2026, 9, 23, 8, 45, 0, 0, wib)
		kunjungan.SelesaiAt = pgtype.Timestamptz{Time: defaultSelesai, Valid: true}
	}

	return satusehat.EncounterInput{
		Kunjungan:     kunjungan,
		Pasien:        pasien,
		NamaDokter:    "dr. Budi Santoso Sp.PD",
		NamaKlinik:    "Klinik Umum",
		IHSPatientID:  "P02280547535",
		IHSDokterID:   "N10002345",
		IHSLocationID: "b03e2300-8888-4663-9562-b9e7c1122334",
		IHSOrgID:      "100028456",
	}
}

func TestMapKunjunganToFHIR_Success_Selesai(t *testing.T) {
	wib := time.FixedZone("WIB", 7*3600)
	dipanggilAt := time.Date(2026, 9, 23, 8, 15, 0, 0, wib)
	selesaiAt := time.Date(2026, 9, 23, 8, 45, 0, 0, wib)

	input := buildBaseEncounterInput("selesai", &dipanggilAt, &selesaiAt)

	enc, err := satusehat.MapKunjunganToFHIR(input)
	require.NoError(t, err)
	require.NotNil(t, enc)

	// ResourceType & Status
	assert.Equal(t, "Encounter", enc.ResourceType)
	assert.Equal(t, "finished", enc.Status)

	// Identifier resmi SATUSEHAT
	require.Len(t, enc.Identifier, 1)
	assert.Equal(t, "http://sys-ids.kemkes.go.id/encounter/100028456", enc.Identifier[0].System)
	assert.Equal(t, "10", enc.Identifier[0].Value)

	// Class AMB (Ambulatory)
	assert.Equal(t, "AMB", enc.Class.Code)
	assert.Equal(t, "ambulatory", enc.Class.Display)

	// Subject → Pasien
	assert.Equal(t, "Patient/P02280547535", enc.Subject.Reference)
	assert.Equal(t, "Ahmad Dhani Setiawan", enc.Subject.Display)

	// Participant → Dokter
	require.Len(t, enc.Participant, 1)
	assert.Equal(t, "Practitioner/N10002345", enc.Participant[0].Individual.Reference)
	assert.Equal(t, "dr. Budi Santoso Sp.PD", enc.Participant[0].Individual.Display)
	assert.Equal(t, "ATND", enc.Participant[0].Type[0].Coding[0].Code)

	// Location → Ruang Klinik
	require.Len(t, enc.Location, 1)
	assert.Equal(t, "Location/b03e2300-8888-4663-9562-b9e7c1122334", enc.Location[0].Location.Reference)

	// ServiceProvider → Faskes
	assert.Equal(t, "Organization/100028456", enc.ServiceProvider.Reference)

	// Period Invariant: period.start = waktu pendaftaran (08:00), period.end = waktu selesai (08:45)
	assert.Contains(t, enc.Period.Start, "08:00:00")
	assert.Contains(t, enc.Period.End, "08:45:00")

	// statusHistory: 3 tahapan kronologis
	require.Len(t, enc.StatusHistory, 3)
	// 1. arrived: 08:00 - 08:15
	assert.Equal(t, "arrived", enc.StatusHistory[0].Status)
	assert.Contains(t, enc.StatusHistory[0].Period.Start, "08:00:00")
	assert.Contains(t, enc.StatusHistory[0].Period.End, "08:15:00")
	// 2. in-progress: 08:15 - 08:45
	assert.Equal(t, "in-progress", enc.StatusHistory[1].Status)
	assert.Contains(t, enc.StatusHistory[1].Period.Start, "08:15:00")
	assert.Contains(t, enc.StatusHistory[1].Period.End, "08:45:00")
	// 3. finished: 08:45 - 08:45
	assert.Equal(t, "finished", enc.StatusHistory[2].Status)
	assert.Contains(t, enc.StatusHistory[2].Period.Start, "08:45:00")
	assert.Contains(t, enc.StatusHistory[2].Period.End, "08:45:00")
}

func TestMapKunjunganToFHIR_Success_Menunggu(t *testing.T) {
	input := buildBaseEncounterInput("menunggu", nil, nil)

	enc, err := satusehat.MapKunjunganToFHIR(input)
	require.NoError(t, err)

	assert.Equal(t, "arrived", enc.Status)
	assert.Contains(t, enc.Period.Start, "08:00:00")
	assert.Empty(t, enc.Period.End) // Belum selesai, period.end wajib kosong

	// statusHistory hanya 1 (arrived)
	require.Len(t, enc.StatusHistory, 1)
	assert.Equal(t, "arrived", enc.StatusHistory[0].Status)
	assert.Contains(t, enc.StatusHistory[0].Period.Start, "08:00:00")
	assert.Empty(t, enc.StatusHistory[0].Period.End)
}

func TestMapKunjunganToFHIR_Success_Dipanggil(t *testing.T) {
	wib := time.FixedZone("WIB", 7*3600)
	dipanggilAt := time.Date(2026, 9, 23, 8, 15, 0, 0, wib)
	input := buildBaseEncounterInput("dipanggil", &dipanggilAt, nil)

	enc, err := satusehat.MapKunjunganToFHIR(input)
	require.NoError(t, err)

	assert.Equal(t, "in-progress", enc.Status)
	assert.Contains(t, enc.Period.Start, "08:00:00")
	assert.Empty(t, enc.Period.End)

	// statusHistory ada 2 (arrived & in-progress)
	require.Len(t, enc.StatusHistory, 2)
	assert.Equal(t, "arrived", enc.StatusHistory[0].Status)
	assert.Contains(t, enc.StatusHistory[0].Period.Start, "08:00:00")
	assert.Contains(t, enc.StatusHistory[0].Period.End, "08:15:00")

	assert.Equal(t, "in-progress", enc.StatusHistory[1].Status)
	assert.Contains(t, enc.StatusHistory[1].Period.Start, "08:15:00")
	assert.Empty(t, enc.StatusHistory[1].Period.End)
}

func TestMapKunjunganToFHIR_StatusMapping(t *testing.T) {
	testCases := []struct {
		statusLokal string
		fhirStatus  string
	}{
		{"menunggu", "arrived"},
		{"dipanggil", "in-progress"},
		{"selesai", "finished"},
		{"tidak_hadir", "cancelled"},
	}

	for _, tc := range testCases {
		t.Run("status_"+tc.statusLokal, func(t *testing.T) {
			input := buildBaseEncounterInput(tc.statusLokal, nil, nil)
			enc, err := satusehat.MapKunjunganToFHIR(input)
			require.NoError(t, err)
			assert.Equal(t, tc.fhirStatus, enc.Status)
		})
	}
}

func TestMapKunjunganToFHIR_Error_MissingSelesaiAt(t *testing.T) {
	// Status selesai tanpa selesai_at harus error (mencegah payload cacat dikirim ke Kemenkes)
	input := buildBaseEncounterInput("selesai", nil, nil)
	input.Kunjungan.SelesaiAt = pgtype.Timestamptz{Valid: false}

	enc, err := satusehat.MapKunjunganToFHIR(input)
	assert.ErrorIs(t, err, satusehat.ErrMissingSelesaiAt)
	assert.Nil(t, enc)
}

func TestMapKunjunganToFHIR_Error_EndBeforeStart(t *testing.T) {
	// Waktu selesai (07:45) mendahului waktu daftar (08:00) harus ditolak
	wib := time.FixedZone("WIB", 7*3600)
	selesaiSebelumDaftar := time.Date(2026, 9, 23, 7, 45, 0, 0, wib)
	input := buildBaseEncounterInput("selesai", nil, &selesaiSebelumDaftar)

	enc, err := satusehat.MapKunjunganToFHIR(input)
	assert.ErrorIs(t, err, satusehat.ErrInvalidPeriod)
	assert.Nil(t, enc)
}

func TestMapKunjunganToFHIR_Error_MissingIHSPatientID(t *testing.T) {
	input := buildBaseEncounterInput("selesai", nil, nil)
	input.IHSPatientID = "" // dikosongkan

	enc, err := satusehat.MapKunjunganToFHIR(input)
	assert.ErrorIs(t, err, satusehat.ErrMissingIHSPatientID)
	assert.Nil(t, enc)
}

func TestMapKunjunganToFHIR_Error_MissingIHSOrgID(t *testing.T) {
	input := buildBaseEncounterInput("selesai", nil, nil)
	input.IHSOrgID = "" // dikosongkan

	enc, err := satusehat.MapKunjunganToFHIR(input)
	assert.ErrorIs(t, err, satusehat.ErrMissingIHSOrgID)
	assert.Nil(t, enc)
}

func TestMapKunjunganToFHIR_Error_InvalidStatus(t *testing.T) {
	input := buildBaseEncounterInput("status_aneh", nil, nil)

	enc, err := satusehat.MapKunjunganToFHIR(input)
	assert.ErrorIs(t, err, satusehat.ErrInvalidStatus)
	assert.Nil(t, enc)
}

func TestMapKunjunganToFHIR_NoDokter(t *testing.T) {
	input := buildBaseEncounterInput("menunggu", nil, nil)
	input.IHSDokterID = ""
	input.NamaDokter = ""

	enc, err := satusehat.MapKunjunganToFHIR(input)
	require.NoError(t, err)
	assert.Empty(t, enc.Participant)
}

func TestMapKunjunganToFHIR_JSONOutput(t *testing.T) {
	wib := time.FixedZone("WIB", 7*3600)
	dipanggilAt := time.Date(2026, 9, 23, 8, 15, 0, 0, wib)
	selesaiAt := time.Date(2026, 9, 23, 8, 45, 0, 0, wib)

	input := buildBaseEncounterInput("selesai", &dipanggilAt, &selesaiAt)

	enc, err := satusehat.MapKunjunganToFHIR(input)
	require.NoError(t, err)

	prettyJSON, err := json.MarshalIndent(enc, "", "  ")
	require.NoError(t, err)

	t.Logf("\n================ OUTPUT JSON SATUSEHAT FHIR R4 (Encounter) ================\n%s\n============================================================================", string(prettyJSON))

	jsonStr := string(prettyJSON)
	assert.Contains(t, jsonStr, `"resourceType": "Encounter"`)
	assert.Contains(t, jsonStr, `"status": "finished"`)
	assert.Contains(t, jsonStr, `"http://sys-ids.kemkes.go.id/encounter/100028456"`)
	assert.Contains(t, jsonStr, `"code": "AMB"`)
	assert.Contains(t, jsonStr, `"reference": "Patient/P02280547535"`)
	assert.Contains(t, jsonStr, `"reference": "Practitioner/N10002345"`)
	assert.Contains(t, jsonStr, `"code": "ATND"`)
	assert.Contains(t, jsonStr, `"reference": "Organization/100028456"`)
	assert.Contains(t, jsonStr, `"statusHistory"`)
	assert.Contains(t, jsonStr, `"arrived"`)
	assert.Contains(t, jsonStr, `"in-progress"`)
}
