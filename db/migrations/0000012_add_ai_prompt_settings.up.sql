-- Insert AI prompt settings into every existing tenant schema
DO $$
DECLARE
    v_schema TEXT;
BEGIN
    FOR v_schema IN
        SELECT DISTINCT tenant_schema
        FROM public.users
        WHERE tenant_schema IS NOT NULL
          AND tenant_schema != ''
    LOOP
        EXECUTE format('
            INSERT INTO %I.setting (group_name, sub_group_name, name, value) VALUES
                (''ai_prompt'', ''AI Product'',     ''ai-product-prompt'',     ''Explain our products clearly when customers ask. Mention name, price, and description. If a product is out of stock, inform the customer and suggest alternatives if available.''),
                (''ai_prompt'', ''AI Delivery'',    ''ai-delivery-prompt'',    ''We offer delivery to the zones listed. If the customer''''s area is not listed, kindly inform them we do not cover that area yet.''),
                (''ai_prompt'', ''AI Operational'', ''ai-operational-prompt'', ''We are open Monday to Saturday, 08:00 - 21:00. We are closed on Sundays and national holidays.''),
                (''ai_prompt'', ''AI About Store'', ''ai-about-store-prompt'', ''We are a local store. We are here to help you find the right products and place your order. Feel free to ask anything about our store.''),
                (''ai_prompt'', ''AI FAQ'',         ''ai-faq-prompt'',         ''Q: How do I place an order? A: Just tell me you want to order and I will guide you. Q: Can I cancel? A: Contact us before the order is processed. Q: How long is delivery? A: Estimated 30-60 minutes.'')
            ON CONFLICT (sub_group_name, name) DO NOTHING
        ', v_schema);
    END LOOP;
END $$;
