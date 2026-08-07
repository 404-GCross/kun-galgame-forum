-- 069: who has edited a galgame, kept locally and authoritatively.
--
-- The wiki owned this fact and froze with it (doc 126 D6), which is why the
-- detail page has been shipping an empty contributor strip. The catalog cannot
-- own it either: it records revisions per ENTITY, in registry ids, for every
-- tenant at once — kungal's question ("who worked on THIS game, in gid space")
-- is a projection of that feed, and a projection is a table.
--
-- Two writers fill it, and `source` says which:
--   0 = the one-shot wiki seed (cmd/seed-contributors, the frozen 17,966-row
--       ledger), which carries no revision counts — only that the person was
--       there, and when;
--   1 = the forward revision sync, which counts.
--
-- Deliberately NO foreign key to galgame. A contribution can predate the local
-- row (the seed replays wiki-era history), and a draft work has no local row at
-- all — an FK would silently drop exactly the rows this table exists to keep.
BEGIN;

CREATE TABLE IF NOT EXISTS galgame_contributor (
  id             BIGSERIAL PRIMARY KEY,
  galgame_id     BIGINT NOT NULL,
  user_id        BIGINT NOT NULL,
  first_at       TIMESTAMPTZ NOT NULL,
  last_at        TIMESTAMPTZ NOT NULL,
  revision_count INT NOT NULL DEFAULT 0,
  source         SMALLINT NOT NULL DEFAULT 0
);

CREATE UNIQUE INDEX IF NOT EXISTS uniq_galgame_contributor
  ON galgame_contributor (galgame_id, user_id);
CREATE INDEX IF NOT EXISTS idx_galgame_contributor_galgame
  ON galgame_contributor (galgame_id);
CREATE INDEX IF NOT EXISTS idx_galgame_contributor_user
  ON galgame_contributor (user_id);

COMMENT ON COLUMN galgame_contributor.source IS
  '0 = wiki-era seed (cmd/seed-contributors), 1 = catalog revision sync. Never rewritten on conflict.';

COMMIT;
