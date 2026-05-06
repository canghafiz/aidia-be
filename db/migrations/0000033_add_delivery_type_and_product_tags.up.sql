DO $$
DECLARE
    r          RECORD;
    pickup_id  UUID;
    has_pickup BOOLEAN;
BEGIN
    FOR r IN
        SELECT DISTINCT tenant_schema
        FROM public.users
        WHERE tenant_schema IS NOT NULL AND tenant_schema != ''
    LOOP
        -- product_tag table
        EXECUTE format('
            CREATE TABLE IF NOT EXISTS %I.product_tag (
                id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
                name       VARCHAR(100) NOT NULL,
                created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
                updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
            )', r.tenant_schema);

        EXECUTE format(
            'CREATE UNIQUE INDEX IF NOT EXISTS idx_product_tag_name ON %I.product_tag (name)',
            r.tenant_schema);

        -- product_tag_dto table
        EXECUTE format('
            CREATE TABLE IF NOT EXISTS %I.product_tag_dto (
                product_id UUID NOT NULL,
                tag_id     UUID NOT NULL,
                CONSTRAINT pk_product_tag_dto PRIMARY KEY (product_id, tag_id),
                CONSTRAINT fk_ptagdto_product FOREIGN KEY (product_id)
                    REFERENCES %I.product (id) ON DELETE CASCADE,
                CONSTRAINT fk_ptagdto_tag FOREIGN KEY (tag_id)
                    REFERENCES %I.product_tag (id) ON DELETE CASCADE
            )',
            r.tenant_schema, r.tenant_schema, r.tenant_schema);

        -- delivery_type: insert for all existing delivery sub_groups that lack it
        EXECUTE format('
            INSERT INTO %I.setting (group_name, sub_group_name, name, value)
            SELECT DISTINCT ''delivery'', sub_group_name, ''delivery_type'', ''Delivery''
            FROM %I.setting s
            WHERE group_name = ''delivery''
              AND NOT EXISTS (
                  SELECT 1 FROM %I.setting
                  WHERE group_name = ''delivery''
                    AND sub_group_name = s.sub_group_name
                    AND name = ''delivery_type''
              )
            ON CONFLICT (sub_group_name, name) DO NOTHING',
            r.tenant_schema, r.tenant_schema, r.tenant_schema);

        -- charge: insert 0.00 for all existing delivery sub_groups that lack it
        EXECUTE format('
            INSERT INTO %I.setting (group_name, sub_group_name, name, value)
            SELECT DISTINCT ''delivery'', sub_group_name, ''charge'', ''0.00''
            FROM %I.setting s
            WHERE group_name = ''delivery''
              AND NOT EXISTS (
                  SELECT 1 FROM %I.setting
                  WHERE group_name = ''delivery''
                    AND sub_group_name = s.sub_group_name
                    AND name = ''charge''
              )
            ON CONFLICT (sub_group_name, name) DO NOTHING',
            r.tenant_schema, r.tenant_schema, r.tenant_schema);

        -- Add default Self Pick Up if no Pickup delivery type exists
        EXECUTE format('
            SELECT EXISTS (
                SELECT 1 FROM %I.setting
                WHERE group_name = ''delivery'' AND name = ''delivery_type'' AND value = ''Pickup''
            )',
            r.tenant_schema) INTO has_pickup;

        IF NOT has_pickup THEN
            pickup_id := gen_random_uuid();
            EXECUTE format('
                INSERT INTO %I.setting (group_name, sub_group_name, name, value) VALUES
                    (''delivery'', $1, ''name'',          ''Self Pick Up''),
                    (''delivery'', $1, ''is_visible'',    ''true''),
                    (''delivery'', $1, ''description'',   ''Customer picks up the order directly.''),
                    (''delivery'', $1, ''delivery_type'', ''Pickup''),
                    (''delivery'', $1, ''charge'',        ''0.00'')
                ON CONFLICT (sub_group_name, name) DO NOTHING',
                r.tenant_schema) USING pickup_id::text;
        END IF;
    END LOOP;
END;
$$;
