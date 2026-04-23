-- ============================================================
-- MIGRATION: 0000016_add_hitpay_settings (DOWN)
-- ============================================================

DELETE FROM public.setting
WHERE sub_group_name = 'HitPay Aidia';

DELETE FROM public.setting
WHERE sub_group_name = 'Payment Gateway'
  AND name = 'active-payment-gateway';

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
                'DELETE FROM %I.setting WHERE sub_group_name = ''HitPay Client''',
                schema_record.schema_name
            );
        EXCEPTION WHEN OTHERS THEN
            RAISE NOTICE 'Skipped schema %: %', schema_record.schema_name, SQLERRM;
        END;
    END LOOP;
END $$;
