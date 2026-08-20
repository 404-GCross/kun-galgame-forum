-- 076 down: restore the '/update' feed links.
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

UPDATE feed_activity SET link = '/update'
WHERE type = 'TODO_CREATION' AND link = '/update/todo';

UPDATE feed_activity SET link = '/update'
WHERE type = 'UPDATE_LOG_CREATION' AND link = '/update/history';

COMMIT;
