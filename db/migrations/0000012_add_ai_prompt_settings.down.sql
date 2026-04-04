-- Remove AI prompt settings from every existing tenant schema
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
            DELETE FROM %I.setting WHERE group_name = ''ai_prompt''
        ', v_schema);
    END LOOP;
END $$;
