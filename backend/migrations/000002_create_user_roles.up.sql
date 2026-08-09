CREATE TABLE user_roles (
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role TEXT NOT NULL CHECK (role IN ('petugas', 'dokter', 'admin')),
    PRIMARY KEY (user_id, role)
);
