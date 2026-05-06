-- Restore operational-hours setting to public (tenant schemas not restored — re-run tenant seed if needed)
INSERT INTO public.setting (group_name, sub_group_name, name, value)
VALUES (
    'integration', 'Telegram', 'operational-hours',
    '{"monday":{"start":"08:00","end":"22:00"},"tuesday":{"start":"08:00","end":"22:00"},"wednesday":{"start":"08:00","end":"22:00"},"thursday":{"start":"08:00","end":"22:00"},"friday":{"start":"08:00","end":"22:00"},"saturday":{"start":"08:00","end":"22:00"},"sunday":{"start":"08:00","end":"22:00"}}'
)
ON CONFLICT (sub_group_name, name) DO NOTHING;
