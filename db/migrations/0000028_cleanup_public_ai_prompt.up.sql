-- Remove ai_prompt settings from public.setting — these are per-tenant settings
-- that were incorrectly inserted into public by migration 0000026.
-- Each client schema has its own copy (seeded by migration 0000027).

DELETE FROM public.setting WHERE group_name = 'ai_prompt';
