-- ============================================================
-- MIGRATION: 000001_initial_users_tenant (DOWN)
-- ============================================================

-- Schema: tenant_acme
DROP TABLE IF EXISTS tenant_acme.business_profile;
DROP SCHEMA IF EXISTS tenant_acme;

-- Schema: public — junction tables first
DROP TABLE IF EXISTS public.menu_permissions;
DROP TABLE IF EXISTS public.role_permissions;
DROP TABLE IF EXISTS public.user_roles;

-- Schema: public — dependent tables
DROP TABLE IF EXISTS public.tenant_approval_logs;
DROP TABLE IF EXISTS public.tenant;
DROP TABLE IF EXISTS public.menus;
DROP TABLE IF EXISTS public.permissions;
DROP TABLE IF EXISTS public.roles;
DROP TABLE IF EXISTS public.users;