-- Remove bot-active-hours from public and all tenant schemas
DELETE FROM public.setting
WHERE group_name = 'integration' AND sub_group_name = 'Telegram' AND name = 'bot-active-hours';

DO $$
DECLARE
    v_schema TEXT;
BEGIN
    FOR v_schema IN
        SELECT DISTINCT tenant_schema FROM public.users
        WHERE tenant_schema IS NOT NULL AND tenant_schema != ''
    LOOP
        EXECUTE format(
            'DELETE FROM %I.setting WHERE group_name=$1 AND sub_group_name=$2 AND name=$3',
            v_schema
        ) USING 'integration', 'Telegram', 'bot-active-hours';
    END LOOP;
END;
$$;
