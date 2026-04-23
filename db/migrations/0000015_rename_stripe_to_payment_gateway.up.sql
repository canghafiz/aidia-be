-- ============================================================
-- MIGRATION: 0000015_rename_stripe_to_payment_gateway (UP)
-- Rename stripe_* columns to payment gateway agnostic names
-- untuk mendukung multi payment gateway (Stripe, Midtrans, dll)
-- Data yang ada TIDAK berubah, hanya nama kolom yang diganti.
-- ============================================================

-- ============================================================
-- 1. public.tenant_plan
-- ============================================================
ALTER TABLE public.tenant_plan
    RENAME COLUMN stripe_session_id             TO payment_session_id;
ALTER TABLE public.tenant_plan
    RENAME COLUMN stripe_session_url            TO payment_session_url;
ALTER TABLE public.tenant_plan
    RENAME COLUMN stripe_payment_status         TO payment_gateway_status;
ALTER TABLE public.tenant_plan
    RENAME COLUMN stripe_payment_message        TO payment_gateway_message;
ALTER TABLE public.tenant_plan
    RENAME COLUMN stripe_subscription_invoice_id TO subscription_invoice_id;

-- Tambah kolom payment_gateway untuk tahu gateway mana yang dipakai
ALTER TABLE public.tenant_plan
    ADD COLUMN IF NOT EXISTS payment_gateway VARCHAR(50) NOT NULL DEFAULT 'stripe';

-- Update index agar nama sesuai kolom baru
DROP INDEX IF EXISTS idx_tenant_plan_stripe_session;
DROP INDEX IF EXISTS idx_tenant_plan_stripe_sub;
CREATE INDEX IF NOT EXISTS idx_tenant_plan_payment_session ON public.tenant_plan (payment_session_id);
CREATE INDEX IF NOT EXISTS idx_tenant_plan_subscription_invoice ON public.tenant_plan (subscription_invoice_id);

-- ============================================================
-- 2. Per-tenant order_payments (semua schema tenant)
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
            -- Rename kolom jika belum di-rename
            IF EXISTS (
                SELECT 1 FROM information_schema.columns
                WHERE table_schema = schema_record.schema_name
                  AND table_name = 'order_payments'
                  AND column_name = 'stripe_session_id'
            ) THEN
                EXECUTE format(
                    'ALTER TABLE %I.order_payments
                        RENAME COLUMN stripe_session_id TO payment_session_id',
                    schema_record.schema_name
                );
                EXECUTE format(
                    'ALTER TABLE %I.order_payments
                        RENAME COLUMN stripe_session_url TO payment_session_url',
                    schema_record.schema_name
                );
                EXECUTE format(
                    'ALTER TABLE %I.order_payments
                        RENAME COLUMN stripe_payment_status TO payment_gateway_status',
                    schema_record.schema_name
                );
                EXECUTE format(
                    'ALTER TABLE %I.order_payments
                        RENAME COLUMN stripe_invoice_id TO payment_invoice_id',
                    schema_record.schema_name
                );

                -- Tambah kolom payment_gateway
                EXECUTE format(
                    'ALTER TABLE %I.order_payments
                        ADD COLUMN IF NOT EXISTS payment_gateway VARCHAR(50) NOT NULL DEFAULT ''stripe''',
                    schema_record.schema_name
                );

                RAISE NOTICE 'Renamed stripe columns in schema: %', schema_record.schema_name;
            ELSE
                RAISE NOTICE 'Schema % already migrated or no stripe columns found', schema_record.schema_name;
            END IF;
        EXCEPTION WHEN OTHERS THEN
            RAISE NOTICE 'Error processing schema %: %', schema_record.schema_name, SQLERRM;
        END;
    END LOOP;
END $$;
