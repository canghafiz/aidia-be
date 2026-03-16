-- Drop constraint lama
ALTER TABLE public.tenant_approval_logs
    DROP CONSTRAINT fk_tenant_approval_logs_tenant,
    DROP CONSTRAINT fk_tenant_approval_logs_action_by;

-- Tambah constraint baru dengan CASCADE
ALTER TABLE public.tenant_approval_logs
    ADD CONSTRAINT fk_tenant_approval_logs_tenant
        FOREIGN KEY (user_id) REFERENCES public.users (user_id)
            ON DELETE CASCADE,
    ADD CONSTRAINT fk_tenant_approval_logs_action_by
        FOREIGN KEY (action_by) REFERENCES public.users (user_id)
            ON DELETE CASCADE;