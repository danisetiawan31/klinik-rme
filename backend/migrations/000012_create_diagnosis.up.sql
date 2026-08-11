CREATE TABLE diagnosis (
    id SERIAL PRIMARY KEY,
    rekam_medis_id INT NOT NULL REFERENCES rekam_medis(id) ON DELETE CASCADE,
    kode_icd VARCHAR(20) NULL,
    deskripsi TEXT NOT NULL
);
