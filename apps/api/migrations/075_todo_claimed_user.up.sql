-- 075: record who claimed a todo entry.
--
-- The todo list becomes a small task board: a pending entry (status 0) can be
-- claimed by a user with update_log.edit (status 0 -> 1), and only that claimer
-- may then complete (1 -> 2) or discard (1 -> 3) it. claimed_user_id holds the
-- claimer's user id so the list can render "已被 X 认领" and gate the
-- complete/discard buttons.
BEGIN;

ALTER TABLE todo ADD COLUMN IF NOT EXISTS claimed_user_id integer;

COMMIT;
