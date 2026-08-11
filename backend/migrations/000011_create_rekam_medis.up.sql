CREATE TABLE rekam_medis (
    id SERIAL PRIMARY KEY,
    kunjungan_id INT NOT NULL REFERENCES kunjungan(id),
    dokter_id INT NOT NULL REFERENCES users(id),
    keluhan TEXT NOT NULL,
    hasil_pemeriksaan TEXT NOT NULL,
    is_addendum BOOLEAN NOT NULL DEFAULT FALSE,
    addendum_of INT NULL REFERENCES rekam_medis(id),
    alasan_addendum TEXT NULL,
    deleted_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX uq_addendum_of_active ON rekam_medis(addendum_of) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX uq_rekam_medis_root_per_kunjungan ON rekam_medis(kunjungan_id) WHERE addendum_of IS NULL AND deleted_at IS NULL;
