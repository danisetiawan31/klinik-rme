CREATE TABLE kunjungan (
    id SERIAL PRIMARY KEY,
    pasien_id INT NOT NULL REFERENCES pasien(id),
    klinik_id INT NOT NULL REFERENCES klinik(id),
    dokter_id INT REFERENCES users(id),
    tanggal_kunjungan DATE NOT NULL,
    nomor_antrian INT NOT NULL,
    is_priority BOOLEAN NOT NULL DEFAULT false,
    priority_reason TEXT,
    skip_count INT NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'menunggu' CHECK (status IN ('menunggu', 'dipanggil', 'selesai', 'tidak_hadir')),
    dipanggil_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_kunjungan_klinik_tanggal_status ON kunjungan(klinik_id, tanggal_kunjungan, status);
