package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	DBHost          string
	DBPort          int
	DBUser          string
	DBPassword      string
	DBName          string
	TZ              string
	HTTPPort        string
	ResendAPIKey    string
	ResendFromEmail string
	FrontendBaseURL string
	SeedAdminEmail  string
}

// Load reads environment variables and strictly validates mandatory variables.
// Returns an error if any mandatory variable is missing or invalid.
func Load() (*Config, error) {
	host := os.Getenv("DB_HOST")
	if host == "" {
		return nil, fmt.Errorf("missing required environment variable: DB_HOST")
	}

	portStr := os.Getenv("DB_PORT")
	if portStr == "" {
		return nil, fmt.Errorf("missing required environment variable: DB_PORT")
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 {
		return nil, fmt.Errorf("invalid environment variable DB_PORT: %s", portStr)
	}

	user := os.Getenv("DB_USER")
	if user == "" {
		return nil, fmt.Errorf("missing required environment variable: DB_USER")
	}

	password := os.Getenv("DB_PASSWORD")
	if password == "" {
		return nil, fmt.Errorf("missing required environment variable: DB_PASSWORD")
	}

	dbname := os.Getenv("DB_NAME")
	if dbname == "" {
		return nil, fmt.Errorf("missing required environment variable: DB_NAME")
	}

	tz := os.Getenv("TZ")
	if tz == "" {
		return nil, fmt.Errorf("missing required environment variable: TZ")
	}

	loc, err := time.LoadLocation(tz)
	if err != nil {
		return nil, fmt.Errorf("invalid environment variable TZ (%s): %w", tz, err)
	}
	time.Local = loc

	resendAPIKey := os.Getenv("RESEND_API_KEY")
	if resendAPIKey == "" {
		return nil, fmt.Errorf("missing required environment variable: RESEND_API_KEY")
	}

	resendFromEmail := os.Getenv("RESEND_FROM_EMAIL")
	if resendFromEmail == "" {
		return nil, fmt.Errorf("missing required environment variable: RESEND_FROM_EMAIL")
	}

	frontendBaseURL := os.Getenv("FRONTEND_BASE_URL")
	if frontendBaseURL == "" {
		return nil, fmt.Errorf("missing required environment variable: FRONTEND_BASE_URL")
	}

	seedAdminEmail := os.Getenv("SEED_ADMIN_EMAIL")
	if seedAdminEmail == "" {
		return nil, fmt.Errorf("missing required environment variable: SEED_ADMIN_EMAIL")
	}

	httpPort := os.Getenv("PORT")
	if httpPort == "" {
		httpPort = os.Getenv("HTTP_PORT")
	}
	if httpPort == "" {
		httpPort = "8080"
	}

	cfg := &Config{
		DBHost:          host,
		DBPort:          port,
		DBUser:          user,
		DBPassword:      password,
		DBName:          dbname,
		TZ:              tz,
		HTTPPort:        httpPort,
		ResendAPIKey:    resendAPIKey,
		ResendFromEmail: resendFromEmail,
		FrontendBaseURL: frontendBaseURL,
		SeedAdminEmail:  seedAdminEmail,
	}

	return cfg, nil
}

// DSN returns the PostgreSQL connection DSN string.
func (c *Config) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName)
}
