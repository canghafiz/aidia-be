-- ============================================================
-- Create Tenant Schema (dynamic, replace :schema_name with actual username)
-- ============================================================

CREATE SCHEMA IF NOT EXISTS :schema_name;

-- ============================================================
-- Guest
-- ============================================================

CREATE TABLE IF NOT EXISTS :schema_name.guest (
                                                  id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    identity     VARCHAR(100) NOT NULL,
    username     VARCHAR(100),
    phone        VARCHAR(30),
    name         VARCHAR(150),
    sosmed       JSONB,
    ai_thread_id VARCHAR(150),
    is_take_over BOOLEAN      NOT NULL DEFAULT FALSE,
    is_read      BOOLEAN      NOT NULL DEFAULT FALSE,
    is_active    BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
    );

CREATE INDEX idx_guest_identity      ON :schema_name.guest (identity);
CREATE INDEX idx_guest_active_read   ON :schema_name.guest (is_active, is_read);
CREATE INDEX idx_guest_ai_thread_id  ON :schema_name.guest (ai_thread_id);
CREATE INDEX idx_guest_created_at    ON :schema_name.guest (created_at);

-- ============================================================
-- Guest Message
-- ============================================================

CREATE TABLE IF NOT EXISTS :schema_name.guest_message (
                                                          id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    guest_id          UUID        NOT NULL,
    role              VARCHAR(30) NOT NULL,
    type              VARCHAR(30),
    message           TEXT,
    ai_run_id         VARCHAR(150),
    ai_run_status     VARCHAR(50),
    ai_run_last_error TEXT,
    is_human          BOOLEAN     NOT NULL DEFAULT FALSE,
    token_usage       INTEGER     DEFAULT 0,
    is_active         BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_guest_message_guest
    FOREIGN KEY (guest_id) REFERENCES :schema_name.guest (id)
    ON DELETE RESTRICT
    );

CREATE INDEX idx_guest_msg_guest_created ON :schema_name.guest_message (guest_id, created_at);
CREATE INDEX idx_guest_msg_ai_run_id     ON :schema_name.guest_message (ai_run_id);
CREATE INDEX idx_guest_msg_ai_run_status ON :schema_name.guest_message (ai_run_status);
CREATE INDEX idx_guest_msg_is_active     ON :schema_name.guest_message (is_active);

-- ============================================================
-- Guest Message Log
-- ============================================================

CREATE TABLE IF NOT EXISTS :schema_name.guest_message_log (
                                                              id                   UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    guest_id             UUID        NOT NULL,
    info                 VARCHAR(255),
    log                  TEXT,
    system_error_message TEXT,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_guest_message_log_guest
    FOREIGN KEY (guest_id) REFERENCES :schema_name.guest (id)
    ON DELETE RESTRICT
    );

CREATE INDEX idx_guest_msg_log_guest_created ON :schema_name.guest_message_log (guest_id, created_at);

-- ============================================================
-- Product
-- ============================================================

CREATE TABLE IF NOT EXISTS :schema_name.product (
                                                    id         UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    name       VARCHAR(150)  NOT NULL,
    code       VARCHAR(30) NOT NULL,
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
    expire_at      TIMESTAMPTZ                 NOT NULL DEFAULT NOW() + INTERVAL '24 hours',
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
    INSERT INTO :schema_name.order_payments (order_id, payment_status, payment_method, total_price)
    VALUES (NEW.id, 'Unpaid':::schema_name.payment_status, 'stripe', NEW.total_price);
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
                                              name           VARCHAR(100) NOT NULL UNIQUE,
                                              value          TEXT,
                                              created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
                                              updated_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Seed from public.setting as template
INSERT INTO :schema_name.setting (group_name, sub_group_name, name, value)
SELECT group_name, sub_group_name, name, value
FROM public.setting
WHERE group_name = 'notification'
   OR (group_name = 'integration' AND sub_group_name = 'Telegram')
    ON CONFLICT (name) DO NOTHING;

INSERT INTO :schema_name.setting (group_name, sub_group_name, name, value) VALUES
    ('integration', 'Stripe Client', 'stripe-client-secret-key', '{stripe-client-secret-key}'),
    ('integration', 'Stripe Client', 'stripe-client-webhook-secret', '{stripe-client-webhook-secret}');

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