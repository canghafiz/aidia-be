-- Migration 0000031: Remove bot-active-hours from all tenant schemas
-- ai-operational-prompt (ai_prompt/AI Operational) is now the single source of truth
-- for bot scheduling. The scheduler and bot controllers read from there directly.
DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN
        SELECT DISTINCT tenant_schema
        FROM public.users
        WHERE tenant_schema IS NOT NULL AND tenant_schema != ''
    LOOP
        EXECUTE format(
            'DELETE FROM %I.setting WHERE group_name = ''integration'' AND name = ''bot-active-hours''',
            r.tenant_schema
        );
    END LOOP;
END;
$$;
