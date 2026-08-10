CREATE TABLE audit_log (
    id SERIAL PRIMARY KEY,
    tabel_target VARCHAR(100) NOT NULL,
    record_id INT NOT NULL,
    actor_user_id INT NOT NULL REFERENCES users(id),
    aksi VARCHAR(10) NOT NULL CHECK (aksi IN ('create', 'update')),
    before_data JSONB NULL,
    after_data JSONB NOT NULL,
    hash_entry VARCHAR(64) NOT NULL,
    previous_hash VARCHAR(64) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE OR REPLACE FUNCTION prevent_audit_log_tamper()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'audit_log is append-only';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER audit_log_prevent_update
BEFORE UPDATE ON audit_log
FOR EACH ROW
EXECUTE FUNCTION prevent_audit_log_tamper();

CREATE TRIGGER audit_log_prevent_delete
BEFORE DELETE ON audit_log
FOR EACH ROW
EXECUTE FUNCTION prevent_audit_log_tamper();
