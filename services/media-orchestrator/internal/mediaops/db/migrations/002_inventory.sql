CREATE TABLE inventory_files (
    id BIGSERIAL PRIMARY KEY,
    path TEXT NOT NULL UNIQUE,
    relative_path TEXT NOT NULL,
    size_bytes BIGINT NOT NULL CHECK (size_bytes >= 0),
    modified_at TIMESTAMPTZ NOT NULL,
    sha256 CHAR(64) NOT NULL UNIQUE,
    container TEXT,
    video_codec TEXT,
    width INTEGER CHECK (width IS NULL OR width > 0),
    height INTEGER CHECK (height IS NULL OR height > 0),
    duration_seconds DOUBLE PRECISION CHECK (duration_seconds IS NULL OR duration_seconds >= 0),
    status TEXT NOT NULL DEFAULT 'discovered' CHECK (status IN ('discovered','identified','needs_review','rejected')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX inventory_files_sha256_idx ON inventory_files(sha256);
