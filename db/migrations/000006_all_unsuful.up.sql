ALTER TABLE public.tenant_plan
    DROP COLUMN stripe_subscription_id,
    DROP COLUMN stripe_subscription_status;