-- 072: collapse update_log / todo / system_message onto one `content` column.
--
-- These three tables carried content_en_us / content_ja_jp / content_zh_cn /
-- content_zh_tw. Only content_zh_cn was ever rendered — the admin form wrote
-- content_en_us on every row (1234/1234 update_log, 161/188 todo) and no page
-- ever read it back; ja_jp and zh_tw were never written at all. The user ruled
-- the non-zh_cn text is legacy data and is to be dropped, not archived.
--
-- Adding + backfilling here, dropping in 073, so a rollback between the two
-- deploys still has its old columns. migrate runs to completion before the API
-- starts (docker-compose.prod.yml depends_on), so the new code never sees the
-- schema without `content`.
BEGIN;

ALTER TABLE update_log ADD COLUMN IF NOT EXISTS content text NOT NULL DEFAULT '';
ALTER TABLE todo ADD COLUMN IF NOT EXISTS content text NOT NULL DEFAULT '';
ALTER TABLE system_message ADD COLUMN IF NOT EXISTS content text NOT NULL DEFAULT '';

UPDATE update_log SET content = content_zh_cn WHERE content = '' AND content_zh_cn <> '';
UPDATE todo SET content = content_zh_cn WHERE content = '' AND content_zh_cn <> '';
UPDATE system_message SET content = content_zh_cn WHERE content = '' AND content_zh_cn <> '';

COMMIT;
