package audit_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/danisetiawan31/klinik-rme/internal/audit"
	"github.com/danisetiawan31/klinik-rme/internal/db"
	dbgen "github.com/danisetiawan31/klinik-rme/internal/db/generated"
)

type auditHashInputTest struct {
	PreviousHash string          `json:"previousHash"`
	TabelTarget  string          `json:"tabelTarget"`
	RecordID     int32           `json:"recordId"`
	ActorUserID  int32           `json:"actorUserId"`
	Aksi         string          `json:"aksi"`
	BeforeData   json.RawMessage `json:"beforeData"`
	AfterData    json.RawMessage `json:"afterData"`
	CreatedAt    string          `json:"createdAt"`
}

func TestAuditTrailHelper_RealPostgreSQL(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	postgresContainer, err := postgres.Run(
		ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("test_audit_helper_db"),
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

	// Create test user for actor_user_id FK requirement
	var actorUserID int32
	err = pool.QueryRow(ctx, "INSERT INTO users (nama, email, password_hash) VALUES ('Audit Actor', 'actor.audit@test.com', 'hash') RETURNING id").Scan(&actorUserID)
	require.NoError(t, err)

	genesisHash := "f5ebe6fb00b0cf82d9b6c624cd93d9ceb6f6647b48ab7c0bad7915f62caffb8f"

	// -------------------------------------------------------------------------
	// Test 1: Single Call & Manual Hash Verification
	// -------------------------------------------------------------------------
	t.Run("1. Single Call & Manual Hash Verification", func(t *testing.T) {
		tx, err := pool.Begin(ctx)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback(ctx) }()

		tabelTarget := "pasien"
		recordID := int32(100)
		aksi := "create"
		afterData := json.RawMessage(`{"nama":"Budi Setyo"}`)

		err = audit.Record(ctx, tx, q, actorUserID, tabelTarget, recordID, aksi, nil, afterData)
		require.NoError(t, err)

		err = tx.Commit(ctx)
		require.NoError(t, err)

		// Verify audit_log row in DB
		var dbID int32
		var dbTabel string
		var dbRecID, dbActorID int32
		var dbAksi, dbHashEntry, dbPrevHash string
		var dbCreatedAt time.Time
		var dbBeforeData []byte
		var dbAfterData []byte

		err = pool.QueryRow(ctx, `SELECT id, tabel_target, record_id, actor_user_id, aksi, before_data, after_data, hash_entry, previous_hash, created_at 
			FROM audit_log WHERE id = (SELECT MAX(id) FROM audit_log)`).Scan(
			&dbID, &dbTabel, &dbRecID, &dbActorID, &dbAksi, &dbBeforeData, &dbAfterData, &dbHashEntry, &dbPrevHash, &dbCreatedAt,
		)
		require.NoError(t, err)

		assert.Equal(t, tabelTarget, dbTabel)
		assert.Equal(t, recordID, dbRecID)
		assert.Equal(t, actorUserID, dbActorID)
		assert.Equal(t, aksi, dbAksi)
		assert.Nil(t, dbBeforeData, "before_data in DB must be SQL NULL for aksi=create")
		assert.JSONEq(t, string(afterData), string(dbAfterData))
		assert.Equal(t, genesisHash, dbPrevHash, "first entry previous_hash must match genesis hash")

		// Recalculate hash manually in test using same formula
		manualInput := auditHashInputTest{
			PreviousHash: genesisHash,
			TabelTarget:  tabelTarget,
			RecordID:     recordID,
			ActorUserID:  actorUserID,
			Aksi:         aksi,
			BeforeData:   json.RawMessage("null"),
			AfterData:    afterData,
			CreatedAt:    dbCreatedAt.Format(time.RFC3339),
		}
		manualBytes, err := json.Marshal(manualInput)
		require.NoError(t, err)
		manualSum := sha256.Sum256(manualBytes)
		expectedHash := hex.EncodeToString(manualSum[:])

		assert.Equal(t, expectedHash, dbHashEntry, "manual SHA256 hash must match db hash_entry")

		// Verify audit_log_tail last_hash is updated to dbHashEntry
		var tailHash string
		err = pool.QueryRow(ctx, "SELECT last_hash FROM audit_log_tail WHERE id = 1").Scan(&tailHash)
		require.NoError(t, err)
		assert.Equal(t, dbHashEntry, tailHash)
	})

	// -------------------------------------------------------------------------
	// Test 2: Sequential Calls & Chain Linkage
	// -------------------------------------------------------------------------
	t.Run("2. Sequential Calls & Chain Linkage", func(t *testing.T) {
		// Clean table for clean sequential verification
		_, _ = pool.Exec(ctx, "TRUNCATE audit_log RESTART IDENTITY")
		_, _ = pool.Exec(ctx, "UPDATE audit_log_tail SET last_hash = $1 WHERE id = 1", genesisHash)

		// 3 sequential calls
		for i := 1; i <= 3; i++ {
			tx, err := pool.Begin(ctx)
			require.NoError(t, err)

			var before json.RawMessage
			if i > 1 {
				before = json.RawMessage(fmt.Sprintf(`{"val":%d}`, i-1))
			}
			after := json.RawMessage(fmt.Sprintf(`{"val":%d}`, i))

			err = audit.Record(ctx, tx, q, actorUserID, "pasien", 200, "update", before, after)
			require.NoError(t, err)

			err = tx.Commit(ctx)
			require.NoError(t, err)
		}

		// Read all 3 rows from audit_log
		rows, err := pool.Query(ctx, "SELECT id, hash_entry, previous_hash FROM audit_log ORDER BY id ASC")
		require.NoError(t, err)
		defer rows.Close()

		type entry struct {
			id       int
			hash     string
			prevHash string
		}
		var entries []entry
		for rows.Next() {
			var e entry
			err := rows.Scan(&e.id, &e.hash, &e.prevHash)
			require.NoError(t, err)
			entries = append(entries, e)
		}
		require.Len(t, entries, 3)

		// Assert chain linkage:
		// Entry 1 previous_hash == genesisHash
		assert.Equal(t, genesisHash, entries[0].prevHash)
		// Entry 2 previous_hash == Entry 1 hash_entry
		assert.Equal(t, entries[0].hash, entries[1].prevHash)
		// Entry 3 previous_hash == Entry 2 hash_entry
		assert.Equal(t, entries[1].hash, entries[2].prevHash)
	})

	// -------------------------------------------------------------------------
	// Test 3: Concurrency Test (10 Goroutines Parallel)
	// -------------------------------------------------------------------------
	t.Run("3. Concurrency Test (10 Goroutines Parallel)", func(t *testing.T) {
		// Reset database state to genesis
		_, _ = pool.Exec(ctx, "TRUNCATE audit_log RESTART IDENTITY")
		_, _ = pool.Exec(ctx, "UPDATE audit_log_tail SET last_hash = $1 WHERE id = 1", genesisHash)

		const numGoroutines = 10
		var wg sync.WaitGroup
		errChan := make(chan error, numGoroutines)

		for i := 1; i <= numGoroutines; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()

				tx, err := pool.Begin(ctx)
				if err != nil {
					errChan <- fmt.Errorf("worker %d failed to begin tx: %w", workerID, err)
					return
				}

				after := json.RawMessage(fmt.Sprintf(`{"worker":%d}`, workerID))
				err = audit.Record(ctx, tx, q, actorUserID, "kunjungan", int32(workerID), "create", nil, after)
				if err != nil {
					_ = tx.Rollback(ctx)
					errChan <- fmt.Errorf("worker %d audit.Record failed: %w", workerID, err)
					return
				}

				err = tx.Commit(ctx)
				if err != nil {
					errChan <- fmt.Errorf("worker %d tx.Commit failed: %w", workerID, err)
					return
				}
			}(i)
		}

		wg.Wait()
		close(errChan)

		for err := range errChan {
			require.NoError(t, err, "no worker should fail during concurrent audit log insertion")
		}

		// (a) Assert exactly 10 rows inserted
		var totalCount int
		err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM audit_log").Scan(&totalCount)
		require.NoError(t, err)
		assert.Equal(t, numGoroutines, totalCount, "exactly 10 audit log rows must be inserted")

		// (b) Assert NO TWO ROWS HAVE THE SAME previous_hash (No duplicate/branching previous_hash)
		var duplicateCount int
		err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM (
			SELECT previous_hash FROM audit_log GROUP BY previous_hash HAVING COUNT(*) > 1
		) duplicates`).Scan(&duplicateCount)
		require.NoError(t, err)
		assert.Equal(t, 0, duplicateCount, "there must be NO duplicate previous_hash entries (zero branching)")

		// (c) Assert single unbroken linear chain from genesis to final entry
		rows, err := pool.Query(ctx, "SELECT hash_entry, previous_hash FROM audit_log")
		require.NoError(t, err)
		defer rows.Close()

		prevToNextMap := make(map[string]string)
		for rows.Next() {
			var hash, prev string
			err := rows.Scan(&hash, &prev)
			require.NoError(t, err)
			prevToNextMap[prev] = hash
		}

		currHash := genesisHash
		chainLength := 0
		for {
			nextHash, exists := prevToNextMap[currHash]
			if !exists {
				break
			}
			chainLength++
			currHash = nextHash
		}

		assert.Equal(t, numGoroutines, chainLength, "linear chain from genesis must traverse all 10 entries sequentially without gap")

		// Verify audit_log_tail matches the end of the chain (currHash)
		var finalTailHash string
		err = pool.QueryRow(ctx, "SELECT last_hash FROM audit_log_tail WHERE id = 1").Scan(&finalTailHash)
		require.NoError(t, err)
		assert.Equal(t, currHash, finalTailHash)
	})

	// -------------------------------------------------------------------------
	// Test 4: Robustness of Hash Formula Test (No Delimiter Boundary Collisions)
	// -------------------------------------------------------------------------
	t.Run("4. Robustness of Hash Formula Test", func(t *testing.T) {
		tx1, err := pool.Begin(ctx)
		require.NoError(t, err)
		defer func() { _ = tx1.Rollback(ctx) }()

		// Pair 1: before="A|B", after="C"
		before1 := json.RawMessage(`"A|B"`)
		after1 := json.RawMessage(`"C"`)
		err = audit.Record(ctx, tx1, q, actorUserID, "pasien", 301, "update", before1, after1)
		require.NoError(t, err)

		var hash1 string
		err = tx1.QueryRow(ctx, "SELECT hash_entry FROM audit_log WHERE record_id = 301").Scan(&hash1)
		require.NoError(t, err)

		tx2, err := pool.Begin(ctx)
		require.NoError(t, err)
		defer func() { _ = tx2.Rollback(ctx) }()

		// Pair 2: before="A", after="B|C"
		before2 := json.RawMessage(`"A"`)
		after2 := json.RawMessage(`"B|C"`)
		err = audit.Record(ctx, tx2, q, actorUserID, "pasien", 302, "update", before2, after2)
		require.NoError(t, err)

		var hash2 string
		err = tx2.QueryRow(ctx, "SELECT hash_entry FROM audit_log WHERE record_id = 302").Scan(&hash2)
		require.NoError(t, err)

		assert.NotEqual(t, hash1, hash2, "hashes for Pair 1 (A|B, C) and Pair 2 (A, B|C) MUST NOT collide")
	})

	// -------------------------------------------------------------------------
	// Test 5: Nil beforeData (aksi='create')
	// -------------------------------------------------------------------------
	t.Run("5. Nil beforeData (aksi=create)", func(t *testing.T) {
		tx, err := pool.Begin(ctx)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback(ctx) }()

		afterData := json.RawMessage(`{"code":"NEW_RECORD"}`)
		err = audit.Record(ctx, tx, q, actorUserID, "rekam_medis", 500, "create", nil, afterData)
		require.NoError(t, err)

		err = tx.Commit(ctx)
		require.NoError(t, err)

		// Assert before_data column in DB is SQL NULL
		var isNull bool
		err = pool.QueryRow(ctx, "SELECT (before_data IS NULL) FROM audit_log WHERE record_id = 500").Scan(&isNull)
		require.NoError(t, err)
		assert.True(t, isNull, "before_data column in DB must be actual SQL NULL when beforeData is nil")
	})

	// -------------------------------------------------------------------------
	// Test 6: Tamper Prevention Triggers on Real Application Row (audit.Record)
	// -------------------------------------------------------------------------
	t.Run("6. Tamper Prevention Triggers on Real Application Row", func(t *testing.T) {
		tx, err := pool.Begin(ctx)
		require.NoError(t, err)

		afterData := json.RawMessage(`{"status":"CREATED_VIA_REAL_PATH"}`)
		err = audit.Record(ctx, tx, q, actorUserID, "pasien", 999, "create", nil, afterData)
		require.NoError(t, err)

		err = tx.Commit(ctx)
		require.NoError(t, err)

		var realRowID int32
		err = pool.QueryRow(ctx, "SELECT id FROM audit_log WHERE record_id = 999 AND tabel_target = 'pasien'").Scan(&realRowID)
		require.NoError(t, err)

		// Attempt UPDATE on real row created via audit.Record -> MUST fail with trigger exception
		_, err = pool.Exec(ctx, "UPDATE audit_log SET tabel_target = 'hacked' WHERE id = $1", realRowID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "audit_log is append-only")

		// Attempt DELETE on real row created via audit.Record -> MUST fail with trigger exception
		_, err = pool.Exec(ctx, "DELETE FROM audit_log WHERE id = $1", realRowID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "audit_log is append-only")
	})
}
