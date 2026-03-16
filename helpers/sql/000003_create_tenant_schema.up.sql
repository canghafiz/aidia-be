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
-- Orders
-- ============================================================

CREATE TABLE IF NOT EXISTS :schema_name.orders (
                                                   id                     UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    code                   VARCHAR(30)   NOT NULL UNIQUE,
    guest_id               UUID          NOT NULL,
    order_type             VARCHAR(50),
    packing_type           VARCHAR(50),
    address                TEXT,
    postcode               VARCHAR(10),
    delivery_charge        NUMERIC(15,2) DEFAULT 0,
    delivery_date          DATE,
    delivery_time          VARCHAR(20),
    delivery_instructions  TEXT,
    status_id              UUID,
    stripe_session_id      VARCHAR(150),
    stripe_session_url     TEXT,
    stripe_payment_status  VARCHAR(50),
    stripe_payment_message TEXT,
    is_paid                BOOLEAN       NOT NULL DEFAULT FALSE,
    is_read                BOOLEAN       NOT NULL DEFAULT FALSE,
    is_active              BOOLEAN       NOT NULL DEFAULT TRUE,
    created_at             TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_orders_guest
    FOREIGN KEY (guest_id) REFERENCES :schema_name.guest (id)
    ON DELETE RESTRICT
    );

CREATE INDEX idx_orders_guest_id       ON :schema_name.orders (guest_id);
CREATE INDEX idx_orders_status_created ON :schema_name.orders (status_id, created_at);
CREATE INDEX idx_orders_paid_active    ON :schema_name.orders (is_paid, is_active);
CREATE INDEX idx_orders_read_active    ON :schema_name.orders (is_read, is_active);
CREATE INDEX idx_orders_stripe_session ON :schema_name.orders (stripe_session_id);
CREATE INDEX idx_orders_delivery_date  ON :schema_name.orders (delivery_date);
CREATE INDEX idx_orders_created_at     ON :schema_name.orders (created_at);

-- ============================================================
-- Order Detail
-- ============================================================

CREATE TABLE IF NOT EXISTS :schema_name.order_detail (
                                                         id           UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id     UUID          NOT NULL,
    product_id   UUID,
    product_name VARCHAR(150)  NOT NULL,
    qty          INTEGER       NOT NULL DEFAULT 1,
    price        NUMERIC(15,2) NOT NULL DEFAULT 0,
    total_price  NUMERIC(15,2) NOT NULL DEFAULT 0,
    is_active    BOOLEAN       NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_order_detail_order
    FOREIGN KEY (order_id) REFERENCES :schema_name.orders (id)
    ON DELETE RESTRICT,

    CONSTRAINT fk_order_detail_product
    FOREIGN KEY (product_id) REFERENCES :schema_name.product (id)
    ON DELETE SET NULL
    );

CREATE INDEX idx_order_detail_order_id   ON :schema_name.order_detail (order_id);
CREATE INDEX idx_order_detail_product_id ON :schema_name.order_detail (product_id);

-- ============================================================
-- Order History
-- ============================================================

CREATE TABLE IF NOT EXISTS :schema_name.order_history (
                                                          id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id   UUID        NOT NULL,
    status_id  UUID        NOT NULL,
    is_active  BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_order_history_order
    FOREIGN KEY (order_id) REFERENCES :schema_name.orders (id)
    ON DELETE RESTRICT
    );

CREATE INDEX idx_order_history_order_created ON :schema_name.order_history (order_id, created_at);

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
