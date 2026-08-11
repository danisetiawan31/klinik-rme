CREATE TABLE queue_counter (
    klinik_id INT NOT NULL REFERENCES klinik(id) ON DELETE CASCADE,
    tanggal DATE NOT NULL,
    last_number INT NOT NULL DEFAULT 0,
    PRIMARY KEY (klinik_id, tanggal)
);
