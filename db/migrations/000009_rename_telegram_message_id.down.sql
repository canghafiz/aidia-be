-- ============================================================
-- MIGRATION: 000009_rename_telegram_message_id (DOWN)
-- ============================================================
-- Revert: Rename platform_message_id back to telegram_message_id
-- ============================================================

DO $$
DECLARE
    schema_record RECORD;
BEGIN
    FOR schema_record IN 
        SELECT schema_name 
        FROM information_schema.schemata 
        WHERE schema_name NOT IN ('public', 'information_schema', 'pg_catalog', 'pg_toast')
          AND schema_name NOT LIKE 'pg_%'
    LOOP
        BEGIN
            -- Rename column back
            EXECUTE format('
                ALTER TABLE %I.guest_message 
                RENAME COLUMN platform_message_id TO telegram_message_id
            ', schema_record.schema_name);
            
            -- Recreate index with old name
            EXECUTE format('
                DROP INDEX IF EXISTS %I.idx_guest_msg_platform_id
            ', schema_record.schema_name);
            
            EXECUTE format('
                CREATE INDEX idx_guest_msg_telegram_id ON %I.guest_message (telegram_message_id)
            ', schema_record.schema_name);
            
            RAISE NOTICE 'Reverted column in schema: %', schema_record.schema_name;
        EXCEPTION
            WHEN OTHERS THEN
                RAISE NOTICE 'Error processing schema %: %', schema_record.schema_name, SQLERRM;
        END;
    END LOOP;
END $$;
