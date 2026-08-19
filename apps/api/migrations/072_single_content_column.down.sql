-- Reverse of 072. The four locale columns still exist at this point (073 is
-- the drop), so undoing this only removes the new column.
BEGIN;

ALTER TABLE update_log DROP COLUMN IF EXISTS content;
ALTER TABLE todo DROP COLUMN IF EXISTS content;
ALTER TABLE system_message DROP COLUMN IF EXISTS content;

COMMIT;
