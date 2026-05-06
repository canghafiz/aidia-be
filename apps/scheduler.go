package apps

import (
	"fmt"
	"log"

	"backend/helpers"

	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

type Scheduler struct {
	cron *cron.Cron
	db   *gorm.DB
}

func NewScheduler(db *gorm.DB) *Scheduler {
	return &Scheduler{
		cron: cron.New(),
		db:   db,
	}
}

func (s *Scheduler) Start() {
	_, err := s.cron.AddFunc("0 0 * * *", s.expireTenantPlans)
	if err != nil {
		log.Printf("[Scheduler] failed to add expireTenantPlans job: %v", err)
		return
	}

	// Run every minute to catch expired orders quickly
	_, err = s.cron.AddFunc("*/1 * * * *", s.expireOrders)
	if err != nil {
		log.Printf("[Scheduler] failed to add expireOrders job: %v", err)
		return
	}

	// Run every minute to auto-manage bot on/off based on ai-operational-prompt
	_, err = s.cron.AddFunc("*/1 * * * *", s.syncBotActiveHours)
	if err != nil {
		log.Printf("[Scheduler] failed to add syncBotActiveHours job: %v", err)
		return
	}

	s.cron.Start()
	log.Println("[Scheduler] started")
}

func (s *Scheduler) Stop() {
	s.cron.Stop()
	log.Println("[Scheduler] stopped")
}

func (s *Scheduler) expireTenantPlans() {
	log.Println("[Scheduler] running expireTenantPlans")

	result := s.db.Exec("SELECT fn_expire_tenant_plans()")
	if result.Error != nil {
		log.Printf("[Scheduler] expireTenantPlans error: %v", result.Error)
		return
	}

	log.Printf("[Scheduler] expireTenantPlans done, rows affected: %d", result.RowsAffected)
}

func (s *Scheduler) expireOrders() {
	log.Println("[Scheduler] running expireOrders")

	// Get all tenant schemas
	var schemas []string
	err := s.db.Raw(`
		SELECT schema_name
		FROM information_schema.schemata
		WHERE schema_name NOT IN ('public', 'information_schema', 'pg_catalog', 'pg_toast')
		  AND schema_name NOT LIKE 'pg_%'
	`).Scan(&schemas).Error

	if err != nil {
		log.Printf("[Scheduler] expireOrders error getting schemas: %v", err)
		return
	}

	totalExpired := 0
	for _, schema := range schemas {
		var count int
		err := s.db.Raw(`SELECT ` + schema + `.fn_expire_orders()`).Scan(&count).Error
		if err != nil {
			log.Printf("[Scheduler] expireOrders error in schema %s: %v", schema, err)
			continue
		}
		totalExpired += count
		log.Printf("[Scheduler] Expired %d orders in schema %s", count, schema)
	}

	log.Printf("[Scheduler] expireOrders done, total expired: %d", totalExpired)
}

// syncBotActiveHours auto-manages manual-mode and bot-enabled for every tenant
// based on ai-operational-prompt (or store-operational-hours as fallback) per schema.
// Runs every minute. Only touches settings when a schedule is configured.
func (s *Scheduler) syncBotActiveHours() {
	var schemas []string
	err := s.db.Raw(`SELECT DISTINCT tenant_schema FROM public.users WHERE tenant_schema IS NOT NULL AND tenant_schema != ''`).Scan(&schemas).Error
	if err != nil {
		log.Printf("[Scheduler] syncBotActiveHours: error getting schemas: %v", err)
		return
	}

	for _, schema := range schemas {
		s.syncSchemaActiveHours(schema)
	}
}

func (s *Scheduler) syncSchemaActiveHours(schema string) {
	// Primary: ai-operational-prompt (AI Operational — JSON schedule, single source of truth)
	// Fallback: store-operational-hours
	var hoursJSON string
	s.db.Raw(fmt.Sprintf(
		`SELECT value FROM %s.setting WHERE group_name = 'ai_prompt' AND sub_group_name = 'AI Operational' AND name = 'ai-operational-prompt' LIMIT 1`,
		schema,
	)).Scan(&hoursJSON)

	if hoursJSON == "" {
		s.db.Raw(fmt.Sprintf(
			`SELECT value FROM %s.setting WHERE group_name = 'ai_prompt' AND sub_group_name = 'Store Operational' AND name = 'store-operational-hours' LIMIT 1`,
			schema,
		)).Scan(&hoursJSON)
	}

	if hoursJSON == "" {
		return
	}

	var timezone string
	s.db.Raw(fmt.Sprintf(
		`SELECT value FROM %s.setting WHERE group_name = 'integration' AND sub_group_name = 'Telegram' AND name = 'timezone' LIMIT 1`,
		schema,
	)).Scan(&timezone)
	if timezone == "" {
		timezone = "Asia/Singapore"
	}

	withinHours, _ := helpers.IsWithinOperationalHours(hoursJSON, timezone)

	manualMode := "true"
	botEnabled := "false"
	if withinHours {
		manualMode = "false"
		botEnabled = "true"
	}

	for _, platform := range []string{"Telegram", "WhatsApp"} {
		s.db.Exec(fmt.Sprintf(
			`INSERT INTO %s.setting (group_name, sub_group_name, name, value)
			 VALUES ('integration', $1, 'manual-mode', $2)
			 ON CONFLICT (sub_group_name, name) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()`,
			schema,
		), platform, manualMode)

		s.db.Exec(fmt.Sprintf(
			`INSERT INTO %s.setting (group_name, sub_group_name, name, value)
			 VALUES ('integration', $1, 'bot-enabled', $2)
			 ON CONFLICT (sub_group_name, name) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()`,
			schema,
		), platform, botEnabled)
	}
}
