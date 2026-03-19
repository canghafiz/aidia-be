-- ============================================================
-- Drop Tenant Schema (dynamic, replace :schema_name with actual username)
-- ============================================================

-- Drop trigger dan function dulu
DROP TRIGGER IF EXISTS trg_insert_order_payment ON :schema_name.orders;
DROP FUNCTION IF EXISTS :schema_name.fn_insert_order_payment();

-- Drop tables yang punya foreign key dulu
DROP TABLE IF EXISTS :schema_name.order_payments;
DROP TABLE IF EXISTS :schema_name.order_products;
DROP TABLE IF EXISTS :schema_name.order_history;
DROP TABLE IF EXISTS :schema_name.orders;
DROP TABLE IF EXISTS :schema_name.customer;
DROP TABLE IF EXISTS :schema_name.product_category_dto;
DROP TABLE IF EXISTS :schema_name.product_image;
DROP TABLE IF EXISTS :schema_name.product_category;
DROP TABLE IF EXISTS :schema_name.product;
DROP TABLE IF EXISTS :schema_name.guest_message_log;
DROP TABLE IF EXISTS :schema_name.guest_message;
DROP TABLE IF EXISTS :schema_name.guest;
DROP TABLE IF EXISTS :schema_name.setting;

-- Drop custom types
DROP TYPE IF EXISTS :schema_name.order_status;
DROP TYPE IF EXISTS :schema_name.payment_status;

-- Drop schema
DROP SCHEMA IF EXISTS :schema_name CASCADE;