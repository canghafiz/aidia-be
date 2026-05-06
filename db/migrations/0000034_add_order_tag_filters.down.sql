DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN
        SELECT schema_name FROM information_schema.schemata
        WHERE schema_name NOT IN ('public','information_schema','pg_catalog','pg_toast')
          AND schema_name NOT LIKE 'pg_%'
    LOOP
        EXECUTE format(
            'ALTER TABLE %I.orders DROP COLUMN IF EXISTS tag_filter_ids',
            r.schema_name
        );
    END LOOP;
END;
$$;
