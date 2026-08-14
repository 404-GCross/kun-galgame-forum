-- 070: the home feed follows `published`, the way every other reader does.
--
-- feed_sync_galgame (034) upserts a GALGAME_CREATION row on INSERT *and* on
-- UPDATE, and it never looked at 068's `published`. So the feed listed every
-- unpublished entry — 124 of them at the time of writing, i.e. every row the
-- lazy mint created for a draft or a withdrawn claim — while the browse list,
-- the RSS feed and the rankings all correctly hid them.
--
-- This also unblocks two things the trigger was silently vetoing: a submission
-- may now stamp its creator on the local row the moment it lands (row exists,
-- published false, feed silent), and a ban may unpublish instead of DELETEing
-- the row. The DELETE was the only way to get the entry out of the feed, and it
-- cascaded: galgame_comment, galgame_like, galgame_favorite and
-- galgame_resource are ON DELETE CASCADE off galgame.
BEGIN;

CREATE OR REPLACE FUNCTION feed_sync_galgame() RETURNS trigger AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN PERFORM feed_delete('GALGAME_CREATION', OLD.id); RETURN OLD; END IF;
    IF NOT NEW.published THEN PERFORM feed_delete('GALGAME_CREATION', NEW.id); RETURN NEW; END IF;
    PERFORM feed_upsert('GALGAME_CREATION', NEW.id, 0, NEW.id, '', '/galgame/' || NEW.id, false, NEW.created);
    RETURN NEW;
END $$ LANGUAGE plpgsql;

DELETE FROM feed_activity
WHERE type = 'GALGAME_CREATION'
  AND source_id IN (SELECT id FROM galgame WHERE NOT published);

COMMIT;
