-- ============================================================
-- MIGRATION: 0000014_whatsapp_connections (DOWN)
-- ============================================================

DROP INDEX IF EXISTS public.idx_whatsapp_connections_tenant_schema;
DROP INDEX IF EXISTS public.idx_whatsapp_connections_phone_number_id;
DROP INDEX IF EXISTS public.idx_whatsapp_connections_user_id;
DROP TABLE IF EXISTS public.whatsapp_connections;
