package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/danisetiawan31/klinik-rme/internal/auth"
	dbgen "github.com/danisetiawan31/klinik-rme/internal/db/generated"
)

// SeedAdmin ensures that an initial admin account exists and has a valid invite token if password setup is incomplete.
// It is idempotent and must be called during server startup.
func SeedAdmin(ctx context.Context, pool *pgxpool.Pool, q *dbgen.Queries, adminEmail string) error {
	user, err := q.GetUserByEmail(ctx, adminEmail)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// 1. Startup first time -> Create admin user and role in an explicit DB transaction
			tx, err := pool.Begin(ctx)
			if err != nil {
				return fmt.Errorf("failed to begin transaction for admin bootstrap: %w", err)
			}
			defer func() {
				_ = tx.Rollback(ctx)
			}()

			qtx := q.WithTx(tx)
			createdUser, err := qtx.CreateUser(ctx, dbgen.CreateUserParams{
				Nama:  "Administrator",
				Email: adminEmail,
			})
			if err != nil {
				return fmt.Errorf("failed to create admin user row: %w", err)
			}

			err = qtx.InsertUserRole(ctx, dbgen.InsertUserRoleParams{
				UserID: createdUser.ID,
				Role:   "admin",
			})
			if err != nil {
				return fmt.Errorf("failed to assign admin role: %w", err)
			}

			if err := tx.Commit(ctx); err != nil {
				return fmt.Errorf("failed to commit admin user creation: %w", err)
			}

			user = dbgen.User{
				ID:           createdUser.ID,
				Nama:         createdUser.Nama,
				Email:        createdUser.Email,
				PasswordHash: pgtype.Text{Valid: false},
			}
			log.Printf("[ADMIN_BOOTSTRAP] Created initial admin account for %s", adminEmail)
		} else {
			return fmt.Errorf("failed to check existing admin user: %w", err)
		}
	}

	// 2. If admin has already set password -> skip total
	if user.PasswordHash.Valid && user.PasswordHash.String != "" {
		log.Printf("[ADMIN_BOOTSTRAP] Admin account %s has already completed setup. Skipping bootstrap.", adminEmail)
		return nil
	}

	// 3. Password hash is NULL -> check for active invite token
	activeToken, err := q.GetActiveInviteTokenByUserID(ctx, user.ID)
	if err == nil {
		// Active invite token exists -> skip generation, log informative message (no reprint of raw token)
		log.Printf("[ADMIN_BOOTSTRAP] Active invite token for admin %s already exists (expires at %s). Skipping token generation.",
			adminEmail, activeToken.ExpiresAt.Time.Format(time.RFC3339))
		return nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("failed to query active invite token: %w", err)
	}

	// 4. No active invite token -> generate new invite token and print plaintext to server log ONCE
	rawToken, err := auth.GenerateToken()
	if err != nil {
		return fmt.Errorf("failed to generate admin invite token: %w", err)
	}

	tokenHash := auth.HashToken(rawToken)
	expiresAt := time.Now().Add(7 * 24 * time.Hour) // Invite token TTL = 7 days

	err = q.InsertPasswordToken(ctx, dbgen.InsertPasswordTokenParams{
		TokenHash: tokenHash,
		UserID:    user.ID,
		Type:      "invite",
		ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to insert admin invite token: %w", err)
	}

	log.Printf("[ADMIN_BOOTSTRAP] Generated new admin invite token for %s: %s (expires at %s). Use this token to complete initial admin setup.",
		adminEmail, rawToken, expiresAt.Format(time.RFC3339))

	return nil
}
