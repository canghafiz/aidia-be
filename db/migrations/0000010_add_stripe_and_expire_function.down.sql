-- ============================================================
-- MIGRATION: 0000010_add_stripe_and_expire_function (DOWN)
-- ============================================================
-- Drop Stripe columns and fn_expire_orders function
-- ============================================================

-- Drop Stripe columns from all tenant schemas
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
                DROP COLUMN IF EXISTS is_paid
            ', schema_record.schema_name);
            
            RAISE NOTICE 'Dropped Stripe columns from schema: %', schema_record.schema_name;
        EXCEPTION
            WHEN OTHERS THEN
                RAISE NOTICE 'Error processing schema %: %', schema_record.schema_name, SQLERRM;
        END;
    END LOOP;
END $$;

-- Drop fn_expire_orders function from all tenant schemas
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
                DROP FUNCTION IF EXISTS %I.fn_expire_orders()
            ', schema_record.schema_name);
            
            RAISE NOTICE 'Dropped fn_expire_orders from schema: %', schema_record.schema_name;
        EXCEPTION
            WHEN OTHERS THEN
                RAISE NOTICE 'Error processing schema %: %', schema_record.schema_name, SQLERRM;
        END;
    END LOOP;
END $$;
