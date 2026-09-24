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

func TestMapPasienToFHIR_Success_Male(t *testing.T) {
	dob := time.Date(1995, 4, 23, 0, 0, 0, 0, time.UTC)
	pasien := dbgen.Pasien{
		ID:           1,
		Nik:          pgtype.Text{String: "3671012304950001", Valid: true},
		Nama:         "Ahmad Dhani Setiawan",
		TanggalLahir: pgtype.Date{Time: dob, Valid: true},
		JenisKelamin: "L",
		Alamat:       "Jl. Dr. Sitanala No. 99, Tangerang",
		NoTelp:       "081234567890",
		ConsentAt:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
		Version:      1,
		DeletedAt:    pgtype.Timestamptz{Valid: false},
	}

	fhirPatient, err := satusehat.MapPasienToFHIR(pasien)
	require.NoError(t, err)
	require.NotNil(t, fhirPatient)

	assert.Equal(t, "Patient", fhirPatient.ResourceType)
	assert.True(t, fhirPatient.Active)
	assert.Equal(t, "male", fhirPatient.Gender)
	assert.Equal(t, "1995-04-23", fhirPatient.BirthDate)

	// Identifier NIK
	require.Len(t, fhirPatient.Identifier, 1)
	assert.Equal(t, "official", fhirPatient.Identifier[0].Use)
	assert.Equal(t, satusehat.SystemNIK, fhirPatient.Identifier[0].System)
	assert.Equal(t, "3671012304950001", fhirPatient.Identifier[0].Value)

	// Nama
	require.Len(t, fhirPatient.Name, 1)
	assert.Equal(t, "official", fhirPatient.Name[0].Use)
	assert.Equal(t, "Ahmad Dhani Setiawan", fhirPatient.Name[0].Text)

	// Telecom
	require.Len(t, fhirPatient.Telecom, 1)
	assert.Equal(t, "phone", fhirPatient.Telecom[0].System)
	assert.Equal(t, "mobile", fhirPatient.Telecom[0].Use)
	assert.Equal(t, "081234567890", fhirPatient.Telecom[0].Value)

	// Alamat
	require.Len(t, fhirPatient.Address, 1)
	assert.Equal(t, "home", fhirPatient.Address[0].Use)
	assert.Equal(t, []string{"Jl. Dr. Sitanala No. 99, Tangerang"}, fhirPatient.Address[0].Line)
}

func TestMapPasienToFHIR_Success_Female(t *testing.T) {
	dob := time.Date(1998, 11, 15, 0, 0, 0, 0, time.UTC)
	pasien := dbgen.Pasien{
		ID:           2,
		Nik:          pgtype.Text{String: "3201015511980002", Valid: true},
		Nama:         "Siti Rahmawati",
		TanggalLahir: pgtype.Date{Time: dob, Valid: true},
		JenisKelamin: "P",
		Alamat:       "Jl. Daan Mogot KM 20",
		NoTelp:       "087812345678",
		DeletedAt:    pgtype.Timestamptz{Valid: false},
	}

	fhirPatient, err := satusehat.MapPasienToFHIR(pasien)
	require.NoError(t, err)
	require.NotNil(t, fhirPatient)

	assert.Equal(t, "female", fhirPatient.Gender)
	assert.Equal(t, "1998-11-15", fhirPatient.BirthDate)
	assert.Equal(t, "Siti Rahmawati", fhirPatient.Name[0].Text)
}

func TestMapPasienToFHIR_MissingNIK(t *testing.T) {
	pasien := dbgen.Pasien{
		ID:           3,
		Nik:          pgtype.Text{Valid: false}, // NIK NULL
		Nama:         "Pasien Tanpa KTP",
		TanggalLahir: pgtype.Date{Time: time.Now(), Valid: true},
		JenisKelamin: "L",
	}

	fhirPatient, err := satusehat.MapPasienToFHIR(pasien)
	assert.ErrorIs(t, err, satusehat.ErrMissingNIK)
	assert.Nil(t, fhirPatient)
}

func TestMapPasienToFHIR_InvalidNIKFormat(t *testing.T) {
	testCases := []string{
		"12345",              // Kurang dari 16 digit
		"367101230495000199", // Lebih dari 16 digit
		"367101230495000A",   // Mengandung karakter non-angka
	}

	for _, tc := range testCases {
		t.Run("NIK_"+tc, func(t *testing.T) {
			pasien := dbgen.Pasien{
				ID:           4,
				Nik:          pgtype.Text{String: tc, Valid: true},
				Nama:         "Budi",
				TanggalLahir: pgtype.Date{Time: time.Now(), Valid: true},
				JenisKelamin: "L",
			}
			fhirPatient, err := satusehat.MapPasienToFHIR(pasien)
			assert.ErrorIs(t, err, satusehat.ErrInvalidNIK)
			assert.Nil(t, fhirPatient)
		})
	}
}

func TestMapPasienToFHIR_InvalidGender(t *testing.T) {
	pasien := dbgen.Pasien{
		ID:           5,
		Nik:          pgtype.Text{String: "3671012304950001", Valid: true},
		Nama:         "Anonim",
		TanggalLahir: pgtype.Date{Time: time.Now(), Valid: true},
		JenisKelamin: "X", // Bukan L maupun P
	}

	fhirPatient, err := satusehat.MapPasienToFHIR(pasien)
	assert.ErrorIs(t, err, satusehat.ErrInvalidGender)
	assert.Nil(t, fhirPatient)
}

func TestMapPasienToFHIR_InvalidBirthDate(t *testing.T) {
	pasien := dbgen.Pasien{
		ID:           6,
		Nik:          pgtype.Text{String: "3671012304950001", Valid: true},
		Nama:         "Pasien Dob Invalid",
		TanggalLahir: pgtype.Date{Valid: false},
		JenisKelamin: "L",
	}

	fhirPatient, err := satusehat.MapPasienToFHIR(pasien)
	assert.ErrorIs(t, err, satusehat.ErrInvalidBirthDate)
	assert.Nil(t, fhirPatient)
}

func TestMapPasienToFHIR_MissingName(t *testing.T) {
	pasien := dbgen.Pasien{
		ID:           7,
		Nik:          pgtype.Text{String: "3671012304950001", Valid: true},
		Nama:         "   ", // Kosong whitespace
		TanggalLahir: pgtype.Date{Time: time.Now(), Valid: true},
		JenisKelamin: "L",
	}

	fhirPatient, err := satusehat.MapPasienToFHIR(pasien)
	assert.ErrorIs(t, err, satusehat.ErrMissingName)
	assert.Nil(t, fhirPatient)
}

func TestMapPasienToFHIR_SoftDeleted(t *testing.T) {
	pasien := dbgen.Pasien{
		ID:           8,
		Nik:          pgtype.Text{String: "3671012304950001", Valid: true},
		Nama:         "Pasien Nonaktif",
		TanggalLahir: pgtype.Date{Time: time.Now(), Valid: true},
		JenisKelamin: "L",
		DeletedAt:    pgtype.Timestamptz{Time: time.Now(), Valid: true}, // Soft deleted
	}

	fhirPatient, err := satusehat.MapPasienToFHIR(pasien)
	require.NoError(t, err)
	require.NotNil(t, fhirPatient)
	assert.False(t, fhirPatient.Active, "Pasien yang memiliki deleted_at harus memiliki active = false")
}

func TestMapPasienToFHIR_JSONMarshalling(t *testing.T) {
	dob := time.Date(1995, 4, 23, 0, 0, 0, 0, time.UTC)
	pasien := dbgen.Pasien{
		ID:           1,
		Nik:          pgtype.Text{String: "3671012304950001", Valid: true},
		Nama:         "Ahmad Dhani",
		TanggalLahir: pgtype.Date{Time: dob, Valid: true},
		JenisKelamin: "L",
		Alamat:       "Jl. Dr. Sitanala No. 99",
		NoTelp:       "08123456789",
	}

	fhirPatient, err := satusehat.MapPasienToFHIR(pasien)
	require.NoError(t, err)

	// Format JSON rapi (Pretty-Print) untuk inspeksi visual
	prettyJSON, err := json.MarshalIndent(fhirPatient, "", "  ")
	require.NoError(t, err)

	t.Logf("\n================ OUTPUT JSON SATUSEHAT FHIR R4 ================\n%s\n================================================================", string(prettyJSON))

	// Pastikan field penting ada di JSON
	jsonStr := string(prettyJSON)
	assert.Contains(t, jsonStr, `"resourceType": "Patient"`)
	assert.Contains(t, jsonStr, `"system": "https://fhir.kemkes.go.id/id/nik"`)
	assert.Contains(t, jsonStr, `"value": "3671012304950001"`)
	assert.Contains(t, jsonStr, `"gender": "male"`)
	assert.Contains(t, jsonStr, `"birthDate": "1995-04-23"`)
	assert.Contains(t, jsonStr, `"text": "Ahmad Dhani"`)
	assert.Contains(t, jsonStr, `"Jl. Dr. Sitanala No. 99"`)
}
