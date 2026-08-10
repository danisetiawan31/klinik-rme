CREATE TABLE audit_log_tail (
    id INT PRIMARY KEY CHECK (id = 1),
    last_hash VARCHAR(64) NOT NULL
);

INSERT INTO audit_log_tail (id, last_hash)
VALUES (1, 'f5ebe6fb00b0cf82d9b6c624cd93d9ceb6f6647b48ab7c0bad7915f62caffb8f');
