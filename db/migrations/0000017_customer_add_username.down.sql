-- Revert: remove username column and restore NOT NULL on phone fields
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
            DROP INDEX IF EXISTS %I.idx_customer_username;
            ALTER TABLE %I.customer
                DROP COLUMN IF EXISTS username,
                ALTER COLUMN phone_country_code SET NOT NULL,
                ALTER COLUMN phone_number SET NOT NULL;
        ', v_schema, v_schema, v_schema);
    END LOOP;
END $$;
