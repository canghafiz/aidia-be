-- Drop knowledge_base table from all tenant schemas
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
        EXECUTE format('DROP TABLE IF EXISTS %I.knowledge_base;', v_schema);
    END LOOP;
END $$;
