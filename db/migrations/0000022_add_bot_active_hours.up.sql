-- Add bot-active-hours to public.setting and all existing tenant schemas.
-- Empty value = bot is always active (no restriction).
-- When set to JSON (via OperationalHoursField), bot only auto-replies during those hours.

-- 1. Add to public settings
INSERT INTO public.setting (group_name, sub_group_name, name, value)
VALUES ('integration', 'Telegram', 'bot-active-hours', '')
ON CONFLICT (sub_group_name, name) DO NOTHING;

-- 2. Add to all existing tenant schemas
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
        ) USING 'integration', 'Telegram', 'bot-active-hours', '';
    END LOOP;
END;
$$;
