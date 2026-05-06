-- Gate kitchen_order triggers with kds_enabled setting across all existing tenant schemas
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
        -- Recreate fn_insert_kitchen_order with KDS gate
        EXECUTE format($fn$
            CREATE OR REPLACE FUNCTION %I.fn_insert_kitchen_order()
                RETURNS TRIGGER AS $body$
            DECLARE
                v_order_status VARCHAR(50);
                v_kds_enabled  TEXT;
            BEGIN
                SELECT value INTO v_kds_enabled
                FROM %I.setting
                WHERE group_name = 'features' AND sub_group_name = 'KDS' AND name = 'kds_enabled';

                IF COALESCE(v_kds_enabled, 'false') != 'true' THEN
                    RETURN NEW;
                END IF;

                IF NEW.payment_status = 'Paid' AND OLD.payment_status != 'Paid' THEN
                    SELECT status INTO v_order_status
                    FROM %I.orders
                    WHERE id = NEW.order_id;

                    IF v_order_status = 'Confirmed' THEN
                        INSERT INTO %I.kitchen_order (order_id, status)
                        VALUES (NEW.order_id, 'new_order')
                        ON CONFLICT (order_id) DO NOTHING;
                    END IF;
                END IF;

                RETURN NEW;
            END;
            $body$ LANGUAGE plpgsql;
        $fn$, v_schema, v_schema, v_schema, v_schema);

        -- Recreate fn_insert_kitchen_order_on_confirmed with KDS gate
        EXECUTE format($fn$
            CREATE OR REPLACE FUNCTION %I.fn_insert_kitchen_order_on_confirmed()
                RETURNS TRIGGER AS $body$
            DECLARE
                v_payment_status VARCHAR(50);
                v_kds_enabled    TEXT;
            BEGIN
                SELECT value INTO v_kds_enabled
                FROM %I.setting
                WHERE group_name = 'features' AND sub_group_name = 'KDS' AND name = 'kds_enabled';

                IF COALESCE(v_kds_enabled, 'false') != 'true' THEN
                    RETURN NEW;
                END IF;

                IF NEW.status = 'Confirmed' AND OLD.status != 'Confirmed' THEN
                    SELECT payment_status INTO v_payment_status
                    FROM %I.order_payments
                    WHERE order_id = NEW.id;

                    IF v_payment_status = 'Paid' THEN
                        INSERT INTO %I.kitchen_order (order_id, status)
                        VALUES (NEW.id, 'new_order')
                        ON CONFLICT (order_id) DO NOTHING;
                    END IF;
                END IF;

                RETURN NEW;
            END;
            $body$ LANGUAGE plpgsql;
        $fn$, v_schema, v_schema, v_schema, v_schema);

    END LOOP;
END $$;
