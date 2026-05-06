-- Add second WA/email recipients to notification settings
-- for public schema and all existing tenant schemas.

DO $$
DECLARE
    v_schema TEXT;
    schemas TEXT[] := ARRAY['public'];
BEGIN
    FOR v_schema IN
        SELECT DISTINCT tenant_schema FROM public.users
        WHERE tenant_schema IS NOT NULL AND tenant_schema != ''
    LOOP
        schemas := schemas || v_schema;
    END LOOP;

    FOREACH v_schema IN ARRAY schemas LOOP
        EXECUTE format(
            'INSERT INTO %I.setting (group_name, sub_group_name, name, value)
             VALUES ($1, $2, $3, $4) ON CONFLICT (sub_group_name, name) DO NOTHING',
            v_schema
        ) USING 'notification', 'Status New / Confirmed Order', 'new-order-wa-number-2', '';

        EXECUTE format(
            'INSERT INTO %I.setting (group_name, sub_group_name, name, value)
             VALUES ($1, $2, $3, $4) ON CONFLICT (sub_group_name, name) DO NOTHING',
            v_schema
        ) USING 'notification', 'Status New / Confirmed Order', 'new-order-email-2', '';
    END LOOP;
END;
$$;
