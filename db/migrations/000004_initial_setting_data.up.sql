-- UP
INSERT INTO public.setting (group_name, sub_group_name, name, value) VALUES
-- Notification: Order Status Check Interval
('notification', 'Order Status Check Interval', 'n-minutes', '{n-minutes}'),

-- Notification: Status New / Confirmed Order
('notification', 'Status New / Confirmed Order', 'new-order-wa-number', '{wa-number}'),
('notification', 'Status New / Confirmed Order', 'new-order-email', '{email}'),
('notification', 'Status New / Confirmed Order', 'new-order-tele-id', '{tele-id}'),
('notification', 'Status New / Confirmed Order', 'new-order-notif-n-hours', '{notif-n-hours}'),

-- Notification: Status Cooking
('notification', 'Status Cooking', 'cooking-wa-number', '{wa-number}'),
('notification', 'Status Cooking', 'cooking-email', '{email}'),
('notification', 'Status Cooking', 'cooking-tele-id', '{tele-id}'),
('notification', 'Status Cooking', 'cooking-notif-n-hours', '{notif-n-hours}'),

-- Notification: Status Packing
('notification', 'Status Packing', 'packing-wa-number', '{wa-number}'),
('notification', 'Status Packing', 'packing-email', '{email}'),
('notification', 'Status Packing', 'packing-tele-id', '{tele-id}'),
('notification', 'Status Packing', 'packing-notif-n-hours', '{notif-n-hours}'),

-- Notification: Status Delivery
('notification', 'Status Delivery', 'delivery-wa-number', '{wa-number}'),
('notification', 'Status Delivery', 'delivery-email', '{email}'),
('notification', 'Status Delivery', 'delivery-tele-id', '{tele-id}'),
('notification', 'Status Delivery', 'delivery-notif-n-hours', '{notif-n-hours}'),

-- Integration: Open Ai
('integration', 'Open Ai', 'openai-token', '{openai-token}'),
('integration', 'Open Ai', 'openai-assistant-id', '{openai-assistant-id}'),

-- Integration: Stripe Aidia
('integration', 'Stripe Aidia', 'stripe-aidia-secret-key', 'sk_test_51TA6Wf23GXia0A6PIIoraLAr2qptkqZCSF9g0qP2Kad4fb3NVAwOLqkICOqyXycJXoW2LnMJWEkFhdtCxNU3Gflj00KuDHUZ9N'),
('integration', 'Stripe Aidia', 'stripe-aidia-webhook-secret', 'whsec_YwDQbH4puDsARWHpN8G6KHeuSgD3mzeK'),
('integration', 'Stripe Aidia', 'stripe-aidia-public-key', 'pk_test_51TA6Wf23GXia0A6P8MBZQxYjXMfk8B27glWFbMzyF5jjvIU1iEUGmFPttm5P4Oyc3qgNtl5QvsIK8wRCNdAVZoqu00kYvgMD8M'),

-- Integration: Telegram
('integration', 'Telegram', 'telegram-bot-token', '{telegram-bot-token}'),
('integration', 'Telegram', 'bot-enabled', 'true'),
('integration', 'Telegram', 'operational-hours', '{"monday":{"start":"08:00","end":"22:00"},"tuesday":{"start":"08:00","end":"22:00"},"wednesday":{"start":"08:00","end":"22:00"},"thursday":{"start":"08:00","end":"22:00"},"friday":{"start":"08:00","end":"22:00"},"saturday":{"start":"08:00","end":"22:00"},"sunday":{"start":"08:00","end":"22:00"}}'),
('integration', 'Telegram', 'timezone', 'Asia/Jakarta'),
('integration', 'Telegram', 'manual-mode', 'false'),

-- Integration: Whatsapp
('integration', 'Whatsapp', 'whatsapp-token', '{whatsapp-token}'),
('integration', 'Whatsapp', 'whatsapp-api-version', '{whatsapp-api-version}'),
('integration', 'Whatsapp', 'whatsapp-phone-id', '{whatsapp-phone-id}'),
('integration', 'Whatsapp', 'whatsapp-webhook-token', '{whatsapp-webhook-token}');