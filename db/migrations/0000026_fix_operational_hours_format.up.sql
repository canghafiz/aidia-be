-- Convert ai-operational-prompt from old plain-text to JSON format in all schemas.
-- Also inserts the row if it does not exist yet (new tenants).

DO $$
DECLARE
    v_schema TEXT;
    default_json TEXT := '{"monday":{"start":"09:00","end":"21:00","closed":false},"tuesday":{"start":"09:00","end":"21:00","closed":false},"wednesday":{"start":"09:00","end":"21:00","closed":false},"thursday":{"start":"09:00","end":"21:00","closed":false},"friday":{"start":"09:00","end":"21:00","closed":false},"saturday":{"start":"09:00","end":"21:00","closed":false},"sunday":{"start":"09:00","end":"21:00","closed":true}}';
BEGIN
    FOR v_schema IN
        SELECT DISTINCT tenant_schema FROM public.users
        WHERE tenant_schema IS NOT NULL AND tenant_schema != ''
    LOOP
        -- Insert if missing
        EXECUTE format(
            'INSERT INTO %I.setting (group_name, sub_group_name, name, value)
             VALUES (''ai_prompt'', ''AI Operational'', ''ai-operational-prompt'', %L)
             ON CONFLICT (sub_group_name, name) DO NOTHING',
            v_schema, default_json
        );

        -- Update rows that are not valid JSON (old plain-text format)
        EXECUTE format(
            'UPDATE %I.setting
             SET value = %L
             WHERE sub_group_name = ''AI Operational''
               AND name = ''ai-operational-prompt''
               AND (value = '''' OR (value NOT LIKE ''{%%'' AND value NOT LIKE ''[%%''))',
            v_schema, default_json
        );
    END LOOP;
END;
$$;
