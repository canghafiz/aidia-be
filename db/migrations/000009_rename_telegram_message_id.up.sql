-- ============================================================
-- MIGRATION: 000009_rename_telegram_message_id (UP)
-- ============================================================
-- Rename telegram_message_id to platform_message_id for multi-platform support
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
            -- Rename column
            EXECUTE format('
                ALTER TABLE %I.guest_message 
                RENAME COLUMN telegram_message_id TO platform_message_id
            ', schema_record.schema_name);
            
            -- Recreate index with new name
            EXECUTE format('
                DROP INDEX IF EXISTS %I.idx_guest_msg_telegram_id
            ', schema_record.schema_name);
            
            EXECUTE format('
                CREATE INDEX idx_guest_msg_platform_id ON %I.guest_message (platform_message_id)
            ', schema_record.schema_name);
            
            RAISE NOTICE 'Renamed column in schema: %', schema_record.schema_name;
        EXCEPTION
            WHEN OTHERS THEN
                RAISE NOTICE 'Error processing schema %: %', schema_record.schema_name, SQLERRM;
        END;
    END LOOP;
END $$;
