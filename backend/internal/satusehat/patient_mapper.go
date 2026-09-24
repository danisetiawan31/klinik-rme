package satusehat

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	dbgen "github.com/danisetiawan31/klinik-rme/internal/db/generated"
)

var (
	ErrMissingNIK       = errors.New("NIK wajib diisi untuk pemetaan SATUSEHAT")
	ErrInvalidNIK       = errors.New("format NIK tidak valid, harus tepat 16 digit angka")
	ErrInvalidGender    = errors.New("jenis kelamin tidak valid, harus 'L' atau 'P'")
	ErrInvalidBirthDate = errors.New("tanggal lahir tidak valid")
	ErrMissingName      = errors.New("nama pasien wajib diisi")
)

var nikRegex = regexp.MustCompile(`^[0-9]{16}$`)

// SystemNIK adalah identifier system URI resmi Kemenkes untuk NIK Dukcapil
const SystemNIK = "https://fhir.kemkes.go.id/id/nik"

// MapPasienToFHIR mengubah data entitas lokal dbgen.Pasien menjadi resource HL7 FHIR R4 Patient
func MapPasienToFHIR(p dbgen.Pasien) (*FHIRPatient, error) {
	// 1. Validasi NIK (Wajib 16 Digit untuk SATUSEHAT)
	if !p.Nik.Valid || strings.TrimSpace(p.Nik.String) == "" {
		return nil, ErrMissingNIK
	}
	nik := strings.TrimSpace(p.Nik.String)
	if !nikRegex.MatchString(nik) {
		return nil, fmt.Errorf("%w: %s", ErrInvalidNIK, nik)
	}

	// 2. Validasi Nama
	nama := strings.TrimSpace(p.Nama)
	if nama == "" {
		return nil, ErrMissingName
	}

	// 3. Value Mapping: Jenis Kelamin ('L' -> 'male', 'P' -> 'female')
	var gender string
	switch strings.ToUpper(strings.TrimSpace(p.JenisKelamin)) {
	case "L":
		gender = "male"
	case "P":
		gender = "female"
	default:
		return nil, fmt.Errorf("%w: %s", ErrInvalidGender, p.JenisKelamin)
	}

	// 4. Format Mapping: Tanggal Lahir (ISO-8601: YYYY-MM-DD)
	if !p.TanggalLahir.Valid {
		return nil, ErrInvalidBirthDate
	}
	birthDate := p.TanggalLahir.Time.Format("2006-01-02")

	// 5. Status Keaktifan (Active = true jika belum di-soft-delete)
	active := !p.DeletedAt.Valid

	// 6. Konstruksi struct FHIR Patient
	patient := &FHIRPatient{
		ResourceType: "Patient",
		Identifier: []FHIRIdentifier{
			{
				Use:    "official",
				System: SystemNIK,
				Value:  nik,
			},
		},
		Active: active,
		Name: []FHIRHumanName{
			{
				Use:  "official",
				Text: nama,
			},
		},
		Gender:    gender,
		BirthDate: birthDate,
	}

	// 7. Mapping Nomor Telepon (Telecom) jika tersedia
	if telp := strings.TrimSpace(p.NoTelp); telp != "" {
		patient.Telecom = []FHIRContactPoint{
			{
				System: "phone",
				Value:  telp,
				Use:    "mobile",
			},
		}
	}

	// 8. Mapping Alamat Domisili jika tersedia
	if alamat := strings.TrimSpace(p.Alamat); alamat != "" {
		patient.Address = []FHIRAddress{
			{
				Use:  "home",
				Line: []string{alamat},
			},
		}
	}

	return patient, nil
}
