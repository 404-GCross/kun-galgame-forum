-- 068: the local galgame row stops being the visibility switch.
--
-- The row is a CONTAINER for local interaction state (view, likes, favorites,
-- ratings, resources), and every interaction path lazy-mints it for whatever
-- gid the caller names — including a catalog work that is still a draft, or one
-- kungal never claimed. Since local-row existence is what the browse list, the
-- home rail, the RSS feed and the rankings are built from, that lazy mint has
-- been publishing entries nobody ever published: "a row exists" was reading as
-- "this is public".
--
-- `published` separates the two. The row keeps holding user content whatever
-- happens to the claim; the flag alone decides whether kungal shows the entry,
-- and only the claim-event cron moves it (live → true, draft/declined → false,
-- hidden → the row is dropped as before).
--
-- Existing rows are set true so this migration changes nothing that is on
-- screen today: the leaks it makes fixable are cleaned by a separate data op
-- (charter 180 D4), which can be reviewed and rolled back on its own.
-- Everything minted from here on starts false — the DEFAULT is the point.
BEGIN;

ALTER TABLE galgame ADD COLUMN IF NOT EXISTS published boolean NOT NULL DEFAULT false;

UPDATE galgame SET published = true;

COMMENT ON COLUMN galgame.published IS
  'Public visibility (180). Row existence = local interaction container; this flag = listable. Written only by the claim-event cron.';

COMMIT;
