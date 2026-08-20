-- 075 down: drop the claimer column.
BEGIN;

ALTER TABLE todo DROP COLUMN IF EXISTS claimed_user_id;

COMMIT;
