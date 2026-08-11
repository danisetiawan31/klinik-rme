-- name: UpsertQueueCounter :one
INSERT INTO queue_counter (klinik_id, tanggal, last_number)
VALUES ($1, $2, 1)
ON CONFLICT (klinik_id, tanggal)
DO UPDATE SET last_number = queue_counter.last_number + 1
RETURNING last_number;
