-- 073: drop the four per-locale content columns 072 replaced.
--
-- deploy-then-drop: 073 is in cmd/migrate's default --exclude list, so a plain
-- deploy skips it. Run it by hand with `--only=073` once the single-content
-- code is live and has been observed. Dropping before that deploy would break
-- the running API, which still selects content_zh_cn.
BEGIN;

ALTER TABLE update_log
    DROP COLUMN IF EXISTS content_en_us,
    DROP COLUMN IF EXISTS content_ja_jp,
    DROP COLUMN IF EXISTS content_zh_cn,
    DROP COLUMN IF EXISTS content_zh_tw;

ALTER TABLE todo
    DROP COLUMN IF EXISTS content_en_us,
    DROP COLUMN IF EXISTS content_ja_jp,
    DROP COLUMN IF EXISTS content_zh_cn,
    DROP COLUMN IF EXISTS content_zh_tw;

ALTER TABLE system_message
    DROP COLUMN IF EXISTS content_en_us,
    DROP COLUMN IF EXISTS content_ja_jp,
    DROP COLUMN IF EXISTS content_zh_cn,
    DROP COLUMN IF EXISTS content_zh_tw;

COMMIT;
