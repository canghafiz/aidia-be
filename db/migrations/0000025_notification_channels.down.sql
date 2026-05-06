-- Remove second WA/email recipients from notification settings.

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
            'DELETE FROM %I.setting WHERE sub_group_name = $1 AND name = $2',
            v_schema
        ) USING 'Status New / Confirmed Order', 'new-order-wa-number-2';

        EXECUTE format(
            'DELETE FROM %I.setting WHERE sub_group_name = $1 AND name = $2',
            v_schema
        ) USING 'Status New / Confirmed Order', 'new-order-email-2';
    END LOOP;
END;
$$;
