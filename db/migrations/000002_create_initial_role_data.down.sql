-- ============================================================
-- MIGRATION: 000002_seed_roles (DOWN)
-- Remove seeded roles: SuperAdmin, Admin, Client
-- ============================================================

DELETE FROM roles WHERE name IN ('SuperAdmin', 'Admin', 'Client');