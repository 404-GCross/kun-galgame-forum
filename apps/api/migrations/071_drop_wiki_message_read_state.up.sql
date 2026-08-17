-- 071: drop the last wiki-message leftovers on this database.
--
-- Wave 169 removed every reader of wiki_message_read_state (the mine /
-- read-state / admin message routes). The table has been an empty
-- historical shell since then; claim-event notifications use the local
-- message table, not this one. Dropping it is safe on a live deploy —
-- no code path selects or inserts here anymore.
--
-- The same wave left wiki:approved/declined/banned/unbanned in
-- kungal_user_state.muted_notification_types. Those keys have no UI and
-- no producer; strip them so a later save does not have to carry them.
BEGIN;

DROP TABLE IF EXISTS wiki_message_read_state;

UPDATE kungal_user_state
SET muted_notification_types = COALESCE((
    SELECT jsonb_agg(to_jsonb(elem))
    FROM jsonb_array_elements_text(muted_notification_types) AS elem
    WHERE elem NOT LIKE 'wiki:%'
), '[]'::jsonb)
WHERE muted_notification_types::text LIKE '%wiki:%';

COMMIT;
