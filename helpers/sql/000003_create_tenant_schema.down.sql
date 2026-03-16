-- ============================================================
-- Drop Tenant Schema (dynamic, replace :schema_name with actual username)
-- ============================================================

DROP TABLE IF EXISTS :schema_name.order_history;
DROP TABLE IF EXISTS :schema_name.order_detail;
DROP TABLE IF EXISTS :schema_name.orders;
DROP TABLE IF EXISTS :schema_name.product_image;
DROP TABLE IF EXISTS :schema_name.product;
DROP TABLE IF EXISTS :schema_name.guest_message_log;
DROP TABLE IF EXISTS :schema_name.guest_message;
DROP TABLE IF EXISTS :schema_name.guest;
DROP TABLE IF EXISTS :schema_name.setting;

DROP SCHEMA IF EXISTS :schema_name CASCADE;