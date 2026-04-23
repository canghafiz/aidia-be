-- Add username column and make phone fields nullable for Telegram customers
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
                ADD COLUMN IF NOT EXISTS username VARCHAR(100) NULL,
                ALTER COLUMN phone_country_code DROP NOT NULL,
                ALTER COLUMN phone_number DROP NOT NULL;
        ', v_schema);

        EXECUTE format('
            DO $inner$
            BEGIN
                IF NOT EXISTS (
                    SELECT 1 FROM pg_indexes
                    WHERE schemaname = %L AND indexname = ''idx_customer_username''
                ) THEN
                    CREATE INDEX idx_customer_username ON %I.customer (username);
                END IF;
            END $inner$;
        ', v_schema, v_schema);
    END LOOP;
END $$;
