-- ============================================================
-- MIGRATION: 000002_seed_roles (UP)
-- Seed data: SuperAdmin, Admin, Client
-- ============================================================

INSERT INTO roles (id, name, description, created_at, updated_at)
VALUES
    (gen_random_uuid(), 'SuperAdmin', 'Has full access to all features and settings', NOW(), NOW()),
    (gen_random_uuid(), 'Admin',      'Has access to manage most features except system settings', NOW(), NOW()),
    (gen_random_uuid(), 'Client',     'Has limited access to client-facing features only', NOW(), NOW());