CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    nama TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT
);
