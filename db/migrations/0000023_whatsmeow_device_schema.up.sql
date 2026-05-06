-- MIGRATION: 0000023_whatsmeow_device_schema (UP)
-- Maps tenant schema_name → whatsmeow device JID for session persistence

CREATE TABLE IF NOT EXISTS public.whatsmeow_device_schema (
    schema_name  VARCHAR(255) PRIMARY KEY,
    jid          VARCHAR(255) NOT NULL,
    phone        VARCHAR(50),
    connected_at TIMESTAMPTZ  DEFAULT NOW(),
    created_at   TIMESTAMPTZ  DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  DEFAULT NOW()
);
