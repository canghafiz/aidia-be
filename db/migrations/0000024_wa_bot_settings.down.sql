-- Reverse: remove WhatsApp bot settings from public and all tenant schemas
DELETE FROM public.setting
WHERE group_name = 'integration' AND sub_group_name = 'WhatsApp'
  AND name IN ('bot-enabled', 'manual-mode', 'bot-active-hours');
