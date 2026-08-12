package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/danisetiawan31/klinik-rme/internal/api"
	"github.com/danisetiawan31/klinik-rme/internal/db"
	dbgen "github.com/danisetiawan31/klinik-rme/internal/db/generated"
)

func TestPasienFullLifecycle_E2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E integration test in short mode")
	}

	ctx := context.Background()

	// Isolated PostgreSQL container exclusively for E2E audit chain verification
	postgresContainer, err := postgres.Run(
		ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("test_pasien_e2e_db"),
		postgres.WithUsername("test_user"),
		postgres.WithPassword("test_password"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	require.NoError(t, err, "failed to start isolated postgres container")
	defer func() {
		_ = postgresContainer.Terminate(ctx)
	}()

	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	err = db.RunMigrations("../../../migrations", connStr)
	require.NoError(t, err, "failed to apply migrations")

	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)
	defer pool.Close()

	q := dbgen.New(pool)
	router := api.SetupRouter(pool, nil, nil, "http://localhost:3000")

	// Step a: Setup authenticated sessions for role petugas and role dokter using helper
	petugasCookie, petugasUser := createTestUserWithSession(t, ctx, pool, q, "petugas.e2e@klinik.com", []string{"petugas"})
	dokterCookie, dokterUser := createTestUserWithSession(t, ctx, pool, q, "dokter.e2e@klinik.com", []string{"dokter"})

	require.NotNil(t, petugasCookie)
	require.NotNil(t, dokterCookie)
	t.Logf("Step a PASS: Sessions initialized for Petugas (ID=%d) & Dokter (ID=%d)", petugasUser.ID, dokterUser.ID)

	// Step b: Sebagai petugas: POST /pasien (data pasien baru lengkap, consent=true) -> 201, simpan id & response body
	initialNik := "3201011508920005"
	initialNama := "Budi E2E Pratama"
	initialDOB := "1992-08-15"
	initialJK := "L"
	initialAlamat := "Jl. E2E Raya No. 100"
	initialNoTelp := "081299887766"

	createBody := map[string]interface{}{
		"nik":          initialNik,
		"nama":         initialNama,
		"tanggalLahir": initialDOB,
		"jenisKelamin": initialJK,
		"alamat":       initialAlamat,
		"noTelp":       initialNoTelp,
		"consent":      true,
	}
	createJSON, _ := json.Marshal(createBody)

	req := httptest.NewRequest("POST", "/api/v1/pasien", bytes.NewBuffer(createJSON))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(petugasCookie)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code, "POST /pasien must return 201 Created")

	var createRes map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &createRes)
	require.NoError(t, err)

	pasienID := int32(createRes["id"].(float64))
	assert.Equal(t, initialNik, createRes["nik"])
	assert.Equal(t, initialNama, createRes["nama"])
	assert.Equal(t, float64(1), createRes["version"])
	assert.NotEmpty(t, createRes["consentAt"])
	t.Logf("Step b PASS: Created patient ID=%d, version=1, consentAt=%s", pasienID, createRes["consentAt"])

	// Step c: Sebagai petugas: GET /pasien/search?nik=<nik> -> assert pasien itu muncul di hasil
	req = httptest.NewRequest("GET", fmt.Sprintf("/api/v1/pasien/search?nik=%s", initialNik), nil)
	req.AddCookie(petugasCookie)
	rec = httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var searchNikRes []map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &searchNikRes)
	require.NoError(t, err)
	require.Len(t, searchNikRes, 1, "exact NIK search must return 1 result")
	assert.Equal(t, float64(pasienID), searchNikRes[0]["id"])
	assert.Equal(t, initialNama, searchNikRes[0]["nama"])
	t.Logf("Step c PASS: Petugas searched patient by exact NIK %s successfully", initialNik)

	// Step d: Sebagai dokter: GET /pasien/search?nama=<sebagian_nama> -> assert pasien itu muncul
	req = httptest.NewRequest("GET", "/api/v1/pasien/search?nama=Pratama", nil)
	req.AddCookie(dokterCookie)
	rec = httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, "Dokter role must be allowed to search patients (200 OK)")

	var searchNamaRes []map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &searchNamaRes)
	require.NoError(t, err)
	require.NotEmpty(t, searchNamaRes)
	assert.Equal(t, float64(pasienID), searchNamaRes[0]["id"])
	t.Logf("Step d PASS: Dokter searched patient by partial name successfully")

	// Step e: Sebagai dokter: GET /pasien/:id -> assert data lengkap sesuai yang di-input, riwayatKunjunganRingkas=[]
	req = httptest.NewRequest("GET", fmt.Sprintf("/api/v1/pasien/%d", pasienID), nil)
	req.AddCookie(dokterCookie)
	rec = httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var detailRes map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &detailRes)
	require.NoError(t, err)

	assert.Equal(t, float64(pasienID), detailRes["id"])
	assert.Equal(t, initialNik, detailRes["nik"])
	assert.Equal(t, initialNama, detailRes["nama"])
	assert.Equal(t, initialDOB, detailRes["tanggalLahir"])
	assert.Equal(t, initialJK, detailRes["jenisKelamin"])
	assert.Equal(t, initialAlamat, detailRes["alamat"])
	assert.Equal(t, initialNoTelp, detailRes["noTelp"])
	assert.NotNil(t, detailRes["riwayatKunjunganRingkas"])
	assert.Len(t, detailRes["riwayatKunjunganRingkas"], 0, "riwayatKunjunganRingkas must be empty array []")
	t.Logf("Step e PASS: Dokter retrieved full detail for patient ID=%d with empty kunjungan array", pasienID)

	// Step f: Sebagai petugas: PATCH /pasien/:id dengan version=1, ubah minimal 1 field (alamat) -> 200, version bertambah jadi 2
	updatedAlamat := "Jl. E2E Terusan Baru No. 200 Subur"
	patchBody := map[string]interface{}{
		"version": 1,
		"alamat":  updatedAlamat,
	}
	patchJSON, _ := json.Marshal(patchBody)

	req = httptest.NewRequest("PATCH", fmt.Sprintf("/api/v1/pasien/%d", pasienID), bytes.NewBuffer(patchJSON))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(petugasCookie)
	rec = httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, "PATCH /pasien/:id with matching version must return 200 OK")

	var patchRes map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &patchRes)
	require.NoError(t, err)

	assert.Equal(t, float64(2), patchRes["version"], "version must increment from 1 to 2")
	assert.Equal(t, updatedAlamat, patchRes["alamat"])
	assert.Equal(t, initialNama, patchRes["nama"], "unsent field nama must remain unchanged")
	t.Logf("Step f PASS: Petugas updated patient address, version incremented from 1 to 2")

	// Step g: Sebagai dokter: GET /pasien/:id lagi -> assert perubahan dari langkah f benar-benar tersimpan (alamat terbaru)
	req = httptest.NewRequest("GET", fmt.Sprintf("/api/v1/pasien/%d", pasienID), nil)
	req.AddCookie(dokterCookie)
	rec = httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var detailUpdatedRes map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &detailUpdatedRes)
	require.NoError(t, err)

	assert.Equal(t, float64(2), detailUpdatedRes["version"])
	assert.Equal(t, updatedAlamat, detailUpdatedRes["alamat"], "updated address must be visible to dokter")
	t.Logf("Step g PASS: Dokter verified updated address (%s) on GET /pasien/%d", updatedAlamat, pasienID)

	// Step h: Verifikasi audit trail LANGSUNG lewat query DB
	type auditLogRecord struct {
		ID           int32
		ActorUserID  int32
		TabelTarget  string
		RecordID     int32
		Aksi         string
		BeforeData   []byte
		AfterData    []byte
		HashEntry    string
		PreviousHash string
	}

	rows, err := pool.Query(ctx, `
		SELECT id, actor_user_id, tabel_target, record_id, aksi, before_data, after_data, hash_entry, previous_hash
		FROM audit_log
		WHERE tabel_target = 'pasien' AND record_id = $1
		ORDER BY id ASC
	`, pasienID)
	require.NoError(t, err)
	defer rows.Close()

	var auditEntries []auditLogRecord
	for rows.Next() {
		var rec auditLogRecord
		err := rows.Scan(
			&rec.ID,
			&rec.ActorUserID,
			&rec.TabelTarget,
			&rec.RecordID,
			&rec.Aksi,
			&rec.BeforeData,
			&rec.AfterData,
			&rec.HashEntry,
			&rec.PreviousHash,
		)
		require.NoError(t, err)
		auditEntries = append(auditEntries, rec)
	}
	require.NoError(t, rows.Err())

	// Assertion 1: EXACTLY 2 rows (create & update)
	require.Len(t, auditEntries, 2, "audit_log must contain EXACTLY 2 entries for this patient")

	// Assertion 2: Row 1 (create)
	row1 := auditEntries[0]
	assert.Equal(t, "create", row1.Aksi)
	assert.Equal(t, "pasien", row1.TabelTarget)
	assert.Equal(t, pasienID, row1.RecordID)
	assert.Equal(t, petugasUser.ID, row1.ActorUserID)
	assert.Nil(t, row1.BeforeData, "before_data MUST be NULL for create action")
	require.NotNil(t, row1.AfterData, "after_data MUST NOT be NULL for create action")

	var row1After map[string]interface{}
	err = json.Unmarshal(row1.AfterData, &row1After)
	require.NoError(t, err)
	assert.Equal(t, initialNik, row1After["nik"])
	assert.Equal(t, initialNama, row1After["nama"])
	assert.Equal(t, initialAlamat, row1After["alamat"])
	assert.NotEmpty(t, row1After["consentAt"])

	// Assertion 3: Row 2 (update)
	row2 := auditEntries[1]
	assert.Equal(t, "update", row2.Aksi)
	assert.Equal(t, "pasien", row2.TabelTarget)
	assert.Equal(t, pasienID, row2.RecordID)
	assert.Equal(t, petugasUser.ID, row2.ActorUserID)
	require.NotNil(t, row2.BeforeData, "before_data MUST NOT be NULL for update action")
	require.NotNil(t, row2.AfterData, "after_data MUST NOT be NULL for update action")

	var row2Before map[string]interface{}
	err = json.Unmarshal(row2.BeforeData, &row2Before)
	require.NoError(t, err)
	assert.Equal(t, initialAlamat, row2Before["alamat"], "before_data must contain address BEFORE update")
	assert.Equal(t, initialNama, row2Before["nama"])

	var row2After map[string]interface{}
	err = json.Unmarshal(row2.AfterData, &row2After)
	require.NoError(t, err)
	assert.Equal(t, updatedAlamat, row2After["alamat"], "after_data must contain address AFTER update")
	assert.Equal(t, initialNama, row2After["nama"])

	// Assertion 4: Strict Hash-Chain Linkage (row2.previous_hash == row1.hash_entry)
	assert.Equal(t, row1.HashEntry, row2.PreviousHash, "row 2 previous_hash MUST EQUAL EXACTLY row 1 hash_entry to prove sequential chain linkage")

	t.Logf("Step h PASS: Audit log verified directly in DB. Row 1 hash: %s..., Row 2 previous_hash: %s...", row1.HashEntry[:16], row2.PreviousHash[:16])
}
