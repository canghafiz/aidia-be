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
	// Jalan setiap hari jam 00:00
	_, err := s.cron.AddFunc("0 0 * * *", s.expireTenantPlans)
	if err != nil {
		log.Printf("[Scheduler] failed to add expireTenantPlans job: %v", err)
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
