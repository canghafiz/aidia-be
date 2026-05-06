-- Seed missing integration settings for all existing tenant schemas:
--   Telegram: timezone (default Asia/Singapore), bot-active-hours (if missing)
--   WhatsApp: bot-active-hours (if missing), whatsapp-webhook-token (if missing)

DO $$
DECLARE
    v_schema TEXT;
BEGIN
    FOR v_schema IN
        SELECT DISTINCT tenant_schema FROM public.users
        WHERE tenant_schema IS NOT NULL AND tenant_schema != ''
    LOOP
        EXECUTE format(
            'INSERT INTO %I.setting (group_name, sub_group_name, name, value) VALUES
                (''integration'', ''Telegram'', ''timezone'',         ''Asia/Singapore''),
                (''integration'', ''Telegram'', ''bot-active-hours'', '''')
             ON CONFLICT (sub_group_name, name) DO NOTHING',
            v_schema
        );

        EXECUTE format(
            'INSERT INTO %I.setting (group_name, sub_group_name, name, value) VALUES
                (''integration'', ''WhatsApp'', ''bot-active-hours'',       ''''),
                (''integration'', ''WhatsApp'', ''whatsapp-webhook-token'', '''')
             ON CONFLICT (sub_group_name, name) DO NOTHING',
            v_schema
        );
    END LOOP;
END;
$$;
