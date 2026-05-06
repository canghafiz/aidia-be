-- Add bot-enabled, manual-mode, and bot-active-hours to integration/WhatsApp
-- for public settings and all existing tenant schemas.

-- 1. Public settings
INSERT INTO public.setting (group_name, sub_group_name, name, value)
VALUES
    ('integration', 'WhatsApp', 'bot-enabled', 'true'),
    ('integration', 'WhatsApp', 'manual-mode', 'false'),
    ('integration', 'WhatsApp', 'bot-active-hours', '')
ON CONFLICT (sub_group_name, name) DO NOTHING;

-- 2. All existing tenant schemas
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
        EXECUTE format(
            'INSERT INTO %I.setting (group_name, sub_group_name, name, value)
             VALUES ($1, $2, $3, $4)
             ON CONFLICT (sub_group_name, name) DO NOTHING',
            v_schema
        ) USING 'integration', 'WhatsApp', 'bot-enabled', 'true';

        EXECUTE format(
            'INSERT INTO %I.setting (group_name, sub_group_name, name, value)
             VALUES ($1, $2, $3, $4)
             ON CONFLICT (sub_group_name, name) DO NOTHING',
            v_schema
        ) USING 'integration', 'WhatsApp', 'manual-mode', 'false';

        EXECUTE format(
            'INSERT INTO %I.setting (group_name, sub_group_name, name, value)
             VALUES ($1, $2, $3, $4)
             ON CONFLICT (sub_group_name, name) DO NOTHING',
            v_schema
        ) USING 'integration', 'WhatsApp', 'bot-active-hours', '';
    END LOOP;
END;
$$;
