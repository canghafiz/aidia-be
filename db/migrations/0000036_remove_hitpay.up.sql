-- ============================================================
-- MIGRATION: 0000036_remove_hitpay (UP)
-- Remove all HitPay-related settings, ensure Stripe Aidia settings exist
-- ============================================================

-- Remove HitPay Aidia keys and Payment Gateway selector from public settings
DELETE FROM public.setting
WHERE sub_group_name IN ('HitPay Aidia', 'Payment Gateway');

-- Ensure Stripe Aidia settings exist in public.setting (for SuperAdmin payment integration)
INSERT INTO public.setting (group_name, sub_group_name, name, value)
VALUES
    ('integration', 'Stripe Aidia', 'stripe-aidia-public-key', ''),
    ('integration', 'Stripe Aidia', 'stripe-aidia-secret-key', ''),
    ('integration', 'Stripe Aidia', 'stripe-aidia-webhook-secret', '')
ON CONFLICT (sub_group_name, name) DO NOTHING;

-- Remove HitPay Client keys from all tenant schemas (if any remain)
DO $$
DECLARE
    schema_record RECORD;
BEGIN
    FOR schema_record IN
        SELECT schema_name
        FROM information_schema.schemata
        WHERE schema_name NOT IN ('public', 'information_schema', 'pg_catalog', 'pg_toast')
          AND schema_name NOT LIKE 'pg_%'
    LOOP
        BEGIN
            EXECUTE format(
                'DELETE FROM %I.setting WHERE sub_group_name IN (''HitPay Client'', ''Payment Gateway'')',
                schema_record.schema_name
            );
        EXCEPTION WHEN OTHERS THEN
            RAISE NOTICE 'Error cleaning schema %: %', schema_record.schema_name, SQLERRM;
        END;
    END LOOP;
END $$;
