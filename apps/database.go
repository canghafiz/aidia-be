package apps

import (
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func OpenConnection(username string, password string, host string, port string, name string) *gorm.DB {
	dsn := "host=" + host + " user=" + username + " password=" + password + " dbname=" + name + " port=" + port + " sslmode=disable TimeZone=Asia/Jakarta"

	dialect := postgres.Open(dsn)

	db, err := gorm.Open(dialect, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		panic("Failed to connect to Postgres database: " + err.Error())
	}

	sqlDB, dbErr := db.DB()
	if dbErr != nil {
		panic("Failed to get sql.DB object: " + dbErr.Error())
	}

	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetMaxIdleConns(100)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	sqlDB.SetConnMaxIdleTime(5 * time.Minute)

	return db
}
