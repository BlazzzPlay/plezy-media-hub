ALTER TABLE reorganization_plans
    ADD COLUMN IF NOT EXISTS operation TEXT NOT NULL DEFAULT 'move';

ALTER TABLE reorganization_plans
    DROP CONSTRAINT IF EXISTS reorganization_plans_operation_check;

ALTER TABLE reorganization_plans
    ADD CONSTRAINT reorganization_plans_operation_check CHECK (operation IN ('copy', 'move'));
