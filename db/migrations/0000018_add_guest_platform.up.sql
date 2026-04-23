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
        EXECUTE format('ALTER TABLE %I.guest ADD COLUMN IF NOT EXISTS platform VARCHAR(30);', v_schema);
        EXECUTE format('CREATE INDEX IF NOT EXISTS idx_guest_platform ON %I.guest (platform);', v_schema);

        EXECUTE format($sql$
            UPDATE %I.guest g
            SET platform = sub.platform
            FROM (
                SELECT DISTINCT ON (guest_id) guest_id, platform
                FROM %I.guest_message
                WHERE platform IS NOT NULL AND platform <> ''
                ORDER BY guest_id, created_at DESC
            ) sub
            WHERE g.id = sub.guest_id
              AND (g.platform IS NULL OR g.platform = '');
        $sql$, v_schema, v_schema);

        EXECUTE format($sql$
            UPDATE %I.guest
            SET platform = 'whatsapp'
            WHERE (platform IS NULL OR platform = '')
              AND sosmed ? 'wa_id';
        $sql$, v_schema);

        EXECUTE format($sql$
            UPDATE %I.guest
            SET platform = 'telegram'
            WHERE (platform IS NULL OR platform = '')
              AND sosmed ? 'username';
        $sql$, v_schema);

        EXECUTE format('ALTER TABLE %I.guest DROP COLUMN IF EXISTS platform_username;', v_schema);
    END LOOP;
END $$;
