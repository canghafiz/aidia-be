-- ============================================================
-- MIGRATION: 0000011_add_stripe_fields_to_order_payments (DOWN)
-- ============================================================
-- Drop Stripe fields from order_payments table
-- ============================================================

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
            EXECUTE format('
                ALTER TABLE %I.order_payments 
                DROP COLUMN IF EXISTS stripe_session_id,
                DROP COLUMN IF EXISTS stripe_session_url,
                DROP COLUMN IF EXISTS stripe_payment_status,
                DROP COLUMN IF EXISTS stripe_invoice_id,
                DROP COLUMN IF EXISTS paid_at,
                DROP COLUMN IF EXISTS is_paid;
            ', schema_record.schema_name);
            
            RAISE NOTICE 'Dropped Stripe columns from schema: %', schema_record.schema_name;
        EXCEPTION
            WHEN OTHERS THEN
                RAISE NOTICE 'Error processing schema %: %', schema_record.schema_name, SQLERRM;
        END;
    END LOOP;
END $$;
