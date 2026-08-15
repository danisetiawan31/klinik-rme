package main

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/danisetiawan31/klinik-rme/internal/auth"
	"github.com/danisetiawan31/klinik-rme/internal/config"
	"github.com/danisetiawan31/klinik-rme/internal/db"
)

func main() {
	_ = godotenv.Load()

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

	log.Println("Seeding demo data...")

	pwHash, err := auth.Hash("Password123!")
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}

	// 1. Seed Accounts (Admin, Dokter, Petugas)
	seedUsers(ctx, pool, pwHash)

	// 2. Seed Sample Patients
	pasienIDs := seedPatients(ctx, pool)

	// 3. Seed Sample Antrian Kunjungan Hari Ini
	seedQueue(ctx, pool, pasienIDs)

	log.Println("==================================================")
	log.Println("🎉 DEMO DATA BERHASIL DI-SEED LENGKAP!")
	log.Println("==================================================")
	log.Println("Akun Siap Pakai (Semua password: Password123!):")
	log.Println("1. Admin   : admin@klinik.local   | Password: Password123!")
	log.Println("2. Dokter  : dokter@klinik.local  | Password: Password123!")
	log.Println("3. Petugas : petugas@klinik.local | Password: Password123!")
	log.Println("==================================================")
}

func seedUsers(ctx context.Context, pool *pgxpool.Pool, pwHash string) {
	users := []struct {
		nama  string
		email string
		role  string
	}{
		{"Administrator", "admin@klinik.local", "admin"},
		{"dr. Budi Santoso", "dokter@klinik.local", "dokter"},
		{"Siti Rahmawati", "petugas@klinik.local", "petugas"},
	}

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

		log.Printf("✓ User: %s (%s) [Role: %s, Password: Password123!]", u.nama, u.email, u.role)
	}
}

func seedPatients(ctx context.Context, pool *pgxpool.Pool) []int32 {
	patients := []struct {
		nik          string
		nama         string
		tanggalLahir string
		jenisKelamin string
		alamat       string
		noTelp       string
	}{
		{"3171010101900001", "Ahmad Pratama", "1990-01-01", "L", "Jl. Sudirman No. 12, Jakarta", "081234567890"},
		{"3271020202950002", "Dewi Lestari", "1995-02-02", "P", "Jl. Thamrin No. 45, Jakarta", "081298765432"},
		{"3371030303880003", "Bambang Hidayat", "1988-03-03", "L", "Jl. Gatot Subroto No. 88, Jakarta", "085612345678"},
		{"3471040404000004", "Siti Aisyah", "2000-04-04", "P", "Jl. Rasuna Said No. 10, Jakarta", "087812345678"},
	}

	var IDs []int32
	for _, p := range patients {
		var existingID int32
		err := pool.QueryRow(ctx, `SELECT id FROM pasien WHERE nik = $1 AND deleted_at IS NULL`, p.nik).Scan(&existingID)
		if err == nil {
			IDs = append(IDs, existingID)
			log.Printf("✓ Pasien existing: %s (ID: %d, NIK: %s)", p.nama, existingID, p.nik)
			continue
		} else if !errors.Is(err, pgx.ErrNoRows) {
			log.Printf("Error checking pasien: %v", err)
			continue
		}

		tgl, _ := time.Parse("2006-01-02", p.tanggalLahir)
		var pasienID int32
		err = pool.QueryRow(ctx, `
			INSERT INTO pasien (nik, nama, tanggal_lahir, jenis_kelamin, alamat, no_telp, consent_at, version)
			VALUES ($1, $2, $3, $4, $5, $6, NOW(), 1)
			RETURNING id
		`, p.nik, p.nama, pgtype.Date{Time: tgl, Valid: true}, p.jenisKelamin, p.alamat, p.noTelp).Scan(&pasienID)
		if err != nil {
			log.Printf("Failed to insert pasien %s: %v", p.nama, err)
			continue
		}
		IDs = append(IDs, pasienID)
		log.Printf("✓ Pasien created: %s (ID: %d, NIK: %s)", p.nama, pasienID, p.nik)
	}
	return IDs
}

func seedQueue(ctx context.Context, pool *pgxpool.Pool, pasienIDs []int32) {
	if len(pasienIDs) < 2 {
		return
	}

	today := time.Now()
	var klinikID int32
	err := pool.QueryRow(ctx, `SELECT id FROM klinik LIMIT 1`).Scan(&klinikID)
	if err != nil {
		log.Printf("Klinik not found, skipping queue seed: %v", err)
		return
	}

	// Check if queue already seeded for today
	var existingQueueCount int64
	_ = pool.QueryRow(ctx, `SELECT COUNT(*) FROM kunjungan WHERE klinik_id = $1 AND tanggal_kunjungan = $2`, klinikID, pgtype.Date{Time: today, Valid: true}).Scan(&existingQueueCount)
	if existingQueueCount > 0 {
		log.Printf("✓ Antrian hari ini sudah ada (%d kunjungan)", existingQueueCount)
		return
	}

	// Queue 1: Ahmad Pratama (Menunggu)
	_, _ = pool.Exec(ctx, `
		INSERT INTO kunjungan (pasien_id, klinik_id, tanggal_kunjungan, nomor_antrian, status, is_priority, created_at)
		VALUES ($1, $2, $3, 1, 'menunggu', false, NOW())
	`, pasienIDs[0], klinikID, pgtype.Date{Time: today, Valid: true})

	// Queue 2: Dewi Lestari (Prioritas - Menunggu)
	reason := "Lansia dengan keluhan pusing berat"
	_, _ = pool.Exec(ctx, `
		INSERT INTO kunjungan (pasien_id, klinik_id, tanggal_kunjungan, nomor_antrian, status, is_priority, priority_reason, created_at)
		VALUES ($1, $2, $3, 2, 'menunggu', true, $4, NOW())
	`, pasienIDs[1], klinikID, pgtype.Date{Time: today, Valid: true}, reason)

	// Update queue_counter
	_, _ = pool.Exec(ctx, `
		INSERT INTO queue_counter (klinik_id, tanggal, last_counter)
		VALUES ($1, $2, 2)
		ON CONFLICT (klinik_id, tanggal) DO UPDATE
		SET last_counter = GREATEST(queue_counter.last_counter, EXCLUDED.last_counter)
	`, klinikID, pgtype.Date{Time: today, Valid: true})

	log.Println("✓ Antrian hari ini: Nomor 1 (Ahmad Pratama) & Nomor 2 Prioritas (Dewi Lestari)")
}
