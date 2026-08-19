-- Restore 034's bodies. Only meaningful while 073 has not run — once the
-- per-locale columns are gone these functions raise on every insert.
BEGIN;

CREATE OR REPLACE FUNCTION feed_sync_todo() RETURNS trigger AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN PERFORM feed_delete('TODO_CREATION', OLD.id); RETURN OLD; END IF;
    PERFORM feed_upsert('TODO_CREATION', NEW.id, NEW.user_id, 0, NEW.content_zh_cn, '/update', false, NEW.created);
    RETURN NEW;
END $$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION feed_sync_update_log() RETURNS trigger AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN PERFORM feed_delete('UPDATE_LOG_CREATION', OLD.id); RETURN OLD; END IF;
    PERFORM feed_upsert('UPDATE_LOG_CREATION', NEW.id, NEW.user_id, 0, NEW.content_zh_cn, '/update', false, NEW.created);
    RETURN NEW;
END $$ LANGUAGE plpgsql;

COMMIT;
