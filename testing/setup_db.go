package testing

import (
	"backend/apps"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/joho/godotenv"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func init() {
	_, filename, _, _ := runtime.Caller(0)
	rootDir := filepath.Join(filepath.Dir(filename), "..")

	err := godotenv.Load(filepath.Join(rootDir, ".env"))
	if err != nil {
		panic("failed to load .env: " + err.Error())
	}
}

func SetUpDbTest(t *testing.T) *gorm.DB {
	t.Helper()

	db := apps.OpenConnection(
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASS"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_TEST_NAME"),
	)

	return db.Session(&gorm.Session{
		Logger: logger.Default.LogMode(logger.Silent),
	})
}
