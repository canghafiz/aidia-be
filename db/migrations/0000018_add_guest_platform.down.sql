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
        EXECUTE format('ALTER TABLE %I.guest ADD COLUMN IF NOT EXISTS platform_username VARCHAR(100);', v_schema);
        EXECUTE format('DROP INDEX IF EXISTS %I.idx_guest_platform;', v_schema);
        EXECUTE format('ALTER TABLE %I.guest DROP COLUMN IF EXISTS platform;', v_schema);
    END LOOP;
END $$;
