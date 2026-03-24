-- ============================================================
-- MIGRATION: 000001_initial_users_tenant (UP)
-- Schema public  : users, roles, permissions, menus, tenant,
--                  user_roles, role_permissions, menu_permissions,
--                  tenant_approval_logs, business_profile, settings, plan, tenant_plan, tenant_usage
-- ============================================================

-- ============================================================
-- SCHEMA: public
-- INDEPENDENT TABLES (no foreign keys)
-- ============================================================

CREATE TABLE IF NOT EXISTS public.users (
                                            user_id      UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
                                            username     VARCHAR(150) NOT NULL,
                                            name         VARCHAR(150) NOT NULL,
                                            email        VARCHAR(150) NOT NULL,
                                            password     TEXT         NOT NULL,
                                            gender       VARCHAR(20),
                                            tenant_schema VARCHAR(100),
                                            is_active    BOOLEAN,
                                            created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
                                            updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

                                            CONSTRAINT uq_users_username UNIQUE (username),
                                            CONSTRAINT uq_users_email    UNIQUE (email)
);

CREATE INDEX idx_users_email     ON public.users (email);
CREATE INDEX idx_users_username  ON public.users (username);
CREATE INDEX idx_users_is_active ON public.users (is_active);

-- ----------------------------------------------------------------

CREATE TABLE IF NOT EXISTS public.roles (
                                            id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
                                            name        VARCHAR(50) NOT NULL,
                                            description TEXT,
                                            created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                                            updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

                                            CONSTRAINT uq_roles_name UNIQUE (name)
);

-- ============================================================
-- SCHEMA: public
-- TENANT TABLES
-- ============================================================

CREATE TABLE IF NOT EXISTS public.tenant (
                                             tenant_id  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
                                             user_id    UUID        NOT NULL,
                                             role       VARCHAR(50) NOT NULL DEFAULT 'owner',
                                             is_active  BOOLEAN     NOT NULL DEFAULT TRUE,
                                             created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                                             updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

                                             CONSTRAINT uq_tenant_user_id UNIQUE (user_id),
                                             CONSTRAINT fk_tenant_user
                                                 FOREIGN KEY (user_id) REFERENCES public.users (user_id)
                                                     ON DELETE RESTRICT
);

CREATE INDEX idx_tenant_user_id   ON public.tenant (user_id);
CREATE INDEX idx_tenant_is_active ON public.tenant (is_active);

-- ----------------------------------------------------------------

CREATE TABLE IF NOT EXISTS public.tenant_approval_logs (
                                                           id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
                                                           user_id  UUID        NOT NULL,
                                                           action     VARCHAR(20) DEFAULT NULL,
                                                           action_by  UUID        DEFAULT NULL,
                                                           created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                                                           updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

                                                           CONSTRAINT fk_tenant_approval_logs_tenant
                                                               FOREIGN KEY (user_id) REFERENCES public.users (user_id)
                                                                   ON DELETE RESTRICT,
                                                           CONSTRAINT fk_tenant_approval_logs_action_by
                                                               FOREIGN KEY (action_by) REFERENCES public.users (user_id)
                                                                   ON DELETE RESTRICT
);

CREATE INDEX idx_tenant_approval_tenant_id  ON public.tenant_approval_logs (user_id);
CREATE INDEX idx_tenant_approval_action_by  ON public.tenant_approval_logs (action_by);
CREATE INDEX idx_tenant_approval_created_at ON public.tenant_approval_logs (created_at);

-- ============================================================
-- SCHEMA: public
-- JUNCTION TABLES
-- ============================================================

CREATE TABLE IF NOT EXISTS public.user_roles (
                                                 user_id UUID NOT NULL,
                                                 role_id UUID NOT NULL,

                                                 CONSTRAINT pk_user_roles PRIMARY KEY (user_id, role_id),
                                                 CONSTRAINT fk_user_roles_user
                                                     FOREIGN KEY (user_id) REFERENCES public.users (user_id)
                                                         ON DELETE CASCADE,
                                                 CONSTRAINT fk_user_roles_role
                                                     FOREIGN KEY (role_id) REFERENCES public.roles (id)
                                                         ON DELETE CASCADE
);

CREATE INDEX idx_user_roles_role_id ON public.user_roles (role_id);

-- ============================================================
-- Business Profile
-- ============================================================

CREATE TABLE IF NOT EXISTS public.business_profile (
                                                            id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
                                                            tenant_id     UUID         NOT NULL,
                                                            business_name VARCHAR(150),
                                                            address       VARCHAR(255),
                                                            phone         VARCHAR(20),
                                                            created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
                                                            updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

                                                            CONSTRAINT fk_business_profile_tenant
                                                                FOREIGN KEY (tenant_id) REFERENCES public.tenant (tenant_id)
                                                                    ON DELETE RESTRICT
);

CREATE INDEX idx_business_profile_tenant_id ON public.business_profile (tenant_id);

-- ============================================================
-- Setting
-- ============================================================

CREATE TABLE IF NOT EXISTS public.setting (
                                              id             UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
                                              group_name     VARCHAR(100) NOT NULL,
                                              sub_group_name VARCHAR(100) NOT NULL,
                                              name           VARCHAR(100) NOT NULL UNIQUE,
                                              value          TEXT,
                                              created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
                                              updated_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_setting_name ON public.setting (name);

-- ============================================================
-- Plan
-- ============================================================

CREATE TABLE IF NOT EXISTS public.plan (
                                           id                UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
                                           name              VARCHAR(100)  NOT NULL,
                                           is_month          BOOLEAN       NOT NULL DEFAULT TRUE,
                                           duration          INTEGER       NOT NULL,
                                           price             NUMERIC(15,2) NOT NULL DEFAULT 0,
                                           is_active         BOOLEAN       NOT NULL DEFAULT TRUE,
                                           created_at        TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
                                           updated_at        TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_plan_active_show ON public.plan (is_active);

-- ============================================================
-- Tenant Plan
-- ============================================================

CREATE TABLE IF NOT EXISTS public.tenant_plan (
                                                  id                             UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
                                                  tenant_id                      UUID          NOT NULL,
                                                  plan_id                        UUID          NOT NULL,
                                                  invoice_number                 VARCHAR(30)   NOT NULL UNIQUE,
                                                  duration                       INTEGER       NOT NULL,
                                                  is_month                       BOOLEAN       NOT NULL,
                                                  price                          NUMERIC(15,2) NOT NULL DEFAULT 0,
                                                  payment_due_date               DATE,
                                                  paid_at                        TIMESTAMPTZ,
                                                  start_date                     DATE,
                                                  expired_date                   DATE,
                                                  plan_status                    VARCHAR(30)   NOT NULL DEFAULT 'Inactive',
                                                  stripe_session_id              VARCHAR(150),
                                                  stripe_session_url             TEXT,
                                                  stripe_payment_status          VARCHAR(50),
                                                  stripe_payment_message         TEXT,
                                                  stripe_subscription_id         VARCHAR(150),
                                                  stripe_subscription_invoice_id VARCHAR(150),
                                                  stripe_subscription_status     VARCHAR(50),
                                                  is_paid                        BOOLEAN       NOT NULL DEFAULT FALSE,
                                                  is_active                      BOOLEAN       NOT NULL DEFAULT TRUE,
                                                  created_at                     TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
                                                  updated_at                     TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

                                                  CONSTRAINT fk_tenant_plan_tenant
                                                      FOREIGN KEY (tenant_id) REFERENCES public.tenant (tenant_id)
                                                          ON DELETE RESTRICT,

                                                  CONSTRAINT fk_tenant_plan_plan
                                                      FOREIGN KEY (plan_id) REFERENCES public.plan (id)
                                                          ON DELETE RESTRICT
);

CREATE INDEX idx_tenant_plan_tenant_id      ON public.tenant_plan (tenant_id);
CREATE INDEX idx_tenant_plan_active_expiry  ON public.tenant_plan (tenant_id, is_active, expired_date);
CREATE INDEX idx_tenant_plan_stripe_session ON public.tenant_plan (stripe_session_id);
CREATE INDEX idx_tenant_plan_stripe_sub     ON public.tenant_plan (stripe_subscription_id);

-- ============================================================
-- Tenant Usage
-- ============================================================

CREATE TABLE IF NOT EXISTS public.tenant_usage (
                                                   id           UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
                                                   tenant_id    UUID          NOT NULL,
                                                   period       DATE          NOT NULL,
                                                   total_tokens BIGINT        DEFAULT 0,
                                                   total_cost   NUMERIC(12,4) DEFAULT 0,
                                                   created_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
                                                   updated_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

                                                   CONSTRAINT uq_tenant_usage_tenant_period
                                                       UNIQUE (tenant_id, period),

                                                   CONSTRAINT fk_tenant_usage_tenant
                                                       FOREIGN KEY (tenant_id) REFERENCES public.tenant (tenant_id)
                                                           ON DELETE RESTRICT
);

CREATE INDEX idx_tenant_usage_period ON public.tenant_usage (period);

-- ============================================================
-- Trigger: insert tenant_usage saat tenant baru dibuat
-- ============================================================

CREATE OR REPLACE FUNCTION fn_insert_tenant_usage()
    RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO public.tenant_usage (tenant_id, period, total_tokens, total_cost)
    VALUES (NEW.tenant_id, (DATE_TRUNC('month', NOW()) + INTERVAL '100 years')::DATE, 1000000, 0)
    ON CONFLICT (tenant_id, period) DO NOTHING;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_insert_tenant_usage
    AFTER INSERT ON public.tenant
    FOR EACH ROW
EXECUTE FUNCTION fn_insert_tenant_usage();

-- ============================================================
-- Trigger: insert tenant_usage saat tenant_plan status menjadi Active
-- ============================================================

CREATE OR REPLACE FUNCTION fn_insert_tenant_usage_on_paid()
    RETURNS TRIGGER AS $$
BEGIN
    -- Hanya jalan ketika plan_status berubah menjadi 'Active'
    IF NEW.plan_status = 'Active' AND OLD.plan_status != 'Active' THEN
        INSERT INTO public.tenant_usage (tenant_id, tenant_plan_id, period, total_tokens, total_cost)
        VALUES (NEW.tenant_id, NEW.id, NEW.start_date, -1, 0)
        ON CONFLICT (tenant_id, period, tenant_plan_id) DO NOTHING;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_insert_tenant_usage_on_paid
    AFTER UPDATE ON public.tenant_plan
    FOR EACH ROW
EXECUTE FUNCTION fn_insert_tenant_usage_on_paid();

-- ============================================================
-- Function: expire orders yang payment-nya sudah lewat expire_at
-- ============================================================

CREATE OR REPLACE FUNCTION fn_expire_orders()
    RETURNS void AS $$
DECLARE
    v_schema TEXT;
BEGIN
    FOR v_schema IN
        SELECT u.username
        FROM public.users u
                 JOIN public.user_roles ur ON ur.user_id = u.user_id
                 JOIN public.roles r ON r.id = ur.role_id
        WHERE r.name = 'Client'
          AND u.tenant_schema IS NOT NULL
        LOOP
            EXECUTE format('
            UPDATE %I.order_payments
            SET payment_status = ''Voided'', updated_at = NOW()
            WHERE payment_status = ''Unpaid''
            AND expire_at < NOW()
        ', v_schema);

            EXECUTE format('
            UPDATE %I.orders
            SET status = ''Cancelled'', updated_at = NOW()
            WHERE status = ''Pending''
            AND id IN (
                SELECT order_id FROM %I.order_payments
                WHERE payment_status = ''Voided''
            )
        ', v_schema, v_schema);
        END LOOP;
END;
$$ LANGUAGE plpgsql;