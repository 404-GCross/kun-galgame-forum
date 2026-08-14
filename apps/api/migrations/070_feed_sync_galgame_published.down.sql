-- Restores 034's function body. The feed rows deleted above are not restored:
-- re-listing unpublished entries is the bug, not the state worth returning to.
BEGIN;

CREATE OR REPLACE FUNCTION feed_sync_galgame() RETURNS trigger AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN PERFORM feed_delete('GALGAME_CREATION', OLD.id); RETURN OLD; END IF;
    PERFORM feed_upsert('GALGAME_CREATION', NEW.id, 0, NEW.id, '', '/galgame/' || NEW.id, false, NEW.created);
    RETURN NEW;
END $$ LANGUAGE plpgsql;

COMMIT;
