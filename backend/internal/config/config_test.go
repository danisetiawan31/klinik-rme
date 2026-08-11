package config_test

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/danisetiawan31/klinik-rme/internal/config"
)

func setValidEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DB_HOST", "localhost")
	t.Setenv("DB_PORT", "5432")
	t.Setenv("DB_USER", "postgres")
	t.Setenv("DB_PASSWORD", "secret")
	t.Setenv("DB_NAME", "klinik_db")
	t.Setenv("TZ", "Asia/Jakarta")
	t.Setenv("RESEND_API_KEY", "re_123456789")
	t.Setenv("RESEND_FROM_EMAIL", "onboarding@resend.dev")
	t.Setenv("FRONTEND_BASE_URL", "http://localhost:4200")
	t.Setenv("SEED_ADMIN_EMAIL", "admin@klinik.local")
	t.Setenv("KLINIK_NAMA", "Klinik Sehat Utama")
	t.Setenv("KLINIK_JAM_BUKA", "08:00")
	t.Setenv("KLINIK_JAM_TUTUP", "17:00")
}

func TestLoad_Success(t *testing.T) {
	setValidEnv(t)

	cfg, err := config.Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "localhost", cfg.DBHost)
	assert.Equal(t, 5432, cfg.DBPort)
	assert.Equal(t, "postgres", cfg.DBUser)
	assert.Equal(t, "secret", cfg.DBPassword)
	assert.Equal(t, "klinik_db", cfg.DBName)
	assert.Equal(t, "Asia/Jakarta", cfg.TZ)
	assert.Equal(t, "re_123456789", cfg.ResendAPIKey)
	assert.Equal(t, "onboarding@resend.dev", cfg.ResendFromEmail)
	assert.Equal(t, "http://localhost:4200", cfg.FrontendBaseURL)
	assert.Equal(t, "admin@klinik.local", cfg.SeedAdminEmail)
	assert.Equal(t, "Klinik Sehat Utama", cfg.KlinikNama)
	assert.Equal(t, "08:00", cfg.KlinikJamBuka)
	assert.Equal(t, "17:00", cfg.KlinikJamTutup)

	assert.Equal(t, "postgres://postgres:secret@localhost:5432/klinik_db?sslmode=disable", cfg.DSN())
	assert.Equal(t, "Asia/Jakarta", time.Local.String())
}

func TestLoad_MissingRequiredEnv(t *testing.T) {
	tests := []struct {
		name       string
		unsetEnv   string
		errMessage string
	}{
		{
			name:       "missing DB_HOST",
			unsetEnv:   "DB_HOST",
			errMessage: "missing required environment variable: DB_HOST",
		},
		{
			name:       "missing DB_PORT",
			unsetEnv:   "DB_PORT",
			errMessage: "missing required environment variable: DB_PORT",
		},
		{
			name:       "missing DB_USER",
			unsetEnv:   "DB_USER",
			errMessage: "missing required environment variable: DB_USER",
		},
		{
			name:       "missing DB_PASSWORD",
			unsetEnv:   "DB_PASSWORD",
			errMessage: "missing required environment variable: DB_PASSWORD",
		},
		{
			name:       "missing DB_NAME",
			unsetEnv:   "DB_NAME",
			errMessage: "missing required environment variable: DB_NAME",
		},
		{
			name:       "missing TZ",
			unsetEnv:   "TZ",
			errMessage: "missing required environment variable: TZ",
		},
		{
			name:       "missing RESEND_API_KEY",
			unsetEnv:   "RESEND_API_KEY",
			errMessage: "missing required environment variable: RESEND_API_KEY",
		},
		{
			name:       "missing RESEND_FROM_EMAIL",
			unsetEnv:   "RESEND_FROM_EMAIL",
			errMessage: "missing required environment variable: RESEND_FROM_EMAIL",
		},
		{
			name:       "missing FRONTEND_BASE_URL",
			unsetEnv:   "FRONTEND_BASE_URL",
			errMessage: "missing required environment variable: FRONTEND_BASE_URL",
		},
		{
			name:       "missing SEED_ADMIN_EMAIL",
			unsetEnv:   "SEED_ADMIN_EMAIL",
			errMessage: "missing required environment variable: SEED_ADMIN_EMAIL",
		},
		{
			name:       "missing KLINIK_NAMA",
			unsetEnv:   "KLINIK_NAMA",
			errMessage: "missing required environment variable: KLINIK_NAMA",
		},
		{
			name:       "missing KLINIK_JAM_BUKA",
			unsetEnv:   "KLINIK_JAM_BUKA",
			errMessage: "missing required environment variable: KLINIK_JAM_BUKA",
		},
		{
			name:       "missing KLINIK_JAM_TUTUP",
			unsetEnv:   "KLINIK_JAM_TUTUP",
			errMessage: "missing required environment variable: KLINIK_JAM_TUTUP",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setValidEnv(t)
			os.Unsetenv(tt.unsetEnv)

			cfg, err := config.Load()
			require.Error(t, err)
			assert.Nil(t, cfg)
			assert.Contains(t, err.Error(), tt.errMessage)
		})
	}
}

func TestLoad_InvalidPort(t *testing.T) {
	setValidEnv(t)
	t.Setenv("DB_PORT", "invalid")

	cfg, err := config.Load()
	require.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "invalid environment variable DB_PORT")
}

func TestLoad_InvalidTZ(t *testing.T) {
	setValidEnv(t)
	t.Setenv("TZ", "Invalid/Timezone_Name")

	cfg, err := config.Load()
	require.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "invalid environment variable TZ")
}

func TestLoad_InvalidKlinikJamFormat(t *testing.T) {
	setValidEnv(t)
	t.Setenv("KLINIK_JAM_BUKA", "8:00") // missing leading zero or invalid format

	cfg, err := config.Load()
	require.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "invalid environment variable KLINIK_JAM_BUKA")

	setValidEnv(t)
	t.Setenv("KLINIK_JAM_TUTUP", "25:00") // invalid hour

	cfg, err = config.Load()
	require.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "invalid environment variable KLINIK_JAM_TUTUP")
}
