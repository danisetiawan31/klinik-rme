CREATE TABLE klinik (
    id SERIAL PRIMARY KEY,
    nama VARCHAR(255) NOT NULL,
    jam_buka TIME NOT NULL,
    jam_tutup TIME NOT NULL
);
