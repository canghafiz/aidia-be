-- ============================================================
-- MIGRATION: 0000010_add_stripe_and_expire_function (UP)
-- ============================================================
-- Add Stripe columns to order_payments AND create expire function
-- ============================================================

-- Add Stripe columns to all tenant schemas
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
            -- Add Stripe columns to order_payments
            EXECUTE format('
                ALTER TABLE %I.order_payments 
                ADD COLUMN IF NOT EXISTS stripe_session_id VARCHAR(255),
                ADD COLUMN IF NOT EXISTS stripe_session_url TEXT,
                ADD COLUMN IF NOT EXISTS stripe_payment_status VARCHAR(50),
                ADD COLUMN IF NOT EXISTS stripe_invoice_id VARCHAR(255),
                ADD COLUMN IF NOT EXISTS paid_at TIMESTAMPTZ,
                ADD COLUMN IF NOT EXISTS is_paid BOOLEAN NOT NULL DEFAULT FALSE
            ', schema_record.schema_name);
            
            RAISE NOTICE 'Added Stripe columns to schema: %', schema_record.schema_name;
        EXCEPTION
            WHEN OTHERS THEN
                RAISE NOTICE 'Error processing schema %: %', schema_record.schema_name, SQLERRM;
        END;
    END LOOP;
END $$;

-- Create fn_expire_orders function for all tenant schemas
DO $$
DECLARE
    schema_record RECORD;
    func_sql TEXT;
BEGIN
    FOR schema_record IN 
        SELECT schema_name 
        FROM information_schema.schemata 
        WHERE schema_name NOT IN ('public', 'information_schema', 'pg_catalog', 'pg_toast')
          AND schema_name NOT LIKE 'pg_%'
    LOOP
        BEGIN
            -- Build function SQL
            func_sql := format('
                CREATE OR REPLACE FUNCTION %I.fn_expire_orders()
                RETURNS INTEGER AS $func$
                DECLARE
                    expired_count INTEGER;
                BEGIN
                    UPDATE %I.order_payments
                    SET payment_status = ''Voided''
                    WHERE payment_status = ''Unpaid''
                      AND expire_at < NOW();
                    
                    GET DIAGNOSTICS expired_count = ROW_COUNT;
                    
                    UPDATE %I.orders
                    SET status = ''Cancelled''
                    WHERE id IN (
                        SELECT order_id 
                        FROM %I.order_payments 
                        WHERE payment_status = ''Voided''
                    )
                    AND status = ''Pending'';
                    
                    RAISE NOTICE ''Expired %% unpaid orders'', expired_count;
                    
                    RETURN expired_count;
                END;
                $func$ LANGUAGE plpgsql
            ', schema_record.schema_name, schema_record.schema_name, schema_record.schema_name, schema_record.schema_name);
            
            -- Execute function creation
            EXECUTE func_sql;
            
            RAISE NOTICE 'Created fn_expire_orders in schema: %', schema_record.schema_name;
        EXCEPTION
            WHEN OTHERS THEN
                RAISE NOTICE 'Error processing schema %: %', schema_record.schema_name, SQLERRM;
        END;
    END LOOP;
END $$;
