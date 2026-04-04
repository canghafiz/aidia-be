-- ============================================================
-- Create Tenant Schema (dynamic, replace :schema_name with actual username)
-- ============================================================

CREATE SCHEMA IF NOT EXISTS :schema_name;

-- ============================================================
-- Guest
-- ============================================================

CREATE TABLE IF NOT EXISTS :schema_name.guest (
    id                    UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id             UUID,
    identity              VARCHAR(100) NOT NULL,
    username              VARCHAR(100),
    phone                 VARCHAR(30),
    name                  VARCHAR(150),
    sosmed                JSONB,
    ai_thread_id          VARCHAR(150),
    is_take_over          BOOLEAN      NOT NULL DEFAULT FALSE,
    is_read               BOOLEAN      NOT NULL DEFAULT FALSE,
    is_active             BOOLEAN      NOT NULL DEFAULT TRUE,
    platform_chat_id      VARCHAR(100),
    platform_username     VARCHAR(100),
    last_message_at       TIMESTAMPTZ,
    conversation_state    JSONB DEFAULT '{}'::jsonb,
    created_at            TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_guest_identity ON :schema_name.guest (identity);
CREATE INDEX idx_guest_active_read ON :schema_name.guest (is_active, is_read);
CREATE INDEX idx_guest_ai_thread_id ON :schema_name.guest (ai_thread_id);
CREATE INDEX idx_guest_created_at ON :schema_name.guest (created_at);
CREATE INDEX idx_guest_platform_chat ON :schema_name.guest (platform_chat_id);
CREATE INDEX idx_guest_last_message ON :schema_name.guest (last_message_at);
CREATE INDEX idx_guest_tenant ON :schema_name.guest (tenant_id);

-- ============================================================
-- Guest Message
-- ============================================================

CREATE TABLE IF NOT EXISTS :schema_name.guest_message (
    id                     UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    guest_id               UUID        NOT NULL,
    role                   VARCHAR(30) NOT NULL,
    type                   VARCHAR(30),
    message                TEXT,
    is_human               BOOLEAN     NOT NULL DEFAULT FALSE,
    is_active              BOOLEAN     NOT NULL DEFAULT TRUE,
    platform_message_id    INTEGER,
    platform               VARCHAR(30) NOT NULL DEFAULT 'telegram',
    session_id             VARCHAR(100),
    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_guest_message_guest
    FOREIGN KEY (guest_id) REFERENCES :schema_name.guest (id)
    ON DELETE RESTRICT
);

CREATE INDEX idx_guest_msg_guest_created ON :schema_name.guest_message (guest_id, created_at);
CREATE INDEX idx_guest_msg_platform_id ON :schema_name.guest_message (platform_message_id);
CREATE INDEX idx_guest_msg_platform ON :schema_name.guest_message (platform);
CREATE INDEX idx_guest_msg_session ON :schema_name.guest_message (session_id);

-- ============================================================
-- Product
-- ============================================================

CREATE TABLE IF NOT EXISTS :schema_name.product (
                                                    id         UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    name       VARCHAR(150)  NOT NULL,
    weight     NUMERIC(15,2) NOT NULL DEFAULT 0,
    price      NUMERIC(15,2) NOT NULL DEFAULT 0,
    original_price NUMERIC(15,2) NOT NULL DEFAULT 0,
    description TEXT,
    delivery_id UUID        NOT NULL,
    is_out_of_stock BOOLEAN NOT NULL DEFAULT FALSE,
    is_active  BOOLEAN       NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ   NOT NULL DEFAULT NOW()
    );

CREATE INDEX idx_product_is_active ON :schema_name.product (is_active);
CREATE INDEX idx_product_name      ON :schema_name.product (name);

-- ============================================================
-- Product Image
-- ============================================================

CREATE TABLE IF NOT EXISTS :schema_name.product_image (
                                                          id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID        NOT NULL,
    image      TEXT        NOT NULL,
    is_active  BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_product_image_product
    FOREIGN KEY (product_id) REFERENCES :schema_name.product (id)
    ON DELETE CASCADE
    );

CREATE INDEX idx_product_image_product_active ON :schema_name.product_image (product_id, is_active);

-- ============================================================
-- Product Category
-- ============================================================

CREATE TABLE IF NOT EXISTS :schema_name.product_category (
                                                             id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(100) NOT NULL,
    is_visible  BOOLEAN      NOT NULL DEFAULT TRUE,
    description VARCHAR(100) DEFAULT NULL,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
    );

CREATE TABLE IF NOT EXISTS :schema_name.product_category_dto (
                                                                 product_id  UUID NOT NULL,
                                                                 category_id UUID NOT NULL,

                                                                 CONSTRAINT pk_product_category_dto
                                                                 PRIMARY KEY (product_id, category_id),

    CONSTRAINT fk_product_category_product
    FOREIGN KEY (product_id) REFERENCES :schema_name.product (id)
    ON DELETE CASCADE,

    CONSTRAINT fk_product_category_category
    FOREIGN KEY (category_id) REFERENCES :schema_name.product_category (id)
    ON DELETE CASCADE
    );

-- ============================================================
-- Customer
-- ============================================================
CREATE TYPE :schema_name.customer_account AS ENUM (
                                                  'Telegram',
                                                  'Whatsapp'
                                              );

CREATE TABLE IF NOT EXISTS :schema_name.customer (
                                                     id                 SERIAL       PRIMARY KEY,
                                                     name               VARCHAR(150) NOT NULL,
    phone_country_code VARCHAR(5)   NOT NULL,
    phone_number       VARCHAR(20)  NOT NULL,
    account_type       :schema_name.customer_account  NOT NULL DEFAULT 'Telegram',
    created_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW()
    );

CREATE INDEX idx_customer_phone ON :schema_name.customer (phone_country_code, phone_number);

-- ============================================================
-- Orders
-- ============================================================

CREATE TYPE :schema_name.order_status AS ENUM (
                                                  'Pending',
                                                  'Confirmed',
                                                  'Completed',
                                                  'Cancelled'
                                              );

CREATE TABLE IF NOT EXISTS :schema_name.orders (
                                                   id                      SERIAL                          PRIMARY KEY,
                                                   customer_id             INTEGER                         NOT NULL,
                                                   total_price             NUMERIC(15,2)                   NOT NULL DEFAULT 0,
    status                  :schema_name.order_status       NOT NULL DEFAULT 'Pending',
    delivery_sub_group_name VARCHAR(100)                    NOT NULL,
    street_address          VARCHAR(100)                    NOT NULL,
    postal_code             VARCHAR(20)                     NOT NULL,
    created_at              TIMESTAMPTZ                     NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ                     NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_orders_customer
    FOREIGN KEY (customer_id) REFERENCES :schema_name.customer (id)
    ON DELETE RESTRICT
    );

CREATE INDEX idx_orders_customer_id    ON :schema_name.orders (customer_id);
CREATE INDEX idx_orders_status_created ON :schema_name.orders (status, created_at);
CREATE INDEX idx_orders_created_at     ON :schema_name.orders (created_at);

-- ============================================================
-- Order Product
-- ============================================================

CREATE TABLE IF NOT EXISTS :schema_name.order_products (
                                                           id          SERIAL        PRIMARY KEY,
                                                           order_id    INTEGER       NOT NULL,
                                                           product_id  UUID          NOT NULL,
                                                           quantity    INTEGER       NOT NULL DEFAULT 1,
                                                           total_price NUMERIC(15,2) NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_order_products_order
    FOREIGN KEY (order_id) REFERENCES :schema_name.orders (id)
    ON DELETE RESTRICT,

    CONSTRAINT fk_order_products_product
    FOREIGN KEY (product_id) REFERENCES :schema_name.product (id)
    ON DELETE RESTRICT
    );

CREATE INDEX idx_order_products_order_id   ON :schema_name.order_products (order_id);
CREATE INDEX idx_order_products_product_id ON :schema_name.order_products (product_id);

-- ============================================================
-- Order Payment
-- ============================================================

CREATE TYPE :schema_name.payment_status AS ENUM (
                                                    'Unpaid',
                                                    'Confirming_Payment',
                                                    'Paid',
                                                    'Refunded',
                                                    'Voided'
                                                );

CREATE TABLE IF NOT EXISTS :schema_name.order_payments (
                                                           id             UUID                        PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id       INTEGER                     NOT NULL,
    payment_status :schema_name.payment_status NOT NULL DEFAULT 'Unpaid',
    payment_method VARCHAR(50)                 NOT NULL DEFAULT 'stripe',
    total_price    NUMERIC(15,2)               NOT NULL DEFAULT 0,
    expire_at      TIMESTAMPTZ                 NOT NULL DEFAULT NOW() + INTERVAL '15 minutes',
    stripe_session_id VARCHAR(255),
    stripe_session_url TEXT,
    stripe_payment_status VARCHAR(50),
    stripe_invoice_id VARCHAR(255),
    paid_at        TIMESTAMPTZ,
    is_paid        BOOLEAN                     NOT NULL DEFAULT FALSE,
    created_at     TIMESTAMPTZ                 NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ                 NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_order_payments_order
    FOREIGN KEY (order_id) REFERENCES :schema_name.orders (id)
    ON DELETE RESTRICT
    );

CREATE INDEX idx_order_payments_order_id ON :schema_name.order_payments (order_id);
CREATE INDEX idx_order_payments_status   ON :schema_name.order_payments (payment_status);

CREATE OR REPLACE FUNCTION :schema_name.fn_insert_order_payment()
    RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO :schema_name.order_payments (order_id, payment_status, payment_method, total_price, expire_at)
    VALUES (NEW.id, 'Unpaid':::schema_name.payment_status, 'stripe', NEW.total_price, NEW.created_at + INTERVAL '15 minutes');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_insert_order_payment
    AFTER INSERT ON :schema_name.orders
    FOR EACH ROW
EXECUTE FUNCTION :schema_name.fn_insert_order_payment();

-- ============================================================
-- Setting
-- ============================================================

CREATE TABLE IF NOT EXISTS :schema_name.setting (
                                              id             UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
                                              group_name     VARCHAR(100) NOT NULL,
                                              sub_group_name VARCHAR(100) NOT NULL,
                                              name           VARCHAR(100) NOT NULL,
                                              value          TEXT,
                                              created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
                                              updated_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
                                              CONSTRAINT uq_setting_subgroup_name UNIQUE (sub_group_name, name)
);

-- Seed from public.setting as template
INSERT INTO :schema_name.setting (group_name, sub_group_name, name, value)
SELECT group_name, sub_group_name, name, value
FROM public.setting
WHERE group_name = 'notification'
   OR (group_name = 'integration' AND sub_group_name = 'Telegram')
ON CONFLICT (sub_group_name, name) DO NOTHING;

INSERT INTO :schema_name.setting (group_name, sub_group_name, name, value) VALUES
    ('integration', 'Stripe Client', 'stripe-client-secret-key', '{stripe-client-secret-key}'),
    ('integration', 'Stripe Client', 'stripe-client-public-key', '{stripe-client-public-key}'),
    ('integration', 'Stripe Client', 'stripe-client-webhook-secret', '{stripe-client-webhook-secret}');

-- AI Prompt settings (per section)
INSERT INTO :schema_name.setting (group_name, sub_group_name, name, value) VALUES
    ('ai_prompt', 'AI Product',     'ai-product-prompt',     'Explain our products clearly when customers ask. Mention name, price, and description. If a product is out of stock, inform the customer and suggest alternatives if available.'),
    ('ai_prompt', 'AI Delivery',    'ai-delivery-prompt',    'We offer delivery to the zones listed. If the customer''s area is not listed, kindly inform them we do not cover that area yet.'),
    ('ai_prompt', 'AI Operational', 'ai-operational-prompt', 'We are open Monday to Saturday, 08:00 - 21:00. We are closed on Sundays and national holidays.'),
    ('ai_prompt', 'AI About Store', 'ai-about-store-prompt', 'We are a local store. We are here to help you find the right products and place your order. Feel free to ask anything about our store.'),
    ('ai_prompt', 'AI FAQ',         'ai-faq-prompt',         'Q: How do I place an order? A: Just tell me you want to order and I will guide you. Q: Can I cancel? A: Contact us before the order is processed. Q: How long is delivery? A: Estimated 30-60 minutes.')
ON CONFLICT (sub_group_name, name) DO NOTHING;

-- ============================================================
-- Kitchen Order
-- ============================================================

CREATE TABLE IF NOT EXISTS :schema_name.kitchen_order (
                                                          id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id   INTEGER     NOT NULL UNIQUE,
    status     VARCHAR(50) NOT NULL DEFAULT 'new_order',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_kitchen_order
    FOREIGN KEY (order_id) REFERENCES :schema_name.orders (id)
    ON DELETE CASCADE
    );

CREATE INDEX idx_kitchen_order_status ON :schema_name.kitchen_order (status);
CREATE INDEX idx_kitchen_order_created_at ON :schema_name.kitchen_order (created_at);

-- ============================================================
-- Trigger: insert kitchen_order saat payment Paid + order Confirmed
-- ============================================================

CREATE OR REPLACE FUNCTION :schema_name.fn_insert_kitchen_order()
    RETURNS TRIGGER AS $$
DECLARE
    v_order_status VARCHAR(50);
BEGIN
    IF NEW.payment_status = 'Paid' AND OLD.payment_status != 'Paid' THEN
        SELECT status INTO v_order_status
        FROM :schema_name.orders
        WHERE id = NEW.order_id;

        IF v_order_status = 'Confirmed' THEN
            INSERT INTO :schema_name.kitchen_order (order_id, status)
            VALUES (NEW.order_id, 'new_order')
            ON CONFLICT (order_id) DO NOTHING;
        END IF;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_insert_kitchen_order
    AFTER UPDATE ON :schema_name.order_payments
    FOR EACH ROW
EXECUTE FUNCTION :schema_name.fn_insert_kitchen_order();

CREATE OR REPLACE FUNCTION :schema_name.fn_insert_kitchen_order_on_confirmed()
    RETURNS TRIGGER AS $$
DECLARE
    v_payment_status VARCHAR(50);
BEGIN
    IF NEW.status = 'Confirmed' AND OLD.status != 'Confirmed' THEN
        SELECT payment_status INTO v_payment_status
        FROM :schema_name.order_payments
        WHERE order_id = NEW.id;

        IF v_payment_status = 'Paid' THEN
            INSERT INTO :schema_name.kitchen_order (order_id, status)
            VALUES (NEW.id, 'new_order')
            ON CONFLICT (order_id) DO NOTHING;
        END IF;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_insert_kitchen_order_on_confirmed
    AFTER UPDATE ON :schema_name.orders
    FOR EACH ROW
EXECUTE FUNCTION :schema_name.fn_insert_kitchen_order_on_confirmed();

-- ============================================================
-- Function: Expire Orders (run by scheduler every 1 minute)
-- ============================================================

CREATE OR REPLACE FUNCTION :schema_name.fn_expire_orders()
RETURNS INTEGER AS $$
DECLARE
    expired_count INTEGER;
BEGIN
    -- Update expired order_payments
    UPDATE :schema_name.order_payments
    SET payment_status = 'Voided'
    WHERE payment_status = 'Unpaid'
      AND expire_at < NOW();
    
    GET DIAGNOSTICS expired_count = ROW_COUNT;
    
    -- Update corresponding orders to Cancelled
    UPDATE :schema_name.orders
    SET status = 'Cancelled'
    WHERE id IN (
        SELECT order_id 
        FROM :schema_name.order_payments 
        WHERE payment_status = 'Voided'
    )
    AND status = 'Pending';
    
    RAISE NOTICE 'Expired % unpaid orders', expired_count;
    
    RETURN expired_count;
END;
$$ LANGUAGE plpgsql;