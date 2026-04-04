-- Revert: rename platform columns back to telegram columns
DO $$
DECLARE
    v_schema TEXT;
BEGIN
    FOR v_schema IN
        SELECT DISTINCT tenant_schema
        FROM public.users
        WHERE tenant_schema IS NOT NULL
          AND tenant_schema != ''
    LOOP
        EXECUTE format('
            ALTER TABLE %I.guest
                RENAME COLUMN platform_chat_id TO telegram_chat_id;
            ALTER TABLE %I.guest
                RENAME COLUMN platform_username TO telegram_username;
        ', v_schema, v_schema);

        EXECUTE format('
            DO $inner$
            BEGIN
                IF EXISTS (
                    SELECT 1 FROM pg_indexes
                    WHERE schemaname = %L AND indexname = ''idx_guest_platform_chat''
                ) THEN
                    ALTER INDEX %I.idx_guest_platform_chat RENAME TO idx_guest_telegram_chat;
                END IF;
            END $inner$;
        ', v_schema, v_schema);
    END LOOP;
END $$;
