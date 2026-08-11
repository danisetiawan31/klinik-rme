ALTER TABLE audit_log DROP CONSTRAINT IF EXISTS audit_log_aksi_check;
ALTER TABLE audit_log ADD CONSTRAINT audit_log_aksi_check CHECK (aksi IN ('create', 'update'));
