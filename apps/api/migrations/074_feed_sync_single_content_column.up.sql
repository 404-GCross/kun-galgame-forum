-- 074: point the todo / update_log feed triggers at 072's `content` column.
--
-- 034 wrote both trigger bodies against NEW.content_zh_cn and nothing has
-- touched them since. 072 added `content`, the Go models stopped writing the
-- per-locale columns, and the sweep missed the triggers because PL/pgSQL bodies
-- are parsed at call time, not at CREATE FUNCTION time — nothing in the build,
-- the schema check or the test suite names a column that lives inside a string.
--
-- Two symptoms, one cause. Live since the single-content deploy: every new
-- 更新日志 / 待办 lands in feed_activity with content = '' (the column the
-- trigger reads is now never written), so the home feed card renders its label
-- and version badge above an empty paragraph. And once 073 drops the column,
-- the next INSERT fails outright:
--
--   ERROR:  record "new" has no field "content_zh_cn"
--   CONTEXT:  PL/pgSQL function feed_sync_todo() line 4 at PERFORM
--
-- DROP COLUMN does not complain, because Postgres tracks no dependency from a
-- function body to a column. So 074 must be applied BEFORE 073 is run by hand.
BEGIN;

CREATE OR REPLACE FUNCTION feed_sync_todo() RETURNS trigger AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN PERFORM feed_delete('TODO_CREATION', OLD.id); RETURN OLD; END IF;
    PERFORM feed_upsert('TODO_CREATION', NEW.id, NEW.user_id, 0, NEW.content, '/update', false, NEW.created);
    RETURN NEW;
END $$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION feed_sync_update_log() RETURNS trigger AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN PERFORM feed_delete('UPDATE_LOG_CREATION', OLD.id); RETURN OLD; END IF;
    PERFORM feed_upsert('UPDATE_LOG_CREATION', NEW.id, NEW.user_id, 0, NEW.content, '/update', false, NEW.created);
    RETURN NEW;
END $$ LANGUAGE plpgsql;

-- Repair the rows written blank between the single-content deploy and this one.
UPDATE feed_activity f SET content = t.content
FROM todo t WHERE f.type = 'TODO_CREATION' AND f.source_id = t.id
  AND f.content = '' AND t.content <> '';

UPDATE feed_activity f SET content = u.content
FROM update_log u WHERE f.type = 'UPDATE_LOG_CREATION' AND f.source_id = u.id
  AND f.content = '' AND u.content <> '';

COMMIT;
