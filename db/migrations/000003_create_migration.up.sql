-- Tenant
ALTER TABLE public.tenant
    DROP CONSTRAINT fk_tenant_user,
    ADD CONSTRAINT fk_tenant_user
        FOREIGN KEY (user_id) REFERENCES public.users (user_id)
            ON DELETE CASCADE;

-- Tenant Approval Logs
ALTER TABLE public.tenant_approval_logs
    DROP CONSTRAINT fk_tenant_approval_logs_tenant,
    ADD CONSTRAINT fk_tenant_approval_logs_tenant
        FOREIGN KEY (user_id) REFERENCES public.users (user_id)
            ON DELETE CASCADE;

-- Business Profile
ALTER TABLE public.business_profile
    DROP CONSTRAINT fk_business_profile_tenant,
    ADD CONSTRAINT fk_business_profile_tenant
        FOREIGN KEY (tenant_id) REFERENCES public.tenant (tenant_id)
            ON DELETE CASCADE;

-- Tenant Plan
ALTER TABLE public.tenant_plan
    DROP CONSTRAINT fk_tenant_plan_tenant,
    ADD CONSTRAINT fk_tenant_plan_tenant
        FOREIGN KEY (tenant_id) REFERENCES public.tenant (tenant_id)
            ON DELETE CASCADE;

-- Tenant Usage
ALTER TABLE public.tenant_usage
    DROP CONSTRAINT fk_tenant_usage_tenant,
    ADD CONSTRAINT fk_tenant_usage_tenant
        FOREIGN KEY (tenant_id) REFERENCES public.tenant (tenant_id)
            ON DELETE CASCADE;

-- User Roles
ALTER TABLE public.user_roles
    DROP CONSTRAINT fk_user_roles_user,
    ADD CONSTRAINT fk_user_roles_user
        FOREIGN KEY (user_id) REFERENCES public.users (user_id)
            ON DELETE CASCADE;