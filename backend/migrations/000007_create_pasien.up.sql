CREATE TABLE pasien (
    id SERIAL PRIMARY KEY,
    nik VARCHAR(50) NULL,
    nama VARCHAR(255) NOT NULL,
    tanggal_lahir DATE NOT NULL,
    jenis_kelamin VARCHAR(1) NOT NULL CHECK (jenis_kelamin IN ('L', 'P')),
    alamat TEXT NOT NULL,
    no_telp VARCHAR(50) NOT NULL,
    consent_at TIMESTAMPTZ NOT NULL,
    version INT NOT NULL DEFAULT 1,
    deleted_at TIMESTAMPTZ NULL
);
