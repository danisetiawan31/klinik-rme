package main

import (
	"fmt"
	"math/rand"
	"time"
)

type FakePatient struct {
	NIK          string
	Nama         string
	TanggalLahir time.Time
	JenisKelamin string
	Alamat       string
	NoTelp       string
}

type DiagnosisItem struct {
	KodeICD   string
	Deskripsi string
}

type TindakanItem struct {
	Jenis     string
	Deskripsi string
}

type MedicalCase struct {
	Keluhan          string
	HasilPemeriksaan string
	Diagnoses        []DiagnosisItem
	TindakanList     []TindakanItem
}

var maleFirstNames = []string{
	"Ahmad", "Budi", "Bambang", "Agus", "Hendra", "Rizki", "Eko", "Tri",
	"Dian", "Bayu", "Fajar", "Aditya", "Dimas", "Wahyu", "Pratama", "Doni",
	"Farhan", "Rian", "Ilham", "Arif", "Surya", "Indra", "Gilang", "Teguh",
}

var femaleFirstNames = []string{
	"Siti", "Dewi", "Sri", "Nur", "Ratna", "Putri", "Aisyah", "Mega",
	"Indah", "Rina", "Lestari", "Nadia", "Annisa", "Fitri", "Kartika",
	"Wulandari", "Yuni", "Maya", "Tari", "Rahma", "Intan", "Desi", "Gita",
}

var lastNames = []string{
	"Santoso", "Pratama", "Hidayat", "Saputra", "Wibowo", "Kusuma", "Lestari",
	"Wijaya", "Setiawan", "Rahmawati", "Kurniawan", "Nugroho", "Gunawan", "Utami",
	"Firmansyah", "Purnama", "Siregar", "Nasution", "Suryono", "Handayani",
}

var streetNames = []string{
	"Jl. Sudirman", "Jl. M.H. Thamrin", "Jl. Gatot Subroto", "Jl. Rasuna Said",
	"Jl. Diponegoro", "Jl. Pemuda", "Jl. Melati", "Jl. Mawar", "Jl. Kenanga",
	"Jl. Cempaka Putih", "Jl. Tebet Raya", "Jl. Fatmawati", "Jl. Margonda",
	"Jl. Raya Bogor", "Jl. Kebon Jeruk", "Jl. Kemang Raya", "Jl. Radio Dalam",
}

var priorityReasons = []string{
	"Lansia (> 60 tahun) dengan keluhan pusing berat",
	"Ibu hamil trimester 3 kontrol keluhan mual muntah",
	"Balita demam tinggi mendadak",
	"Penyandang disabilitas membutuhkan pendampingan",
	"Nyeri perut akut / kondisi darurat ringan",
}

var clinicalCases = []MedicalCase{
	{
		Keluhan:          "Demam sejak 3 hari disertai batuk berdahak dan hidung tersumbat. Tenggorokan terasa nyeri saat menelan.",
		HasilPemeriksaan: "Keadaan umum: Tampak sakit sedang. TD: 120/80 mmHg, Nadi: 82x/m, RR: 18x/m, Suhu: 38.2 C. Faring hiperemis (+), Tonsil T1-T1 tenang, Rhonki (-/-), Wheezing (-/-).",
		Diagnoses: []DiagnosisItem{
			{KodeICD: "J00", Deskripsi: "Nasofaringitis Akut (Common Cold)"},
			{KodeICD: "J02.9", Deskripsi: "Faringitis Akut Tidak Spesifik"},
		},
		TindakanList: []TindakanItem{
			{Jenis: "tindakan", Deskripsi: "Edukasi hidrasi cairan dan istirahat cukup"},
			{Jenis: "resep", Deskripsi: "Paracetamol 500mg 3x1 tab prn demam"},
			{Jenis: "resep", Deskripsi: "Ambroxol syrup 3x1 sendok takar"},
			{Jenis: "resep", Deskripsi: "Vitamin C 500mg 1x1 tab"},
		},
	},
	{
		Keluhan:          "Nyeri ulu hati seperti terbakar setelah makan pedas. Perut kembung, mual (+), muntah (-).",
		HasilPemeriksaan: "Keadaan umum: Compos mentis. TD: 110/70 mmHg, Nadi: 76x/m, RR: 16x/m, Suhu: 36.6 C. Abdomen supel, nyeri tekan epigastrium (+), bising usus normal.",
		Diagnoses: []DiagnosisItem{
			{KodeICD: "K29.7", Deskripsi: "Gastritis Tidak Spesifik"},
			{KodeICD: "K30", Deskripsi: "Dispepsia Fungsional"},
		},
		TindakanList: []TindakanItem{
			{Jenis: "tindakan", Deskripsi: "Konseling diet lambung & hindari konsumsi kopi/pedas"},
			{Jenis: "resep", Deskripsi: "Omeprazole 20mg 2x1 kapsul (30 menit ac)"},
			{Jenis: "resep", Deskripsi: "Antasida syrup 3x1 sendok takar (1 jam ac)"},
			{Jenis: "resep", Deskripsi: "Domperidone 10mg 3x1 tab prn mual"},
		},
	},
	{
		Keluhan:          "Kontrol rutin hipertensi. Mengeluhkan tengkuk terasa pegal jika kelelahan. Tidak ada keluhan sesak atau nyeri dada.",
		HasilPemeriksaan: "Keadaan umum: Baik. TD: 145/90 mmHg, Nadi: 78x/m, RR: 16x/m, Suhu: 36.4 C. Cor: S1-S2 murni reguler, murmur (-). Pulmo: Vesikuler (+/+). Ekstremitas edema (-).",
		Diagnoses: []DiagnosisItem{
			{KodeICD: "I10", Deskripsi: "Hipertensi Esensial (Primer)"},
		},
		TindakanList: []TindakanItem{
			{Jenis: "tindakan", Deskripsi: "Edukasi diet rendah garam dan olahraga aerobik 150 menit/minggu"},
			{Jenis: "resep", Deskripsi: "Amlodipine 5mg 1x1 tab pagi"},
		},
	},
	{
		Keluhan:          "BAB cair > 4x sejak pagi, berbusa tanpa darah/lendir. Badan terasa lemas dan haus terus menerus.",
		HasilPemeriksaan: "Keadaan umum: Tampak lemas. TD: 100/70 mmHg, Nadi: 88x/m, RR: 18x/m, Suhu: 37.4 C. Mata cowong (-), turgor kulit kembali cepat. Abdomen bising usus meningkat (18x/m).",
		Diagnoses: []DiagnosisItem{
			{KodeICD: "A09", Deskripsi: "Gastroenteritis Akut Tanpa Dehidrasi Berat"},
		},
		TindakanList: []TindakanItem{
			{Jenis: "tindakan", Deskripsi: "Pemberian rehidrasi oralit 200ml per BAB cair"},
			{Jenis: "resep", Deskripsi: "Oralit sachet 6 bungkus"},
			{Jenis: "resep", Deskripsi: "Zinc tablet dispersible 20mg 1x1 tab (10 hari)"},
			{Jenis: "resep", Deskripsi: "Attapulgite 600mg 2 tab per BAB cair (maks 12 tab/hari)"},
		},
	},
	{
		Keluhan:          "Pegal-pegal seluruh badan dan nyeri otot punggung bawah setelah mengangkat beban berat saat bekerja.",
		HasilPemeriksaan: "Keadaan umum: Baik. TD: 120/80 mmHg, Nadi: 74x/m, RR: 16x/m, Suhu: 36.5 C. Spasme otot paravertebral lumbal (+), Lasegue test (-), Patrick (-).",
		Diagnoses: []DiagnosisItem{
			{KodeICD: "M79.1", Deskripsi: "Myalgia / Nyeri Otot"},
			{KodeICD: "M54.5", Deskripsi: "Low Back Pain (LBP) Mekanikal"},
		},
		TindakanList: []TindakanItem{
			{Jenis: "tindakan", Deskripsi: "Edukasi postur ergonomis saat mengangkat barang"},
			{Jenis: "resep", Deskripsi: "Natrium Diklofenak 50mg 2x1 tab pc"},
			{Jenis: "resep", Deskripsi: "Vitamin B Kompleks 1x1 tab"},
		},
	},
	{
		Keluhan:          "Sakit kepala berdenyut di kedua sisi kepala seperti diikat, terutama saat sore hari setelah bekerja di depan monitor.",
		HasilPemeriksaan: "Keadaan umum: Baik. TD: 125/80 mmHg, Nadi: 76x/m, RR: 16x/m, Suhu: 36.5 C. Pemeriksaan neurologis GCS 15, refleks fisiologis normal, rangsang meningeal (-).",
		Diagnoses: []DiagnosisItem{
			{KodeICD: "G44.2", Deskripsi: "Tension-Type Headache (TTH)"},
			{KodeICD: "R51", Deskripsi: "Nyeri Kepala / Cephalgia"},
		},
		TindakanList: []TindakanItem{
			{Jenis: "tindakan", Deskripsi: "Edukasi istirahat mata berkala (metode 20-20-20)"},
			{Jenis: "resep", Deskripsi: "Ibuprofen 400mg 3x1 tab prn sakit kepala"},
		},
	},
}

// GeneratePatients generates realistic Indonesian patients with guaranteed unique NIKs.
func GeneratePatients(r *rand.Rand, count int) []FakePatient {
	patients := make([]FakePatient, count)
	provinces := []int{3171, 3172, 3173, 3174, 3175, 3271, 3273, 3374, 3578}
	seenNIK := make(map[string]bool)

	for i := 0; i < count; i++ {
		isMale := r.Intn(2) == 0
		var gender, firstName string
		if isMale {
			gender = "L"
			firstName = maleFirstNames[r.Intn(len(maleFirstNames))]
		} else {
			gender = "P"
			firstName = femaleFirstNames[r.Intn(len(femaleFirstNames))]
		}

		lastName := lastNames[r.Intn(len(lastNames))]
		fullName := fmt.Sprintf("%s %s", firstName, lastName)

		// Random birth date between 3 and 75 years ago
		ageYears := r.Intn(72) + 3
		birthDate := time.Now().AddDate(-ageYears, -r.Intn(12), -r.Intn(28))

		// Construct unique NIK 16 digit: [4 digit kota][2 digit kec][6 digit tgl/bln/thn][4 digit seri]
		prov := provinces[r.Intn(len(provinces))]
		kec := r.Intn(10) + 1
		day := birthDate.Day()
		if !isMale {
			day += 40 // Indonesian NIK convention for females
		}
		yearSuffix := birthDate.Year() % 100

		var nik string
		for {
			series := r.Intn(9000) + 1000
			nik = fmt.Sprintf("%04d%02d%02d%02d%02d%04d", prov, kec, day, int(birthDate.Month()), yearSuffix, series)
			if !seenNIK[nik] {
				seenNIK[nik] = true
				break
			}
		}

		street := streetNames[r.Intn(len(streetNames))]
		noRumah := r.Intn(150) + 1
		alamat := fmt.Sprintf("%s No. %d, RT %02d/RW %02d, Jakarta", street, noRumah, r.Intn(12)+1, r.Intn(8)+1)

		prefixes := []string{"0812", "0813", "0856", "0857", "0878", "0896"}
		noTelp := fmt.Sprintf("%s%08d", prefixes[r.Intn(len(prefixes))], r.Intn(90000000)+10000000)

		patients[i] = FakePatient{
			NIK:          nik,
			Nama:         fullName,
			TanggalLahir: birthDate,
			JenisKelamin: gender,
			Alamat:       alamat,
			NoTelp:       noTelp,
		}
	}

	return patients
}

// GetRandomMedicalCase picks a random realistic clinical case.
func GetRandomMedicalCase(r *rand.Rand) MedicalCase {
	return clinicalCases[r.Intn(len(clinicalCases))]
}

// GetRandomPriorityReason picks a priority reason for triage.
func GetRandomPriorityReason(r *rand.Rand) string {
	return priorityReasons[r.Intn(len(priorityReasons))]
}
