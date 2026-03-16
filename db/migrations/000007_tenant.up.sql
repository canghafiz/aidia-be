-- ============================================================
-- MIGRATION: 000003_tenant_usage_plan_id_and_scheduler (UP)
-- ============================================================

-- ============================================================
-- 1. Drop unique constraint lama (tenant_id, period)
--    karena sekarang 1 tenant bisa punya multiple rows per period
--    (free row + paid plan rows)
-- ============================================================

ALTER TABLE public.tenant_usage
    DROP CONSTRAINT IF EXISTS uq_tenant_usage_tenant_period;

-- ============================================================
-- 2. Tambah kolom tenant_plan_id ke tenant_usage
-- ============================================================

ALTER TABLE public.tenant_usage
    ADD COLUMN IF NOT EXISTS tenant_plan_id UUID DEFAULT NULL;

ALTER TABLE public.tenant_usage
    ADD CONSTRAINT fk_tenant_usage_tenant_plan
        FOREIGN KEY (tenant_plan_id) REFERENCES public.tenant_plan (id)
            ON DELETE SET NULL;

-- Unique constraint baru: 1 row per kombinasi tenant_id + period + tenant_plan_id
-- tenant_plan_id NULL  = free usage row
-- tenant_plan_id = ID  = paid plan usage row
ALTER TABLE public.tenant_usage
    ADD CONSTRAINT uq_tenant_usage_tenant_period_plan
        UNIQUE NULLS NOT DISTINCT (tenant_id, period, tenant_plan_id);

CREATE INDEX IF NOT EXISTS idx_tenant_usage_tenant_plan_id ON public.tenant_usage (tenant_plan_id);

-- ============================================================
-- 3. Update trigger fn_insert_tenant_usage
--    Sekarang insert dengan tenant_plan_id = NULL (free row)
-- ============================================================

CREATE OR REPLACE FUNCTION fn_insert_tenant_usage()
    RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO public.tenant_usage (tenant_id, period, total_tokens, total_cost, tenant_plan_id)
    VALUES (NEW.tenant_id, (DATE_TRUNC('month', NOW()) + INTERVAL '100 years')::DATE, 1000000, 0, NULL)
    ON CONFLICT (tenant_id, period, tenant_plan_id) DO NOTHING;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ============================================================
-- 4. Function expire tenant_plan
--    Update plan_status = 'Expired' untuk plan yang sudah lewat expired_date
-- ============================================================

CREATE OR REPLACE FUNCTION fn_expire_tenant_plans()
    RETURNS void AS $$
BEGIN
    UPDATE public.tenant_plan
    SET
        plan_status = 'Expired',
        updated_at  = NOW()
    WHERE
        plan_status  = 'Active'
      AND is_paid  = TRUE
      AND expired_date < NOW()::DATE;
END;
$$ LANGUAGE plpgsql;