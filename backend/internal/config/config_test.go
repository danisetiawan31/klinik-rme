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
