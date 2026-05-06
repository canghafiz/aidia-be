-- Remove legacy operational-hours setting from public and all tenant schemas.
-- Operational hours are now managed via ai-operational-prompt (JSON) in the AI Agent settings.

-- 1. Remove from public settings
DELETE FROM public.setting
WHERE group_name = 'integration'
  AND sub_group_name = 'Telegram'
  AND name = 'operational-hours';

-- 2. Remove from all existing tenant schemas
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
        EXECUTE format(
            'DELETE FROM %I.setting WHERE group_name = $1 AND sub_group_name = $2 AND name = $3',
            v_schema
        ) USING 'integration', 'Telegram', 'operational-hours';
    END LOOP;
END;
$$;
