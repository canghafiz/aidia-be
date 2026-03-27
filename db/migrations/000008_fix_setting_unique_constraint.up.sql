-- ============================================================
-- MIGRATION: 000008_fix_setting_unique_constraint (UP)
-- ============================================================
-- This migration is IDEMPOTENT - safe to run multiple times
-- ============================================================

-- Fix public schema
DO $$
BEGIN
    -- Drop old constraint if exists (any name pattern)
    ALTER TABLE public.setting DROP CONSTRAINT IF EXISTS setting_name_key;
    ALTER TABLE public.setting DROP CONSTRAINT IF EXISTS uq_setting_name;
    ALTER TABLE public.setting DROP CONSTRAINT IF EXISTS uq_setting_subgroup_name;
    
    -- Add new composite unique constraint
    ALTER TABLE public.setting ADD CONSTRAINT uq_setting_subgroup_name UNIQUE (sub_group_name, name);
EXCEPTION
    WHEN OTHERS THEN
        RAISE NOTICE 'Error processing public schema: %', SQLERRM;
END $$;

-- Fix all tenant schemas
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
            -- Drop old constraints
            EXECUTE format('ALTER TABLE %I.setting DROP CONSTRAINT IF EXISTS setting_name_key', schema_record.schema_name);
            EXECUTE format('ALTER TABLE %I.setting DROP CONSTRAINT IF EXISTS uq_setting_name', schema_record.schema_name);
            EXECUTE format('ALTER TABLE %I.setting DROP CONSTRAINT IF EXISTS uq_setting_subgroup_name', schema_record.schema_name);
            
            -- Add new constraint
            EXECUTE format('ALTER TABLE %I.setting ADD CONSTRAINT uq_setting_subgroup_name UNIQUE (sub_group_name, name)', schema_record.schema_name);
        EXCEPTION
            WHEN OTHERS THEN
                RAISE NOTICE 'Error processing schema %: %', schema_record.schema_name, SQLERRM;
        END;
    END LOOP;
END $$;
