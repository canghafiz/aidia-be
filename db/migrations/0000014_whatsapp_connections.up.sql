-- ============================================================
-- MIGRATION: 0000014_whatsapp_connections (UP)
-- Menyimpan koneksi WhatsApp Business per tenant
-- yang terhubung melalui Meta Embedded Signup
-- ============================================================

CREATE TABLE IF NOT EXISTS public.whatsapp_connections (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID        NOT NULL REFERENCES public.users(user_id) ON DELETE CASCADE,
    tenant_schema   VARCHAR(255) NOT NULL,
    phone_number_id VARCHAR(255) NOT NULL,
    waba_id         VARCHAR(255) NOT NULL,
    access_token    TEXT        NOT NULL,
    phone_number    VARCHAR(50),
    display_name    VARCHAR(255),
    connected_at    TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Satu tenant hanya boleh punya satu koneksi WhatsApp
CREATE UNIQUE INDEX IF NOT EXISTS idx_whatsapp_connections_user_id
    ON public.whatsapp_connections(user_id);

-- Routing webhook global: phone_number_id → tenant_schema
CREATE UNIQUE INDEX IF NOT EXISTS idx_whatsapp_connections_phone_number_id
    ON public.whatsapp_connections(phone_number_id);

CREATE INDEX IF NOT EXISTS idx_whatsapp_connections_tenant_schema
    ON public.whatsapp_connections(tenant_schema);
