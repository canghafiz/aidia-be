-- ============================================================
-- MIGRATION: 000008_fix_setting_unique_constraint (DOWN)
-- ============================================================

-- Revert back to unique on name only
DO $$
DECLARE
    schema_record RECORD;
BEGIN
    -- For public schema
    ALTER TABLE public.setting DROP CONSTRAINT IF EXISTS uq_setting_subgroup_name;
    
    -- For all tenant schemas
    FOR schema_record IN 
        SELECT schema_name 
        FROM information_schema.schemata 
        WHERE schema_name NOT IN ('public', 'information_schema', 'pg_catalog', 'pg_toast')
          AND schema_name NOT LIKE 'pg_%'
    LOOP
        BEGIN
            EXECUTE format('
                ALTER TABLE %I.setting DROP CONSTRAINT IF EXISTS uq_setting_subgroup_name
            ', schema_record.schema_name);
        EXCEPTION
            WHEN OTHERS THEN
                RAISE NOTICE 'Error processing schema %: %', schema_record.schema_name, SQLERRM;
        END;
    END LOOP;
END $$;
