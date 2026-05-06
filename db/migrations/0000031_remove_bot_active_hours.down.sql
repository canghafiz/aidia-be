-- Rollback: re-seed bot-active-hours for Telegram and WhatsApp in all tenant schemas
DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN
        SELECT DISTINCT tenant_schema
        FROM public.users
        WHERE tenant_schema IS NOT NULL AND tenant_schema != ''
    LOOP
        EXECUTE format(
            'INSERT INTO %I.setting (group_name, sub_group_name, name, value)
             VALUES (''integration'', ''Telegram'', ''bot-active-hours'', '''')
             ON CONFLICT (sub_group_name, name) DO NOTHING',
            r.tenant_schema
        );
        EXECUTE format(
            'INSERT INTO %I.setting (group_name, sub_group_name, name, value)
             VALUES (''integration'', ''WhatsApp'', ''bot-active-hours'', '''')
             ON CONFLICT (sub_group_name, name) DO NOTHING',
            r.tenant_schema
        );
    END LOOP;
END;
$$;
