-- ============================================================
-- MIGRATION: 0000036_remove_hitpay (DOWN)
-- Restore HitPay Aidia settings to public.setting
-- ============================================================

INSERT INTO public.setting (group_name, sub_group_name, name, value)
VALUES
    ('integration', 'HitPay Aidia', 'hitpay-aidia-api-key', ''),
    ('integration', 'HitPay Aidia', 'hitpay-aidia-salt-key', ''),
    ('integration', 'Payment Gateway', 'payment-gateway-active', 'stripe')
ON CONFLICT (sub_group_name, name) DO NOTHING;
