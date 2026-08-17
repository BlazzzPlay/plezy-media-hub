ALTER TABLE inventory_files ADD COLUMN IF NOT EXISTS content_id BIGINT REFERENCES contents(id);
CREATE INDEX IF NOT EXISTS inventory_files_content_idx ON inventory_files(content_id);
