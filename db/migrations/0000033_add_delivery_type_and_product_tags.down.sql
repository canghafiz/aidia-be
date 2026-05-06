DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN
        SELECT DISTINCT tenant_schema
        FROM public.users
        WHERE tenant_schema IS NOT NULL AND tenant_schema != ''
    LOOP
        -- Remove product tag tables
        EXECUTE format('DROP TABLE IF EXISTS %I.product_tag_dto', r.tenant_schema);
        EXECUTE format('DROP TABLE IF EXISTS %I.product_tag', r.tenant_schema);

        -- Remove delivery_type and charge settings
        EXECUTE format('
            DELETE FROM %I.setting
            WHERE group_name = ''delivery'' AND name IN (''delivery_type'', ''charge'')',
            r.tenant_schema);

        -- Remove Self Pick Up delivery (all rows for sub_groups where delivery_type was Pickup)
        -- Note: delivery_type rows already removed above; remove any remaining sub_groups named ''Self Pick Up''
        EXECUTE format('
            DELETE FROM %I.setting
            WHERE group_name = ''delivery''
              AND sub_group_name IN (
                  SELECT sub_group_name FROM %I.setting
                  WHERE group_name = ''delivery'' AND name = ''name'' AND value = ''Self Pick Up''
              )',
            r.tenant_schema, r.tenant_schema);
    END LOOP;
END;
$$;
