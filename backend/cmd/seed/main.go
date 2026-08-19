package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/danisetiawan31/klinik-rme/internal/audit"
	"github.com/danisetiawan31/klinik-rme/internal/auth"
	"github.com/danisetiawan31/klinik-rme/internal/config"
	"github.com/danisetiawan31/klinik-rme/internal/db"
	dbgen "github.com/danisetiawan31/klinik-rme/internal/db/generated"
)

func main() {
	_ = godotenv.Load()

	patientCount := flag.Int("patients", 30, "Jumlah data pasien yang di-generate")
	queueCount := flag.Int("queue", 12, "Jumlah antrian kunjungan hari ini")
	completedCount := flag.Int("completed", 6, "Jumlah antrian selesai lengkap dengan Rekam Medis SOAP")
	clean := flag.Bool("clean", true, "Bersihkan data operasional sebelum seeding untuk mencegah duplikasi data")
	randomSeed := flag.Int64("seed", time.Now().UnixNano(), "Random seed untuk reproducibility dataset")
	flag.Parse()

	if *queueCount > *patientCount {
		*patientCount = *queueCount
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	ctx := context.Background()
	pool, err := db.NewPool(ctx, cfg.DSN())
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	defer pool.Close()

	r := rand.New(rand.NewSource(*randomSeed))

	log.Println("==================================================")
	log.Println("🚀 MENJALANKAN BULK SEEDER GENERATOR")
	log.Printf("Target : %d Pasien, %d Antrian (%d Selesai dengan SOAP)", *patientCount, *queueCount, *completedCount)
	log.Printf("Seed   : %d | Clean DB: %t", *randomSeed, *clean)
	log.Println("==================================================")

	if *clean {
		log.Println("🧹 Membersihkan data operasional lama agar tidak ada data dobel...")
		resetOperationalData(ctx, pool)
	}

	// 1. Seed / Ensure Default Clinic
	klinikID := seedKlinik(ctx, pool)

	// 2. Seed Default Staff Accounts (Admin, Dokter, Petugas)
	pwHash, err := auth.Hash("Password123!")
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}
	dokterID := seedUsers(ctx, pool, pwHash)

	// 3. Bulk Seed Patients (100% Generated, 0 Hardcode, Unique NIKs)
	pasienIDs := bulkSeedPatients(ctx, pool, r, *patientCount)

	// 4. Bulk Seed Queue & Medical Records (Distinct Patient per Queue Ticket)
	seedQueueAndRecords(ctx, pool, r, klinikID, dokterID, pasienIDs, *queueCount, *completedCount)

	log.Println("==================================================")
	log.Println("🎉 BULK SEED BERHASIL SELESAI TANPA DATA DOBEL!")
	log.Println("==================================================")
	log.Println("Akun Staf Demo (Password: Password123!):")
	log.Println("1. Admin   : admin@klinik.local   (Tata Kelola & Audit Log)")
	log.Println("2. Dokter  : dokter@klinik.local  (Workspace Klinis & EMR SOAP)")
	log.Println("3. Petugas : petugas@klinik.local (Triage & Pendaftaran Pasien)")
	log.Println("==================================================")
}

func resetOperationalData(ctx context.Context, pool *pgxpool.Pool) {
	_, err := pool.Exec(ctx, `
		TRUNCATE TABLE tindakan, diagnosis, rekam_medis, kunjungan, queue_counter, pasien, audit_log RESTART IDENTITY CASCADE;
	`)
	if err != nil {
		log.Printf("Warning saat reset tabel: %v", err)
	} else {
		// Reset audit_log_tail last_hash to 64 zeros as in initial migration
		_, _ = pool.Exec(ctx, `
			INSERT INTO audit_log_tail (id, last_hash) VALUES (1, '0000000000000000000000000000000000000000000000000000000000000000')
			ON CONFLICT (id) DO UPDATE SET last_hash = '0000000000000000000000000000000000000000000000000000000000000000';
		`)
		log.Println("✓ Tabel operasional & Audit Trail berhasil di-reset bersih.")
	}
}

func seedKlinik(ctx context.Context, pool *pgxpool.Pool) int32 {
	var klinikID int32
	displayTokenHash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" // sha256 placeholder
	err := pool.QueryRow(ctx, `
		INSERT INTO klinik (id, nama, jam_buka, jam_tutup, display_token_hash)
		VALUES (1, 'Klinik Sehat Jaya', '08:00', '20:00', $1)
		ON CONFLICT (id) DO UPDATE SET
			nama = EXCLUDED.nama,
			jam_buka = EXCLUDED.jam_buka,
			jam_tutup = EXCLUDED.jam_tutup
		RETURNING id
	`, displayTokenHash).Scan(&klinikID)
	if err != nil {
		log.Fatalf("Failed to seed klinik: %v", err)
	}
	log.Printf("✓ Klinik: 'Klinik Sehat Jaya' (ID: %d, Jam: 08:00 - 20:00)", klinikID)
	return klinikID
}

func seedUsers(ctx context.Context, pool *pgxpool.Pool, pwHash string) int32 {
	users := []struct {
		nama  string
		email string
		role  string
	}{
		{"Administrator", "admin@klinik.local", "admin"},
		{"dr. Budi Santoso", "dokter@klinik.local", "dokter"},
		{"Siti Rahmawati", "petugas@klinik.local", "petugas"},
	}

	var dokterID int32
	for _, u := range users {
		var userID int32
		err := pool.QueryRow(ctx, `
			INSERT INTO users (nama, email, password_hash)
			VALUES ($1, $2, $3)
			ON CONFLICT (email) DO UPDATE 
			SET password_hash = EXCLUDED.password_hash, nama = EXCLUDED.nama
			RETURNING id
		`, u.nama, u.email, pwHash).Scan(&userID)
		if err != nil {
			log.Fatalf("Failed to upsert user %s: %v", u.email, err)
		}

		_, err = pool.Exec(ctx, `
			INSERT INTO user_roles (user_id, role)
			VALUES ($1, $2)
			ON CONFLICT (user_id, role) DO NOTHING
		`, userID, u.role)
		if err != nil {
			log.Fatalf("Failed to insert role for user %s: %v", u.email, err)
		}

		if u.role == "dokter" {
			dokterID = userID
		}

		log.Printf("✓ User Akun: %s (%s) [Role: %s]", u.nama, u.email, u.role)
	}
	return dokterID
}

func bulkSeedPatients(ctx context.Context, pool *pgxpool.Pool, r *rand.Rand, count int) []int32 {
	// Generate 100% dynamic unique patients via factory (0 hardcoding)
	generated := GeneratePatients(r, count)
	rows := make([][]any, len(generated))
	for i, p := range generated {
		rows[i] = []any{
			p.NIK,
			p.Nama,
			pgtype.Date{Time: p.TanggalLahir, Valid: true},
			p.JenisKelamin,
			p.Alamat,
			p.NoTelp,
			time.Now(),
			1,
		}
	}

	copyCount, err := pool.CopyFrom(
		ctx,
		pgx.Identifier{"pasien"},
		[]string{"nik", "nama", "tanggal_lahir", "jenis_kelamin", "alamat", "no_telp", "consent_at", "version"},
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		log.Printf("Gagal copy pasien via CopyFrom: %v, fallback ke batch insert...", err)
		for _, p := range generated {
			_, _ = pool.Exec(ctx, `
				INSERT INTO pasien (nik, nama, tanggal_lahir, jenis_kelamin, alamat, no_telp, consent_at, version)
				VALUES ($1, $2, $3, $4, $5, $6, NOW(), 1)
			`, p.NIK, p.Nama, pgtype.Date{Time: p.TanggalLahir, Valid: true}, p.JenisKelamin, p.Alamat, p.NoTelp)
		}
	} else {
		log.Printf("✓ Berhasil memuat %d pasien unik via Postgres COPY protocol!", copyCount)
	}

	// Fetch all IDs in insertion order
	dbRows, err := pool.Query(ctx, `SELECT id FROM pasien WHERE deleted_at IS NULL ORDER BY id ASC`)
	if err != nil {
		log.Fatalf("Failed to read patient IDs: %v", err)
	}
	defer dbRows.Close()

	var IDs []int32
	for dbRows.Next() {
		var id int32
		if err := dbRows.Scan(&id); err == nil {
			IDs = append(IDs, id)
		}
	}

	log.Printf("✓ Total Pasien Terdaftar: %d pasien unik", len(IDs))
	return IDs
}

func seedQueueAndRecords(
	ctx context.Context,
	pool *pgxpool.Pool,
	r *rand.Rand,
	klinikID int32,
	dokterID int32,
	pasienIDs []int32,
	queueCount int,
	completedCount int,
) {
	if len(pasienIDs) == 0 {
		return
	}

	today := time.Now()
	todayDate := pgtype.Date{Time: today, Valid: true}

	if queueCount > len(pasienIDs) {
		queueCount = len(pasienIDs)
	}

	if completedCount > queueCount {
		completedCount = queueCount
	}

	q := dbgen.New(pool)

	log.Printf("⚡ Menerbitkan %d Antrian Hari Ini (%s) tanpa duplikasi...", queueCount, today.Format("2006-01-02"))

	var activeCounter int
	for i := 1; i <= queueCount; i++ {
		activeCounter = i
		// Strictly distinct patient per ticket
		pasienID := pasienIDs[i-1]

		isPriority := (i%4 == 0) // setiap kelipatan 4 adalah prioritas
		var priorityReason *string
		if isPriority {
			reason := GetRandomPriorityReason(r)
			priorityReason = &reason
		}

		var status string
		var dipanggilAt *time.Time
		if i <= completedCount {
			status = "selesai"
			callTime := today.Add(time.Duration(-45*(queueCount-i+1)) * time.Minute)
			dipanggilAt = &callTime
		} else if i == completedCount+1 {
			status = "dipanggil"
			callTime := today.Add(-5 * time.Minute)
			dipanggilAt = &callTime
		} else {
			status = "menunggu"
		}

		var kunjunganID int32
		err := pool.QueryRow(ctx, `
			INSERT INTO kunjungan (pasien_id, klinik_id, dokter_id, tanggal_kunjungan, nomor_antrian, is_priority, priority_reason, status, dipanggil_at, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			RETURNING id
		`, pasienID, klinikID, dokterID, todayDate, i, isPriority, priorityReason, status, dipanggilAt, today.Add(time.Duration(-100+i)*time.Minute)).Scan(&kunjunganID)
		if err != nil {
			log.Printf("Failed to insert kunjungan %d: %v", i, err)
			continue
		}

		// If status is 'selesai', create full SOAP Medical Record + Diagnosis + Tindakan + SHA256 Audit Trail
		if status == "selesai" {
			createMedicalRecordWithAudit(ctx, pool, q, r, kunjunganID, dokterID, pasienID)
		}
	}

	// Update queue_counter table atomically
	_, _ = pool.Exec(ctx, `
		INSERT INTO queue_counter (klinik_id, tanggal, last_counter)
		VALUES ($1, $2, $3)
		ON CONFLICT (klinik_id, tanggal) DO UPDATE
		SET last_counter = GREATEST(queue_counter.last_counter, EXCLUDED.last_counter)
	`, klinikID, todayDate, activeCounter)

	log.Printf("✓ Queue Counter di-set ke: %d", activeCounter)
	log.Printf("✓ %d Kunjungan dibuat: %d Selesai (dengan SOAP & Audit Log), 1 Sedang Dipanggil, %d Menunggu",
		queueCount, completedCount, queueCount-completedCount-1)
}

func createMedicalRecordWithAudit(
	ctx context.Context,
	pool *pgxpool.Pool,
	q *dbgen.Queries,
	r *rand.Rand,
	kunjunganID int32,
	dokterID int32,
	pasienID int32,
) {
	medCase := GetRandomMedicalCase(r)

	// Execute inside explicit database transaction for atomic audit trail logging
	tx, err := pool.Begin(ctx)
	if err != nil {
		log.Printf("Failed to begin transaction for rekam medis: %v", err)
		return
	}
	defer tx.Rollback(ctx)

	var rmID int32
	err = tx.QueryRow(ctx, `
		INSERT INTO rekam_medis (kunjungan_id, dokter_id, keluhan, hasil_pemeriksaan, created_at)
		VALUES ($1, $2, $3, $4, NOW())
		RETURNING id
	`, kunjunganID, dokterID, medCase.Keluhan, medCase.HasilPemeriksaan).Scan(&rmID)
	if err != nil {
		log.Printf("Failed to insert rekam medis: %v", err)
		return
	}

	// Insert Diagnoses
	for _, d := range medCase.Diagnoses {
		_, _ = tx.Exec(ctx, `
			INSERT INTO diagnosis (rekam_medis_id, kode_icd, deskripsi)
			VALUES ($1, $2, $3)
		`, rmID, d.KodeICD, d.Deskripsi)
	}

	// Insert Tindakan & Resep
	for _, t := range medCase.TindakanList {
		_, _ = tx.Exec(ctx, `
			INSERT INTO tindakan (rekam_medis_id, jenis, deskripsi)
			VALUES ($1, $2, $3)
		`, rmID, t.Jenis, t.Deskripsi)
	}

	// Record SHA-256 Hash Chain Audit Trail
	afterDataMap := map[string]any{
		"id":               rmID,
		"kunjunganId":      kunjunganID,
		"dokterId":         dokterID,
		"pasienId":         pasienID,
		"keluhan":          medCase.Keluhan,
		"hasilPemeriksaan": medCase.HasilPemeriksaan,
		"diagnoses":        medCase.Diagnoses,
		"tindakan":         medCase.TindakanList,
	}
	afterJSON, _ := json.Marshal(afterDataMap)

	err = audit.Record(ctx, tx, q, dokterID, "rekam_medis", rmID, "create", nil, afterJSON)
	if err != nil {
		log.Printf("Warning: failed to record audit log for RM %d: %v", rmID, err)
	}

	if err := tx.Commit(ctx); err != nil {
		log.Printf("Failed to commit rekam medis tx: %v", err)
	}
}
