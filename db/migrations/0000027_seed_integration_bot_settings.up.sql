-- Seed bot-enabled, manual-mode for Telegram and WhatsApp into every existing tenant schema.
-- Also copies ai-operational-prompt from public.setting into each client schema
-- (user may have configured it there). Falls back to default JSON if public has none.

DO $$
DECLARE
    v_schema TEXT;
    default_hours TEXT := '{"monday":{"start":"09:00","end":"21:00","closed":false},"tuesday":{"start":"09:00","end":"21:00","closed":false},"wednesday":{"start":"09:00","end":"21:00","closed":false},"thursday":{"start":"09:00","end":"21:00","closed":false},"friday":{"start":"09:00","end":"21:00","closed":false},"saturday":{"start":"09:00","end":"21:00","closed":false},"sunday":{"start":"09:00","end":"21:00","closed":true}}';
    public_hours TEXT;
BEGIN
    -- Get the configured value from public.setting (user may have set it there)
    SELECT value INTO public_hours
    FROM public.setting
    WHERE sub_group_name = 'AI Operational' AND name = 'ai-operational-prompt'
    LIMIT 1;

    -- Use default if public has nothing or still plain-text
    IF public_hours IS NULL OR public_hours = ''
       OR (public_hours NOT LIKE '{%' AND public_hours NOT LIKE '[%') THEN
        public_hours := default_hours;
    END IF;

    FOR v_schema IN
        SELECT DISTINCT tenant_schema FROM public.users
        WHERE tenant_schema IS NOT NULL AND tenant_schema != ''
    LOOP
        -- ai-operational-prompt: copy from public if not already in client schema
        EXECUTE format(
            'INSERT INTO %I.setting (group_name, sub_group_name, name, value)
             VALUES (''ai_prompt'', ''AI Operational'', ''ai-operational-prompt'', %L)
             ON CONFLICT (sub_group_name, name) DO NOTHING',
            v_schema, public_hours
        );

        -- If client schema has old plain-text format, upgrade to JSON
        EXECUTE format(
            'UPDATE %I.setting
             SET value = %L
             WHERE sub_group_name = ''AI Operational''
               AND name = ''ai-operational-prompt''
               AND (value = '''' OR (value NOT LIKE ''{%%'' AND value NOT LIKE ''[%%''))',
            v_schema, public_hours
        );

        -- store-operational-hours: separate setting for order blocking logic (same JSON structure)
        EXECUTE format(
            'INSERT INTO %I.setting (group_name, sub_group_name, name, value)
             VALUES (''ai_prompt'', ''Store Operational'', ''store-operational-hours'', %L)
             ON CONFLICT (sub_group_name, name) DO NOTHING',
            v_schema, public_hours
        );

        -- Telegram bot settings
        EXECUTE format(
            'INSERT INTO %I.setting (group_name, sub_group_name, name, value) VALUES
                (''integration'', ''Telegram'', ''bot-enabled'',      ''true''),
                (''integration'', ''Telegram'', ''manual-mode'',      ''false''),
                (''integration'', ''Telegram'', ''bot-active-hours'', '''')
             ON CONFLICT (sub_group_name, name) DO NOTHING',
            v_schema
        );

        -- WhatsApp bot settings
        EXECUTE format(
            'INSERT INTO %I.setting (group_name, sub_group_name, name, value) VALUES
                (''integration'', ''WhatsApp'', ''bot-enabled'',              ''true''),
                (''integration'', ''WhatsApp'', ''manual-mode'',              ''false''),
                (''integration'', ''WhatsApp'', ''bot-active-hours'',         ''''),
                (''integration'', ''WhatsApp'', ''whatsapp-phone-number-id'', ''''),
                (''integration'', ''WhatsApp'', ''whatsapp-token'',           ''''),
                (''integration'', ''WhatsApp'', ''whatsapp-api-version'',     ''v20.0'')
             ON CONFLICT (sub_group_name, name) DO NOTHING',
            v_schema
        );
    END LOOP;
END;
$$;
