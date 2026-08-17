-- SHA-256 identifica el contenido, pero no debe impedir registrar dos rutas duplicadas.
ALTER TABLE inventory_files DROP CONSTRAINT IF EXISTS inventory_files_sha256_key;
CREATE INDEX IF NOT EXISTS inventory_files_sha256_idx ON inventory_files(sha256);

CREATE TABLE identity_candidates (
    id BIGSERIAL PRIMARY KEY,
    inventory_file_id BIGINT NOT NULL REFERENCES inventory_files(id) ON DELETE CASCADE,
    provider TEXT NOT NULL CHECK (provider IN ('tmdb','tvdb','imdb','filebot','manual')),
    external_id TEXT,
    title TEXT NOT NULL,
    year INTEGER,
    score NUMERIC(5,4),
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','accepted','rejected')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(inventory_file_id, provider, external_id)
);

CREATE TABLE media_editions (
    id BIGSERIAL PRIMARY KEY,
    content_id BIGINT NOT NULL REFERENCES contents(id) ON DELETE CASCADE,
    label TEXT NOT NULL DEFAULT 'standard',
    UNIQUE(content_id, label)
);

CREATE TABLE technical_versions (
    id BIGSERIAL PRIMARY KEY,
    edition_id BIGINT NOT NULL REFERENCES media_editions(id) ON DELETE CASCADE,
    container TEXT,
    video_codec TEXT,
    width INTEGER,
    height INTEGER,
    hdr TEXT,
    bitrate BIGINT,
    selected BOOLEAN NOT NULL DEFAULT false
);

ALTER TABLE inventory_files ADD COLUMN IF NOT EXISTS identity_candidate_id BIGINT REFERENCES identity_candidates(id);
ALTER TABLE inventory_files ADD COLUMN IF NOT EXISTS technical_version_id BIGINT REFERENCES technical_versions(id);
CREATE INDEX IF NOT EXISTS identity_candidates_file_idx ON identity_candidates(inventory_file_id);
