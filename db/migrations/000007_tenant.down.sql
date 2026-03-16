-- ============================================================
-- MIGRATION: 000003_tenant_usage_plan_id_and_scheduler (DOWN)
-- ============================================================

DROP FUNCTION IF EXISTS fn_expire_tenant_plans();

ALTER TABLE public.tenant_usage
    DROP CONSTRAINT IF EXISTS uq_tenant_usage_tenant_period_plan,
    DROP CONSTRAINT IF EXISTS fk_tenant_usage_tenant_plan,
    DROP COLUMN IF EXISTS tenant_plan_id;

ALTER TABLE public.tenant_usage
    ADD CONSTRAINT uq_tenant_usage_tenant_period UNIQUE (tenant_id, period);