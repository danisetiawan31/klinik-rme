package audit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	dbgen "github.com/danisetiawan31/klinik-rme/internal/db/generated"
)

// auditHashInput defines the immutable schema for calculating audit entry hashes.
type auditHashInput struct {
	PreviousHash string          `json:"previousHash"`
	TabelTarget  string          `json:"tabelTarget"`
	RecordID     int32           `json:"recordId"`
	ActorUserID  int32           `json:"actorUserId"`
	Aksi         string          `json:"aksi"`
	BeforeData   json.RawMessage `json:"beforeData"`
	AfterData    json.RawMessage `json:"afterData"`
	CreatedAt    string          `json:"createdAt"`
}

// Record writes a tamper-evident hash-chain audit log entry within the caller's active transaction.
// It locks the audit_log_tail row (FOR UPDATE), computes the hash_entry using SHA256 of the JSON-marshaled
// auditHashInput struct, inserts into audit_log, and updates audit_log_tail.last_hash.
//
// The caller is responsible for opening (pool.Begin) and committing/rolling back the transaction.
func Record(
	ctx context.Context,
	tx pgx.Tx,
	q *dbgen.Queries,
	actorUserID int32,
	tabelTarget string,
	recordID int32,
	aksi string,
	beforeData json.RawMessage,
	afterData json.RawMessage,
) error {
	if tx == nil {
		return fmt.Errorf("audit.Record requires an active non-nil transaction")
	}

	qtx := q.WithTx(tx)

	// Step b: Lock audit_log_tail (id=1) FOR UPDATE
	lastHash, err := qtx.LockAuditLogTail(ctx)
	if err != nil {
		return fmt.Errorf("failed to lock audit_log_tail: %w", err)
	}

	// Step c: Timestamp generated in Go before hash calculation
	createdAt := time.Now()

	// Ensure beforeData evaluates to JSON literal null if empty/nil
	var hashBeforeData json.RawMessage
	if len(beforeData) == 0 {
		hashBeforeData = json.RawMessage("null")
	} else {
		hashBeforeData = beforeData
	}

	// Step d: Compute hash_entry using formula
	hashInput := auditHashInput{
		PreviousHash: lastHash,
		TabelTarget:  tabelTarget,
		RecordID:     recordID,
		ActorUserID:  actorUserID,
		Aksi:         aksi,
		BeforeData:   hashBeforeData,
		AfterData:    afterData,
		CreatedAt:    createdAt.Format(time.RFC3339),
	}

	jsonBytes, err := json.Marshal(hashInput)
	if err != nil {
		return fmt.Errorf("failed to marshal audit hash input: %w", err)
	}

	hashSum := sha256.Sum256(jsonBytes)
	hashEntry := hex.EncodeToString(hashSum[:])

	// Step e: Insert row into audit_log (passing SQL NULL if beforeData is empty/nil)
	var dbBeforeData []byte
	if len(beforeData) > 0 {
		dbBeforeData = beforeData
	}

	err = qtx.InsertAuditLog(ctx, dbgen.InsertAuditLogParams{
		TabelTarget:  tabelTarget,
		RecordID:     recordID,
		ActorUserID:  actorUserID,
		Aksi:         aksi,
		BeforeData:   dbBeforeData,
		AfterData:    afterData,
		HashEntry:    hashEntry,
		PreviousHash: lastHash,
		CreatedAt:    pgtype.Timestamptz{Time: createdAt, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to insert audit_log: %w", err)
	}

	// Step f: Update audit_log_tail with new hash_entry
	err = qtx.UpdateAuditLogTailHash(ctx, hashEntry)
	if err != nil {
		return fmt.Errorf("failed to update audit_log_tail: %w", err)
	}

	// Step g: Return nil (caller manages transaction lifecycle)
	return nil
}
