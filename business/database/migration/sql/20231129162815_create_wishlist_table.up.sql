CREATE TABLE IF NOT EXISTS "wishlist" (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    owner_id INT NOT NULL,
    CONSTRAINT owner
        FOREIGN KEY(owner_id)
            REFERENCES "user"(id)
            ON DELETE CASCADE
);