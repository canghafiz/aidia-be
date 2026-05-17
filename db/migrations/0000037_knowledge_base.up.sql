-- Add knowledge_base table to all existing tenant schemas
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
            CREATE TABLE IF NOT EXISTS %I.knowledge_base (
                id             UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
                file_name      VARCHAR(255) NOT NULL,
                file_url       TEXT         NOT NULL,
                extracted_text TEXT         NOT NULL DEFAULT '''',
                created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
                updated_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
            );
        ', v_schema);

        EXECUTE format('
            CREATE INDEX IF NOT EXISTS idx_knowledge_base_created_at
                ON %I.knowledge_base (created_at);
        ', v_schema);
    END LOOP;
END $$;
