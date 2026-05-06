-- Revert: rename whatsapp-phone-number-id back to whatsapp-phone-id in all tenant schemas.

DO $$
DECLARE
    v_schema TEXT;
BEGIN
    FOR v_schema IN
        SELECT DISTINCT tenant_schema FROM public.users
        WHERE tenant_schema IS NOT NULL AND tenant_schema != ''
    LOOP
        EXECUTE format(
            'UPDATE %I.setting
             SET name = ''whatsapp-phone-id''
             WHERE sub_group_name = ''WhatsApp''
               AND name = ''whatsapp-phone-number-id''',
            v_schema
        );
    END LOOP;
END;
$$;
