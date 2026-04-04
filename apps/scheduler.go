package apps

import (
	"log"

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
