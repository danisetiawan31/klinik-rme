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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/danisetiawan31/klinik-rme/internal/api"
	"github.com/danisetiawan31/klinik-rme/internal/api/handler"
	"github.com/danisetiawan31/klinik-rme/internal/db"
	dbgen "github.com/danisetiawan31/klinik-rme/internal/db/generated"
)

func TestRekamMedisFullLifecycle_E2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	postgresContainer, err := postgres.Run(
		ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("test_rm_e2e_db"),
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

	err = db.RunMigrations("../../../migrations", connStr)
	require.NoError(t, err)

	pool, err := db.NewPool(ctx, connStr)
	require.NoError(t, err)
	defer pool.Close()

	q := dbgen.New(pool)

	// Setup Klinik (ID=1)
	var klinikID int32
	err = pool.QueryRow(ctx, `
		INSERT INTO klinik (nama, jam_buka, jam_tutup)
		VALUES ('Klinik Rekam Medis E2E', '08:00', '23:59')
		RETURNING id
	`).Scan(&klinikID)
	require.NoError(t, err)

	router := api.SetupRouter(pool, nil, "http://localhost:3000")

	// Skenario a: Setup sesi petugas & dokter
	var petugasCookie *http.Cookie
	var dokterCookie *http.Cookie
	var dokterUser dbgen.CreateUserRow

	t.Run("Skenario_a_-_Setup_sesi_terautentikasi_petugas_&_dokter", func(t *testing.T) {
		petugasCookie, _ = createKlinikAntrianTestUser(t, ctx, pool, q, "petugas.rme2e@test.com", []string{"petugas"})
		dokterCookie, dokterUser = createKlinikAntrianTestUser(t, ctx, pool, q, "dokter.rme2e@test.com", []string{"dokter"})

		require.NotNil(t, petugasCookie)
		require.NotNil(t, dokterCookie)
		t.Log("Skenario a PASS: Setup sesi terautentikasi petugas & dokter sukses")
	})

	// Skenario b: Petugas: POST /pasien (1 pasien baru)
	var pasienID int32
	t.Run("Skenario_b_-_Petugas_POST_/pasien_1_pasien_baru", func(t *testing.T) {
		consentTrue := true
		pasienReq := handler.CreatePasienRequest{
			Nama:         "Pasien E2E Rekam Medis",
			TanggalLahir: "1992-06-15",
			JenisKelamin: "L",
			Alamat:       "Jl E2E RM No 123",
			NoTelp:       "081234567890",
			Consent:      &consentTrue,
		}
		jsonBody, _ := json.Marshal(pasienReq)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/pasien", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(petugasCookie)

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusCreated, rec.Code)

		var resp handler.PasienResponse
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)

		pasienID = resp.ID
		require.Greater(t, pasienID, int32(0))
		t.Logf("Skenario b PASS: Petugas berhasil membuat pasien baru (ID=%d)", pasienID)
	})

	// Skenario c: Petugas: POST /kunjungan untuk pasien itu
	var kunjunganID int32
	t.Run("Skenario_c_-_Petugas_POST_/kunjungan_untuk_pasien", func(t *testing.T) {
		kunjunganReq := handler.CreateKunjunganRequest{
			PasienID: pasienID,
		}
		jsonBody, _ := json.Marshal(kunjunganReq)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/kunjungan", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(petugasCookie)

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusCreated, rec.Code)

		var resp handler.CreateKunjunganResponse
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)

		kunjunganID = resp.ID
		assert.Equal(t, int32(1), resp.NomorAntrian)
		assert.Equal(t, "menunggu", resp.Status)
		t.Logf("Skenario c PASS: Petugas berhasil mendaftarkan kunjungan (ID=%d, Nomor=%d, Status=%s)", kunjunganID, resp.NomorAntrian, resp.Status)
	})

	// Skenario d: Dokter: POST /klinik/:id/panggil-berikutnya -> status kunjungan 'dipanggil'
	t.Run("Skenario_d_-_Dokter_POST_/klinik/:id/panggil-berikutnya", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/klinik/%d/panggil-berikutnya", klinikID), nil)
		req.AddCookie(dokterCookie)

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)

		var resp handler.PanggilBerikutnyaResponse
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Equal(t, kunjunganID, resp.ID)
		t.Logf("Skenario d PASS: Dokter memanggil antrian berikutnya, kunjungan ID=%d dipanggil", kunjunganID)
	})

	// Skenario e: Dokter: POST /kunjungan/:id/rekam-medis (Level 0) -> status kunjungan 'selesai'
	var rmLevel0ID int32
	t.Run("Skenario_e_-_Dokter_POST_/kunjungan/:id/rekam-medis_Level_0", func(t *testing.T) {
		icd0 := "J00"
		tindakan0 := []handler.CreateTindakanItemRequest{
			{Jenis: "tindakan", Deskripsi: "Pemeriksaan Fisik Paru"},
			{Jenis: "resep", Deskripsi: "Paracetamol 500mg 3x1"},
		}
		body0 := handler.CreateRekamMedisRequest{
			Keluhan:          "Demam dan batuk pilek 3 hari",
			HasilPemeriksaan: "Suhu 38.5C, Faring hiperemis",
			Diagnosis: []handler.CreateDiagnosisItemRequest{
				{KodeIcd: &icd0, Deskripsi: "Acute nasopharyngitis [common cold]"},
			},
			Tindakan: &tindakan0,
		}
		jsonBody, _ := json.Marshal(body0)

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/kunjungan/%d/rekam-medis", kunjunganID), bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(dokterCookie)

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusCreated, rec.Code)

		var resp handler.RekamMedisResponse
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)

		rmLevel0ID = resp.ID
		require.Greater(t, rmLevel0ID, int32(0))

		// Assert via GET /kunjungan/:id bahwa status kunjungan sekarang 'selesai'
		reqGetK, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/kunjungan/%d", kunjunganID), nil)
		reqGetK.AddCookie(dokterCookie)
		recGetK := httptest.NewRecorder()
		router.ServeHTTP(recGetK, reqGetK)

		require.Equal(t, http.StatusOK, recGetK.Code)

		var kResp handler.CreateKunjunganResponse
		_ = json.Unmarshal(recGetK.Body.Bytes(), &kResp)
		assert.Equal(t, "selesai", kResp.Status, "Status kunjungan harus berubah menjadi 'selesai' setelah rekam medis awal disimpan")

		t.Logf("Skenario e PASS: Rekam medis awal Level 0 (ID=%d) berhasil disimpan & status kunjungan ter-update ke 'selesai'", rmLevel0ID)
	})

	// Skenario f: Dokter: POST /rekam-medis/:id/addendum ke Level 0 -> 201 ("Level 1")
	var rmLevel1ID int32
	t.Run("Skenario_f_-_Dokter_POST_/rekam-medis/:id/addendum_ke_Level_0_->_Level_1", func(t *testing.T) {
		newKeluhan1 := "Demam berkurang, timbul nyeri telinga kanan"
		addBody1 := handler.CreateAddendumRequest{
			AlasanAddendum: "Keluhan baru pasien saat follow up sore hari",
			Keluhan:        &newKeluhan1,
		}
		jsonBody, _ := json.Marshal(addBody1)

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/rekam-medis/%d/addendum", rmLevel0ID), bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(dokterCookie)

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusCreated, rec.Code)

		var resp handler.AddendumResponse
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)

		rmLevel1ID = resp.ID
		assert.Equal(t, rmLevel0ID, resp.AddendumOf)
		assert.Equal(t, "Demam berkurang, timbul nyeri telinga kanan", resp.Keluhan)

		t.Logf("Skenario f PASS: Addendum Level 1 (ID=%d) berhasil dibuat merujuk ke Level 0 (ID=%d)", rmLevel1ID, rmLevel0ID)
	})

	// Skenario g: Dokter: POST /rekam-medis/:id/addendum ke Level 1 -> 201 ("Level 2") — Addendum Berantai
	var rmLevel2ID int32
	t.Run("Skenario_g_-_Dokter_POST_/rekam-medis/:id/addendum_ke_Level_1_->_Level_2_Addendum_Berantai", func(t *testing.T) {
		icd0 := "J00"
		icd2 := "H66.0"
		newDiags2 := []handler.CreateDiagnosisItemRequest{
			{KodeIcd: &icd0, Deskripsi: "Acute nasopharyngitis [common cold]"},
			{KodeIcd: &icd2, Deskripsi: "Acute suppurative otitis media"},
		}
		addBody2 := handler.CreateAddendumRequest{
			AlasanAddendum: "Penambahan diagnosis Otitis Media setelah otoskopi",
			Diagnosis:      &newDiags2,
		}
		jsonBody, _ := json.Marshal(addBody2)

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/rekam-medis/%d/addendum", rmLevel1ID), bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(dokterCookie)

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusCreated, rec.Code)

		var resp handler.AddendumResponse
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)

		rmLevel2ID = resp.ID
		assert.Equal(t, rmLevel1ID, resp.AddendumOf, "Addendum Level 2 harus merujuk ke Level 1 (addendum-berantai)")
		require.Len(t, resp.Diagnosis, 2)

		t.Logf("Skenario g PASS: Addendum berantai Level 2 (ID=%d) berhasil dibuat merujuk ke Level 1 (ID=%d)", rmLevel2ID, rmLevel1ID)
	})

	// Skenario h: Dokter: GET /kunjungan/:id/rekam-medis -> assert Level 2 leaf, isAddendum=true
	t.Run("Skenario_h_-_Dokter_GET_/kunjungan/:id/rekam-medis_returns_Level_2_leaf", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/kunjungan/%d/rekam-medis", kunjunganID), nil)
		req.AddCookie(dokterCookie)

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)

		var resp handler.GetRekamMedisResponse
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Equal(t, rmLevel2ID, resp.ID, "GET /kunjungan/:id/rekam-medis harus mengembalikan leaf terkini (Level 2)")
		assert.True(t, resp.IsAddendum, "isAddendum harus true pada leaf addendum")
		assert.Equal(t, "Demam berkurang, timbul nyeri telinga kanan", resp.Keluhan)
		require.Len(t, resp.Diagnosis, 2)

		t.Logf("Skenario h PASS: GET kunjungan rekam medis terverifikasi mengembalikan versi leaf paling akhir (ID=%d)", rmLevel2ID)
	})

	// Skenario i: Dokter: GET /pasien/:id/riwayat -> assert 1 entry, rekamMedis = Level 2
	t.Run("Skenario_i_-_Dokter_GET_/pasien/:id/riwayat_returns_1_entry_Level_2", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/pasien/%d/riwayat", pasienID), nil)
		req.AddCookie(dokterCookie)

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)

		var riwayat []handler.RiwayatKunjunganItem
		err := json.Unmarshal(rec.Body.Bytes(), &riwayat)
		require.NoError(t, err)

		require.Len(t, riwayat, 1, "Riwayat pasien harus berisi tepat 1 kunjungan")
		assert.Equal(t, kunjunganID, riwayat[0].KunjunganID)
		assert.Equal(t, rmLevel2ID, riwayat[0].RekamMedis.ID)
		assert.True(t, riwayat[0].RekamMedis.IsAddendum)

		t.Logf("Skenario i PASS: GET riwayat pasien terverifikasi mengembalikan 1 kunjungan dengan leaf Level 2 (ID=%d)", rmLevel2ID)
	})

	// Skenario j: Verifikasi audit_log LANGSUNG via query DB (3 rows, create -> addendum -> addendum, chain linkage)
	t.Run("Skenario_j_-_Verifikasi_DB_Audit_Log_Chain_Linkage", func(t *testing.T) {
		rows, err := pool.Query(ctx, `
			SELECT id, record_id, aksi, actor_user_id, hash_entry, previous_hash
			FROM audit_log
			WHERE tabel_target = 'rekam_medis' AND record_id IN ($1, $2, $3)
			ORDER BY id ASC
		`, rmLevel0ID, rmLevel1ID, rmLevel2ID)
		require.NoError(t, err)
		defer rows.Close()

		type AuditRow struct {
			ID           int64
			RecordID     int32
			Aksi         string
			ActorUserID  int32
			HashEntry    string
			PreviousHash string
		}

		var auditRows []AuditRow
		for rows.Next() {
			var r AuditRow
			err := rows.Scan(&r.ID, &r.RecordID, &r.Aksi, &r.ActorUserID, &r.HashEntry, &r.PreviousHash)
			require.NoError(t, err)
			auditRows = append(auditRows, r)
		}
		require.NoError(t, rows.Err())

		// Assert PERSIS 3 row
		require.Len(t, auditRows, 3, "Harus ada PERSIS 3 row audit_log untuk 3 level rekam medis")

		// Assert Record ID & Urutan Aksi
		assert.Equal(t, rmLevel0ID, auditRows[0].RecordID)
		assert.Equal(t, "create", auditRows[0].Aksi)
		assert.Equal(t, dokterUser.ID, auditRows[0].ActorUserID)

		assert.Equal(t, rmLevel1ID, auditRows[1].RecordID)
		assert.Equal(t, "addendum", auditRows[1].Aksi)
		assert.Equal(t, dokterUser.ID, auditRows[1].ActorUserID)

		assert.Equal(t, rmLevel2ID, auditRows[2].RecordID)
		assert.Equal(t, "addendum", auditRows[2].Aksi)
		assert.Equal(t, dokterUser.ID, auditRows[2].ActorUserID)

		// Assert CHAIN LINKAGE:
		// Row 2.previous_hash == Row 1.hash_entry
		// Row 3.previous_hash == Row 2.hash_entry
		assert.Equal(t, auditRows[0].HashEntry, auditRows[1].PreviousHash, "Linkage Row 2 -> Row 1 failed: Row 2 previous_hash must match Row 1 hash_entry")
		assert.Equal(t, auditRows[1].HashEntry, auditRows[2].PreviousHash, "Linkage Row 3 -> Row 2 failed: Row 3 previous_hash must match Row 2 hash_entry")

		t.Logf("Skenario j PASS: Audit log chain linkage terverifikasi sempurna (Row 1 hash=%s... -> Row 2 prev=%s..., Row 2 hash=%s... -> Row 3 prev=%s...)",
			auditRows[0].HashEntry[:10], auditRows[1].PreviousHash[:10], auditRows[1].HashEntry[:10], auditRows[2].PreviousHash[:10])
	})

	// Skenario k: Verifikasi kunjungan.status TETAP 'selesai' setelah kedua addendum
	t.Run("Skenario_k_-_Verifikasi_DB_Kunjungan_Status_Tetap_Selesai", func(t *testing.T) {
		var currentStatus string
		err := pool.QueryRow(ctx, `SELECT status FROM kunjungan WHERE id = $1`, kunjunganID).Scan(&currentStatus)
		require.NoError(t, err)

		assert.Equal(t, "selesai", currentStatus, "Status kunjungan harus TETAP 'selesai' dan tidak terpengaruh oleh pembuatan addendum")
		t.Logf("Skenario k PASS: Status kunjungan di DB terverifikasi TETAP '%s'", currentStatus)
	})
}
