-- ============================================================
-- MIGRATION: 0000016_add_hitpay_settings (UP)
-- Add HitPay integration settings and active gateway selector
-- ============================================================

-- Platform HitPay keys (Aidia subscription payments)
INSERT INTO public.setting (group_name, sub_group_name, name, value)
VALUES
    ('integration', 'HitPay Aidia', 'hitpay-aidia-api-key',       ''),
    ('integration', 'HitPay Aidia', 'hitpay-aidia-webhook-salt',   ''),
    ('integration', 'HitPay Aidia', 'hitpay-aidia-sandbox',        'false')
ON CONFLICT (sub_group_name, name) DO NOTHING;

-- Active payment gateway selector (stripe | hitpay)
INSERT INTO public.setting (group_name, sub_group_name, name, value)
VALUES
    ('integration', 'Payment Gateway', 'active-payment-gateway', 'stripe')
ON CONFLICT (sub_group_name, name) DO NOTHING;

-- Per-tenant HitPay client keys (seeded into new tenant schemas going forward)
-- Existing tenants get the keys via the tenant schema template below
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
                'INSERT INTO %I.setting (group_name, sub_group_name, name, value)
                 VALUES
                     (''integration'', ''HitPay Client'', ''hitpay-client-api-key'',     ''''),
                     (''integration'', ''HitPay Client'', ''hitpay-client-webhook-salt'', '''')
                 ON CONFLICT (sub_group_name, name) DO NOTHING',
                schema_record.schema_name
            );
            RAISE NOTICE 'Seeded HitPay client keys for schema: %', schema_record.schema_name;
        EXCEPTION WHEN OTHERS THEN
            RAISE NOTICE 'Skipped schema %: %', schema_record.schema_name, SQLERRM;
        END;
    END LOOP;
END $$;
