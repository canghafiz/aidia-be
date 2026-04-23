-- ============================================================
-- MIGRATION: 0000015_rename_stripe_to_payment_gateway (DOWN)
-- Rollback: kembalikan nama kolom ke stripe_* semula
-- ============================================================

-- ============================================================
-- 1. public.tenant_plan
-- ============================================================
ALTER TABLE public.tenant_plan
    RENAME COLUMN payment_session_id      TO stripe_session_id;
ALTER TABLE public.tenant_plan
    RENAME COLUMN payment_session_url     TO stripe_session_url;
ALTER TABLE public.tenant_plan
    RENAME COLUMN payment_gateway_status  TO stripe_payment_status;
ALTER TABLE public.tenant_plan
    RENAME COLUMN payment_gateway_message TO stripe_payment_message;
ALTER TABLE public.tenant_plan
    RENAME COLUMN subscription_invoice_id TO stripe_subscription_invoice_id;

ALTER TABLE public.tenant_plan DROP COLUMN IF EXISTS payment_gateway;

DROP INDEX IF EXISTS idx_tenant_plan_payment_session;
DROP INDEX IF EXISTS idx_tenant_plan_subscription_invoice;
CREATE INDEX IF NOT EXISTS idx_tenant_plan_stripe_session ON public.tenant_plan (stripe_session_id);
CREATE INDEX IF NOT EXISTS idx_tenant_plan_stripe_sub     ON public.tenant_plan (stripe_subscription_invoice_id);

-- ============================================================
-- 2. Per-tenant order_payments
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
            IF EXISTS (
                SELECT 1 FROM information_schema.columns
                WHERE table_schema = schema_record.schema_name
                  AND table_name = 'order_payments'
                  AND column_name = 'payment_session_id'
            ) THEN
                EXECUTE format('ALTER TABLE %I.order_payments RENAME COLUMN payment_session_id     TO stripe_session_id',     schema_record.schema_name);
                EXECUTE format('ALTER TABLE %I.order_payments RENAME COLUMN payment_session_url    TO stripe_session_url',    schema_record.schema_name);
                EXECUTE format('ALTER TABLE %I.order_payments RENAME COLUMN payment_gateway_status TO stripe_payment_status', schema_record.schema_name);
                EXECUTE format('ALTER TABLE %I.order_payments RENAME COLUMN payment_invoice_id     TO stripe_invoice_id',     schema_record.schema_name);
                EXECUTE format('ALTER TABLE %I.order_payments DROP COLUMN IF EXISTS payment_gateway', schema_record.schema_name);

                RAISE NOTICE 'Rolled back schema: %', schema_record.schema_name;
            END IF;
        EXCEPTION WHEN OTHERS THEN
            RAISE NOTICE 'Error rolling back schema %: %', schema_record.schema_name, SQLERRM;
        END;
    END LOOP;
END $$;
