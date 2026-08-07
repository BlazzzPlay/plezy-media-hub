ALTER TABLE reorganization_plans
    ADD COLUMN IF NOT EXISTS source_sha256 CHAR(64),
    ADD COLUMN IF NOT EXISTS bytes BIGINT CHECK (bytes IS NULL OR bytes >= 0);
