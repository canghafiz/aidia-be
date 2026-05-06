-- Revert: remove address and postal_code columns from customer table
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
                DROP COLUMN IF EXISTS address,
                DROP COLUMN IF EXISTS postal_code;
        ', v_schema);
    END LOOP;
END $$;
