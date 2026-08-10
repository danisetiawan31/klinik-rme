package bootstrap_test

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/danisetiawan31/klinik-rme/internal/auth"
	"github.com/danisetiawan31/klinik-rme/internal/bootstrap"
	"github.com/danisetiawan31/klinik-rme/internal/db"
	dbgen "github.com/danisetiawan31/klinik-rme/internal/db/generated"
)

func TestSeedAdmin_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	postgresContainer, err := postgres.Run(
		ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("test_db"),
		postgres.WithUsername("test_user"),
		postgres.WithPassword("test_password"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	require.NoError(t, err)
	defer func() {
		_ = postgresContainer.Terminate(ctx)
	}()

	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	err = db.RunMigrations("../../migrations", connStr)
	require.NoError(t, err)

	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)
	defer pool.Close()

	q := dbgen.New(pool)
	adminEmail := "admin.bootstrap@klinik.local"

	// 1. Skenario 1: Startup Pertama Kali (User Belum Ada)
	t.Run("Skenario 1 - Startup Pertama Kali", func(t *testing.T) {
		err := bootstrap.SeedAdmin(ctx, pool, q, adminEmail)
		require.NoError(t, err)

		// Assert user created with null password_hash
		user, err := q.GetUserByEmail(ctx, adminEmail)
		require.NoError(t, err)
		assert.Equal(t, "Administrator", user.Nama)
		assert.Equal(t, adminEmail, user.Email)
		assert.False(t, user.PasswordHash.Valid)

		// Assert role admin inserted
		roles, err := q.GetRolesByUserID(ctx, user.ID)
		require.NoError(t, err)
		assert.Equal(t, []string{"admin"}, roles)

		// Assert 1 invite token generated
		activeToken, err := q.GetActiveInviteTokenByUserID(ctx, user.ID)
		require.NoError(t, err)
		assert.NotEmpty(t, activeToken.TokenHash)
		assert.Equal(t, "invite", activeToken.Type)
	})

	// 2. Skenario 2: Restart dengan Token Invite Masih Valid
	t.Run("Skenario 2 - Restart dengan Token Invite Masih Valid", func(t *testing.T) {
		user, err := q.GetUserByEmail(ctx, adminEmail)
		require.NoError(t, err)

		tokenBefore, err := q.GetActiveInviteTokenByUserID(ctx, user.ID)
		require.NoError(t, err)

		// Re-run SeedAdmin (simulating server restart)
		err = bootstrap.SeedAdmin(ctx, pool, q, adminEmail)
		require.NoError(t, err)

		tokenAfter, err := q.GetActiveInviteTokenByUserID(ctx, user.ID)
		require.NoError(t, err)

		// Assert token hash is unchanged and no duplicate tokens generated
		assert.Equal(t, tokenBefore.TokenHash, tokenAfter.TokenHash)

		var tokenCount int
		err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM password_tokens WHERE user_id = $1", user.ID).Scan(&tokenCount)
		require.NoError(t, err)
		assert.Equal(t, 1, tokenCount)
	})

	// 3. Skenario 3: Restart dengan Token Invite Sudah Expired
	t.Run("Skenario 3 - Restart dengan Token Invite Expired", func(t *testing.T) {
		user, err := q.GetUserByEmail(ctx, adminEmail)
		require.NoError(t, err)

		// Manually expire the existing token in DB
		_, err = pool.Exec(ctx, "UPDATE password_tokens SET expires_at = now() - INTERVAL '1 hour' WHERE user_id = $1", user.ID)
		require.NoError(t, err)

		// Re-run SeedAdmin (simulating server restart after token expiration)
		err = bootstrap.SeedAdmin(ctx, pool, q, adminEmail)
		require.NoError(t, err)

		// Assert NEW active token generated
		newActiveToken, err := q.GetActiveInviteTokenByUserID(ctx, user.ID)
		require.NoError(t, err)
		assert.NotEmpty(t, newActiveToken.TokenHash)

		// Assert total tokens in DB is now 2 (1 expired + 1 new active)
		var tokenCount int
		err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM password_tokens WHERE user_id = $1", user.ID).Scan(&tokenCount)
		require.NoError(t, err)
		assert.Equal(t, 2, tokenCount)
	})

	// 4. Skenario 4: Admin Sudah Selesai Setup (Password Hash Sudah Terisi)
	t.Run("Skenario 4 - Admin Sudah Selesai Setup Password", func(t *testing.T) {
		user, err := q.GetUserByEmail(ctx, adminEmail)
		require.NoError(t, err)

		// Manually complete password setup by setting password_hash
		passHash, _ := auth.Hash("AdminSecurePassword!123")
		_, err = pool.Exec(ctx, "UPDATE users SET password_hash = $1 WHERE id = $2", passHash, user.ID)
		require.NoError(t, err)

		// Get token count before restart
		var countBefore int
		_ = pool.QueryRow(ctx, "SELECT COUNT(*) FROM password_tokens WHERE user_id = $1", user.ID).Scan(&countBefore)

		// Re-run SeedAdmin (simulating server restart after admin completed password setup)
		err = bootstrap.SeedAdmin(ctx, pool, q, adminEmail)
		require.NoError(t, err)

		// Assert SeedAdmin skipped total: token count unchanged
		var countAfter int
		_ = pool.QueryRow(ctx, "SELECT COUNT(*) FROM password_tokens WHERE user_id = $1", user.ID).Scan(&countAfter)
		assert.Equal(t, countBefore, countAfter)
	})
}
