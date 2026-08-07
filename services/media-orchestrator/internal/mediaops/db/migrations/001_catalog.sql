CREATE TABLE contents (
    id BIGSERIAL PRIMARY KEY,
    category TEXT NOT NULL,
    kind TEXT NOT NULL CHECK (kind IN ('movie','series','anime','cartoon','documentary','novel','comedy','sports','music','podcast','festival','other')),
    title TEXT NOT NULL,
    year INTEGER CHECK (year IS NULL OR year BETWEEN 1800 AND 2200),
    external_id TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(category, kind, title, year, external_id)
);
CREATE INDEX contents_category_title_idx ON contents(category, title);

CREATE TABLE physical_files (
    id BIGSERIAL PRIMARY KEY,
    content_id BIGINT NOT NULL REFERENCES contents(id) ON DELETE CASCADE,
    path TEXT NOT NULL,
    sha256 CHAR(64),
    container TEXT,
    video_codec TEXT,
    width INTEGER,
    height INTEGER,
    status TEXT NOT NULL DEFAULT 'discovered' CHECK (status IN ('discovered','selected','rejected','needs_review')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(path),
    UNIQUE(sha256)
);
