CREATE TABLE tindakan (
    id SERIAL PRIMARY KEY,
    rekam_medis_id INT NOT NULL REFERENCES rekam_medis(id) ON DELETE CASCADE,
    jenis VARCHAR(20) NOT NULL CHECK (jenis IN ('tindakan', 'resep')),
    deskripsi TEXT NOT NULL
);
