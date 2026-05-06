-- Rename whatsapp-phone-id → whatsapp-phone-number-id in all tenant schemas.
-- Migration 0000027 already seeds the correct name; this fixes tenants that were
-- created before 0000027 and still have the old field name.

DO $$
DECLARE
    v_schema TEXT;
BEGIN
    FOR v_schema IN
        SELECT DISTINCT tenant_schema FROM public.users
        WHERE tenant_schema IS NOT NULL AND tenant_schema != ''
    LOOP
        -- Rename old field if it exists and the new name does not yet exist
        EXECUTE format(
            'UPDATE %I.setting
             SET name = ''whatsapp-phone-number-id''
             WHERE sub_group_name = ''WhatsApp''
               AND name = ''whatsapp-phone-id''
               AND NOT EXISTS (
                   SELECT 1 FROM %I.setting
                   WHERE sub_group_name = ''WhatsApp'' AND name = ''whatsapp-phone-number-id''
               )',
            v_schema, v_schema
        );

        -- If both existed (old and new), delete the old duplicate
        EXECUTE format(
            'DELETE FROM %I.setting
             WHERE sub_group_name = ''WhatsApp''
               AND name = ''whatsapp-phone-id''',
            v_schema
        );
    END LOOP;
END;
$$;
