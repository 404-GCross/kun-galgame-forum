-- 076: point the todo / update_log feed links at their real pages.
--
-- Since 034 (and untouched by 074) both feed triggers wrote '/update' as the
-- link. '/update' is the section shell and had no index page, so a todo or
-- update-log card on the home "其他" feed jumped to an empty page. The todo
-- board now lives at /update/todo and the changelog at /update/history.
BEGIN;

CREATE OR REPLACE FUNCTION feed_sync_todo() RETURNS trigger AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN PERFORM feed_delete('TODO_CREATION', OLD.id); RETURN OLD; END IF;
    PERFORM feed_upsert('TODO_CREATION', NEW.id, NEW.user_id, 0, NEW.content, '/update/todo', false, NEW.created);
    RETURN NEW;
END $$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION feed_sync_update_log() RETURNS trigger AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN PERFORM feed_delete('UPDATE_LOG_CREATION', OLD.id); RETURN OLD; END IF;
    PERFORM feed_upsert('UPDATE_LOG_CREATION', NEW.id, NEW.user_id, 0, NEW.content, '/update/history', false, NEW.created);
    RETURN NEW;
END $$ LANGUAGE plpgsql;

UPDATE feed_activity SET link = '/update/todo'
WHERE type = 'TODO_CREATION' AND link = '/update';

UPDATE feed_activity SET link = '/update/history'
WHERE type = 'UPDATE_LOG_CREATION' AND link = '/update';

COMMIT;
