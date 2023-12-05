CREATE TABLE IF NOT EXISTS "user" (
    id              SERIAL         PRIMARY KEY UNIQUE,
    email           TEXT        NOT NULL UNIQUE,
    username        TEXT        NOT NULL UNIQUE,
    password_hash   TEXT        NOT NULL,
    name            TEXT        NOT NULL,
    created_at      TIMESTAMP   NOT NULL
);