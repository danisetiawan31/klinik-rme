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
	"github.com/danisetiawan31/klinik-rme/internal/bootstrap"
	"github.com/danisetiawan31/klinik-rme/internal/config"
	"github.com/danisetiawan31/klinik-rme/internal/db"
	dbgen "github.com/danisetiawan31/klinik-rme/internal/db/generated"
)

func TestKlinikAntrianFullLifecycle_E2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	postgresContainer, err := postgres.Run(
		ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("test_klinik_antrian_e2e_db"),
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

	// Seed klinik
	cfg := &config.Config{
		KlinikNama:     "Klinik Sehat Utama E2E",
		KlinikJamBuka:  "08:00",
		KlinikJamTutup: "23:59",
	}
	err = bootstrap.SeedKlinik(ctx, pool, q, cfg)
	require.NoError(t, err)

	klinik, err := q.GetSingleKlinik(ctx)
	require.NoError(t, err)

	router := api.SetupRouter(pool, nil, "http://localhost:3000")

	// Skenario a: Setup sesi terautentikasi petugas, dokter, & admin
	petugasCookie, _ := createKlinikAntrianTestUser(t, ctx, pool, q, "petugas.e2e@test.com", []string{"petugas"})
	dokterCookie, dokterUser := createKlinikAntrianTestUser(t, ctx, pool, q, "dokter.e2e@test.com", []string{"dokter"})
	adminCookie, _ := createKlinikAntrianTestUser(t, ctx, pool, q, "admin.e2e@test.com", []string{"admin"})

	t.Log("Skenario a PASS: Setup sesi terautentikasi petugas, dokter, admin sukses")

	// Skenario b: Sebagai petugas, buat 2 pasien baru (Pasien 1 Normal, Pasien 2 Prioritas Candidate)
	reqP1 := handler.CreatePasienRequest{
		Nama:         "Budi Santoso",
		TanggalLahir: "1990-01-01",
		JenisKelamin: "L",
		Alamat:       "Jl E2E Normal",
		NoTelp:       "0811111111",
		Consent:      boolPtr(true),
	}
	jsonP1, _ := json.Marshal(reqP1)
	httpReqP1, _ := http.NewRequest(http.MethodPost, "/api/v1/pasien", bytes.NewBuffer(jsonP1))
	httpReqP1.Header.Set("Content-Type", "application/json")
	httpReqP1.AddCookie(petugasCookie)
	recP1 := httptest.NewRecorder()
	router.ServeHTTP(recP1, httpReqP1)
	require.Equal(t, http.StatusCreated, recP1.Code)
	var respP1 handler.PasienResponse
	err = json.Unmarshal(recP1.Body.Bytes(), &respP1)
	require.NoError(t, err)
	pasien1ID := respP1.ID

	reqP2 := handler.CreatePasienRequest{
		Nama:         "Siti Rahma",
		TanggalLahir: "1992-02-02",
		JenisKelamin: "P",
		Alamat:       "Jl E2E Priority",
		NoTelp:       "0822222222",
		Consent:      boolPtr(true),
	}
	jsonP2, _ := json.Marshal(reqP2)
	httpReqP2, _ := http.NewRequest(http.MethodPost, "/api/v1/pasien", bytes.NewBuffer(jsonP2))
	httpReqP2.Header.Set("Content-Type", "application/json")
	httpReqP2.AddCookie(petugasCookie)
	recP2 := httptest.NewRecorder()
	router.ServeHTTP(recP2, httpReqP2)
	require.Equal(t, http.StatusCreated, recP2.Code)
	var respP2 handler.PasienResponse
	err = json.Unmarshal(recP2.Body.Bytes(), &respP2)
	require.NoError(t, err)
	pasien2ID := respP2.ID

	t.Logf("Skenario b PASS: Berhasil membuat 2 pasien baru: Pasien 1 (ID=%d), Pasien 2 (ID=%d)", pasien1ID, pasien2ID)

	// Skenario c: Sebagai petugas, daftarkan kunjungan untuk kedua pasien (1 Normal -> nomor 1, 1 Prioritas -> nomor 2)
	bodyK1 := handler.CreateKunjunganRequest{
		PasienID: pasien1ID,
	}
	jsonK1, _ := json.Marshal(bodyK1)
	httpReqK1, _ := http.NewRequest(http.MethodPost, "/api/v1/kunjungan", bytes.NewBuffer(jsonK1))
	httpReqK1.Header.Set("Content-Type", "application/json")
	httpReqK1.AddCookie(petugasCookie)
	recK1 := httptest.NewRecorder()
	router.ServeHTTP(recK1, httpReqK1)
	require.Equal(t, http.StatusCreated, recK1.Code)
	var respK1 handler.CreateKunjunganResponse
	err = json.Unmarshal(recK1.Body.Bytes(), &respK1)
	require.NoError(t, err)
	assert.Equal(t, int32(1), respK1.NomorAntrian)
	kunjungan1ID := respK1.ID

	isPriority := true
	reason := "Lansia Gawat"
	bodyK2 := handler.CreateKunjunganRequest{
		PasienID:       pasien2ID,
		IsPriority:     &isPriority,
		PriorityReason: &reason,
	}
	jsonK2, _ := json.Marshal(bodyK2)
	httpReqK2, _ := http.NewRequest(http.MethodPost, "/api/v1/kunjungan", bytes.NewBuffer(jsonK2))
	httpReqK2.Header.Set("Content-Type", "application/json")
	httpReqK2.AddCookie(petugasCookie)
	recK2 := httptest.NewRecorder()
	router.ServeHTTP(recK2, httpReqK2)
	require.Equal(t, http.StatusCreated, recK2.Code)
	var respK2 handler.CreateKunjunganResponse
	err = json.Unmarshal(recK2.Body.Bytes(), &respK2)
	require.NoError(t, err)
	assert.Equal(t, int32(2), respK2.NomorAntrian)
	kunjungan2ID := respK2.ID

	t.Logf("Skenario c PASS: Dibuat Kunjungan 1 (ID=%d, nomor=1) & Kunjungan 2 (ID=%d, nomor=2, priority=true)", kunjungan1ID, kunjungan2ID)

	// Skenario d: Sebagai dokter, GET /klinik/:id/antrian -> assert 2 kunjungan muncul berstatus 'menunggu'
	httpReqList, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/klinik/%d/antrian", klinik.ID), nil)
	httpReqList.AddCookie(dokterCookie)
	recList := httptest.NewRecorder()
	router.ServeHTTP(recList, httpReqList)
	require.Equal(t, http.StatusOK, recList.Code)
	var antrianList []handler.AntrianItemResponse
	err = json.Unmarshal(recList.Body.Bytes(), &antrianList)
	require.NoError(t, err)
	require.Len(t, antrianList, 2)
	assert.Equal(t, "menunggu", antrianList[0].Status)
	assert.Equal(t, "menunggu", antrianList[1].Status)

	t.Log("Skenario d PASS: GET /klinik/:id/antrian menampilkan 2 kunjungan status 'menunggu'")

	// Skenario e: Sebagai dokter, POST /panggil-berikutnya -> assert Kunjungan 2 (PRIORITAS, nomor 2) kepanggil duluan meski didaftarkan belakangan!
	httpReqCall1, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/klinik/%d/panggil-berikutnya", klinik.ID), nil)
	httpReqCall1.AddCookie(dokterCookie)
	recCall1 := httptest.NewRecorder()
	router.ServeHTTP(recCall1, httpReqCall1)
	require.Equal(t, http.StatusOK, recCall1.Code)
	var respCall1 handler.PanggilBerikutnyaResponse
	err = json.Unmarshal(recCall1.Body.Bytes(), &respCall1)
	require.NoError(t, err)
	assert.Equal(t, kunjungan2ID, respCall1.ID, "Kunjungan prioritas (ID 2) harus dipanggil duluan")
	assert.Equal(t, "Siti Rahma", respCall1.PasienNama)
	assert.Equal(t, dokterUser.ID, respCall1.DokterID)

	t.Logf("Skenario e PASS: POST /panggil-berikutnya berhasil memanggil Kunjungan 2 (ID=%d, Prioritas, Nama=%s) lebih dulu dari nomor 1", respCall1.ID, respCall1.PasienNama)

	// Skenario f: Sebagai dokter YANG SAMA, POST /kunjungan/:id/lewati untuk Kunjungan 2 -> assert status balik 'menunggu', skipCount=1
	httpReqSkip, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/kunjungan/%d/lewati", kunjungan2ID), nil)
	httpReqSkip.AddCookie(dokterCookie)
	recSkip := httptest.NewRecorder()
	router.ServeHTTP(recSkip, httpReqSkip)
	require.Equal(t, http.StatusOK, recSkip.Code)
	var respSkip handler.UpdateSkipResponse
	err = json.Unmarshal(recSkip.Body.Bytes(), &respSkip)
	require.NoError(t, err)
	assert.Equal(t, kunjungan2ID, respSkip.ID)
	assert.Equal(t, "menunggu", respSkip.Status)
	assert.Equal(t, int32(1), respSkip.SkipCount)

	t.Logf("Skenario f PASS: POST /kunjungan/%d/lewati berhasil (status='menunggu', skipCount=1)", kunjungan2ID)

	// Skenario g: Sebagai dokter, POST /panggil-berikutnya LAGI.
	// Evaluasi ORDER BY k.is_priority DESC, k.skip_count ASC, k.nomor_antrian ASC:
	// Kunjungan 2 (prioritas=true, skip=1, nomor=2) vs Kunjungan 1 (prioritas=false, skip=0, nomor=1).
	// Karena is_priority DESC adalah kunci utama sorting, is_priority=true (Kunjungan 2) tetap memenangkan klaim dibanding is_priority=false (Kunjungan 1)!
	httpReqCall2, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/klinik/%d/panggil-berikutnya", klinik.ID), nil)
	httpReqCall2.AddCookie(dokterCookie)
	recCall2 := httptest.NewRecorder()
	router.ServeHTTP(recCall2, httpReqCall2)
	require.Equal(t, http.StatusOK, recCall2.Code)
	var respCall2 handler.PanggilBerikutnyaResponse
	err = json.Unmarshal(recCall2.Body.Bytes(), &respCall2)
	require.NoError(t, err)
	assert.Equal(t, kunjungan2ID, respCall2.ID, "Kunjungan prioritas (ID 2, is_priority=true) tetap menang klaim karena is_priority DESC paling utama")

	t.Logf("Skenario g PASS: POST /panggil-berikutnya kedua memanggil Kunjungan %d (Prioritas=true) sesuai urutan ORDER BY is_priority DESC", respCall2.ID)

	// Skenario h: Sebagai admin, POST /kunjungan/:id/tidak-hadir untuk Kunjungan 2 yang dipanggil lagi -> assert status 'tidak_hadir'
	httpReqTH, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/kunjungan/%d/tidak-hadir", kunjungan2ID), nil)
	httpReqTH.AddCookie(adminCookie)
	recTH := httptest.NewRecorder()
	router.ServeHTTP(recTH, httpReqTH)
	require.Equal(t, http.StatusOK, recTH.Code)
	var respTH handler.UpdateTidakHadirResponse
	err = json.Unmarshal(recTH.Body.Bytes(), &respTH)
	require.NoError(t, err)
	assert.Equal(t, "tidak_hadir", respTH.Status)

	t.Logf("Skenario h PASS: POST /kunjungan/%d/tidak-hadir oleh Admin sukses (status='tidak_hadir')", kunjungan2ID)

	// Skenario i: Sebagai dokter, GET /klinik/:id/antrian -> assert kondisi akhir antrian: Kunjungan 1 'menunggu', Kunjungan 2 'tidak_hadir'
	httpReqFinal, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/klinik/%d/antrian", klinik.ID), nil)
	httpReqFinal.AddCookie(dokterCookie)
	recFinal := httptest.NewRecorder()
	router.ServeHTTP(recFinal, httpReqFinal)
	require.Equal(t, http.StatusOK, recFinal.Code)
	var antrianFinal []handler.AntrianItemResponse
	err = json.Unmarshal(recFinal.Body.Bytes(), &antrianFinal)
	require.NoError(t, err)
	require.Len(t, antrianFinal, 2)

	assert.Equal(t, kunjungan1ID, antrianFinal[0].ID)
	assert.Equal(t, "menunggu", antrianFinal[0].Status)

	assert.Equal(t, kunjungan2ID, antrianFinal[1].ID)
	assert.Equal(t, "tidak_hadir", antrianFinal[1].Status)

	t.Log("Skenario i PASS: GET /klinik/:id/antrian akhir terverifikasi (Kunjungan 1 'menunggu', Kunjungan 2 'tidak_hadir')")

	// Skenario j: Verifikasi DB langsung: TIDAK ADA row audit_log untuk tabel_target IN ('kunjungan', 'klinik', 'queue_counter')
	var auditCount int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM audit_log WHERE tabel_target IN ('kunjungan', 'klinik', 'queue_counter')`).Scan(&auditCount)
	require.NoError(t, err)
	assert.Equal(t, 0, auditCount, "Queue operations MUST NOT produce any audit log entries")

	t.Log("Skenario j PASS: Query DB memverifikasi 0 row audit_log tercatat untuk tabel antrian (kunjungan, klinik, queue_counter)")
}

func boolPtr(b bool) *bool {
	return &b
}
