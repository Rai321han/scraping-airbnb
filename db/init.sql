CREATE TABLE IF NOT EXISTS properties (
    id SERIAL PRIMARY KEY,
    platform TEXT NOT NULL,
    title TEXT,
    price REAL,
    location TEXT,
    url TEXT UNIQUE,
    rating REAL,
    description TEXT
);

CREATE INDEX IF NOT EXISTS idx_properties_location ON properties (location);
