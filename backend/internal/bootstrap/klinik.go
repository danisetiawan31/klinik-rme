package bootstrap

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/danisetiawan31/klinik-rme/internal/config"
	dbgen "github.com/danisetiawan31/klinik-rme/internal/db/generated"
)

// SeedKlinik ensures that an initial single klinik row exists in DB.
// It is idempotent and must be called during server startup.
func SeedKlinik(ctx context.Context, pool *pgxpool.Pool, q *dbgen.Queries, cfg *config.Config) error {
	count, err := q.CountKlinik(ctx)
	if err != nil {
		return fmt.Errorf("failed to check existing klinik count: %w", err)
	}

	if count > 0 {
		log.Printf("[KLINIK_BOOTSTRAP] Klinik row already exists (count=%d). Skipping bootstrap.", count)
		return nil
	}

	jamBukaTime, err := parseHHMMToPgTime(cfg.KlinikJamBuka)
	if err != nil {
		return fmt.Errorf("invalid KLINIK_JAM_BUKA (%s): %w", cfg.KlinikJamBuka, err)
	}

	jamTutupTime, err := parseHHMMToPgTime(cfg.KlinikJamTutup)
	if err != nil {
		return fmt.Errorf("invalid KLINIK_JAM_TUTUP (%s): %w", cfg.KlinikJamTutup, err)
	}

	createdKlinik, err := q.InsertKlinik(ctx, dbgen.InsertKlinikParams{
		Nama:     cfg.KlinikNama,
		JamBuka:  jamBukaTime,
		JamTutup: jamTutupTime,
	})
	if err != nil {
		return fmt.Errorf("failed to insert initial klinik: %w", err)
	}

	log.Printf("[KLINIK_BOOTSTRAP] Created initial klinik (ID=%d, Nama=%s, Buka=%s, Tutup=%s)",
		createdKlinik.ID, createdKlinik.Nama, cfg.KlinikJamBuka, cfg.KlinikJamTutup)

	return nil
}

func parseHHMMToPgTime(s string) (pgtype.Time, error) {
	t, err := time.Parse("15:04", s)
	if err != nil {
		return pgtype.Time{}, err
	}
	microseconds := int64(t.Hour()*3600+t.Minute()*60) * 1_000_000
	return pgtype.Time{Microseconds: microseconds, Valid: true}, nil
}
