-- Add address and postal_code columns to customer table across all tenant schemas
DO $$
DECLARE
    v_schema TEXT;
BEGIN
    FOR v_schema IN
        SELECT DISTINCT tenant_schema
        FROM public.users
        WHERE tenant_schema IS NOT NULL
          AND tenant_schema != ''
    LOOP
        EXECUTE format('
            ALTER TABLE %I.customer
                ADD COLUMN IF NOT EXISTS address     VARCHAR(255) NULL,
                ADD COLUMN IF NOT EXISTS postal_code VARCHAR(20)  NULL;
        ', v_schema);
    END LOOP;
END $$;
