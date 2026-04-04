-- ============================================================
-- MIGRATION: 0000011_add_stripe_fields_to_order_payments (UP)
-- ============================================================
-- Add Stripe session fields to order_payments table
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
            -- Add stripe_session_id column
            EXECUTE format('
                ALTER TABLE %I.order_payments 
                ADD COLUMN IF NOT EXISTS stripe_session_id VARCHAR(255),
                ADD COLUMN IF NOT EXISTS stripe_session_url TEXT,
                ADD COLUMN IF NOT EXISTS stripe_payment_status VARCHAR(50),
                ADD COLUMN IF NOT EXISTS stripe_invoice_id VARCHAR(255),
                ADD COLUMN IF NOT EXISTS paid_at TIMESTAMPTZ,
                ADD COLUMN IF NOT EXISTS is_paid BOOLEAN NOT NULL DEFAULT FALSE;
            ', schema_record.schema_name);
            
            RAISE NOTICE 'Added Stripe columns to schema: %', schema_record.schema_name;
        EXCEPTION
            WHEN OTHERS THEN
                RAISE NOTICE 'Error processing schema %: %', schema_record.schema_name, SQLERRM;
        END;
    END LOOP;
END $$;
