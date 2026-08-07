ALTER TABLE inventory_files
    ADD COLUMN IF NOT EXISTS category TEXT NOT NULL DEFAULT 'pending-review',
    ADD COLUMN IF NOT EXISTS kind TEXT NOT NULL DEFAULT 'other',
    ADD COLUMN IF NOT EXISTS parsed_title TEXT,
    ADD COLUMN IF NOT EXISTS parsed_year INTEGER,
    ADD COLUMN IF NOT EXISTS season INTEGER,
    ADD COLUMN IF NOT EXISTS episode INTEGER,
    ADD COLUMN IF NOT EXISTS edition TEXT,
    ADD COLUMN IF NOT EXISTS source TEXT,
    ADD COLUMN IF NOT EXISTS audio TEXT,
    ADD COLUMN IF NOT EXISTS identity_status TEXT NOT NULL DEFAULT 'unidentified',
    ADD COLUMN IF NOT EXISTS external_id TEXT;

CREATE INDEX IF NOT EXISTS inventory_identity_idx ON inventory_files(kind, parsed_title, parsed_year);
