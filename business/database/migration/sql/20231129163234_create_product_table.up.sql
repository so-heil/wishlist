CREATE TABLE IF NOT EXISTS "product" (
       id SERIAL PRIMARY KEY UNIQUE,
       name TEXT NOT NULL,
       description TEXT,
       created_at TIMESTAMP NOT NULL,
       updated_at TIMESTAMP NOT NULL,
       image_url TEXT,
       price TEXT,
       price_updated_at TIMESTAMP NOT NULL,
       wishlist_id INT NOT NULL,
       CONSTRAINT wishlist
           FOREIGN KEY(wishlist_id)
               REFERENCES "wishlist"(id)
               ON DELETE CASCADE
);