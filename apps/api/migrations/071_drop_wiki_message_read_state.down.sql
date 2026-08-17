-- Recreate the empty wiki read-state table for rollback symmetry.
-- Stripped wiki:* mute keys are not restored — they had no remaining producer.
BEGIN;

CREATE TABLE IF NOT EXISTS wiki_message_read_state (
    user_id              INT PRIMARY KEY,
    last_read_message_id BIGINT NOT NULL DEFAULT 0,
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMIT;
