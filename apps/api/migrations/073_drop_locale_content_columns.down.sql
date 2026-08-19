-- Reverse of 073. Recreates the columns empty — the text they held is gone,
-- which is the intended outcome of the drop, not an accident of this rollback.
BEGIN;

ALTER TABLE update_log
    ADD COLUMN IF NOT EXISTS content_en_us text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS content_ja_jp text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS content_zh_cn text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS content_zh_tw text NOT NULL DEFAULT '';

ALTER TABLE todo
    ADD COLUMN IF NOT EXISTS content_en_us text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS content_ja_jp text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS content_zh_cn text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS content_zh_tw text NOT NULL DEFAULT '';

ALTER TABLE system_message
    ADD COLUMN IF NOT EXISTS content_en_us text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS content_ja_jp text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS content_zh_cn text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS content_zh_tw text NOT NULL DEFAULT '';

COMMIT;
