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
            'ALTER TABLE %I.product DROP COLUMN IF EXISTS product_quantity',
            r.schema_name
        );
    END LOOP;
END;
$$;
