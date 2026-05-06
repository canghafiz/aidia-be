-- Revert: remove timezone and whatsapp-webhook-token from all tenant schemas.
-- (bot-active-hours is not removed as it may have user-configured values.)

DO $$
DECLARE
    v_schema TEXT;
BEGIN
    FOR v_schema IN
        SELECT DISTINCT tenant_schema FROM public.users
        WHERE tenant_schema IS NOT NULL AND tenant_schema != ''
    LOOP
        EXECUTE format(
            'DELETE FROM %I.setting
             WHERE (sub_group_name = ''Telegram'' AND name = ''timezone'')
                OR (sub_group_name = ''WhatsApp'' AND name = ''whatsapp-webhook-token'')',
            v_schema
        );
    END LOOP;
END;
$$;
