package main

import (
	"backend/apps"
	"backend/dependencies"
	"backend/helpers"
	"os"
	"time"

	_ "backend/docs"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
)

// @title           AI-Dia API
// @version         1.0
// @description     REST API for AI-Dia application
// @host            data.ai-dia.com
// @BasePath        /api/v1
// @securityDefinitions.apiKey BearerAuth
// @in              header
// @name            Authorization
// @description     Format: "Bearer {token}" — paste token dari login response, accessToken
func main() {
	// Load .env file
	err := godotenv.Load(".env")

	if os.Getenv("APP_STATUS") == "Debug" {
		gin.SetMode(gin.DebugMode)
	}

	if os.Getenv("APP_STATUS") == "Production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Env
	port := os.Getenv("APP_PORT")
	jwtKey := os.Getenv("JWT_KEY")

	// Database Config
	dbPort := os.Getenv("DB_PORT")
	dbHost := os.Getenv("DB_HOST")
	dbUser := os.Getenv("DB_USER")
	dbPass := os.Getenv("DB_PASS")
	dbName := os.Getenv("DB_NAME")
	db := apps.OpenConnection(dbUser, dbPass, dbHost, dbPort, dbName)

	// Other
	validate := validator.New()

	// Dependency
	dependency := dependencies.NewDependency(db, validate, jwtKey)

	// Scheduler
	s := apps.NewScheduler(db)
	s.Start()
	defer s.Stop()

	// Setup Router
	engine := gin.Default()
	engine.Use(cors.New(cors.Config{
		AllowOrigins:  []string{"*"},
		AllowMethods:  []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:  []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders: []string{"Content-Length"},
		MaxAge:        24 * time.Hour,
	}))
	routerParent := apps.Router{
		Dependency: dependency,
		Engine:     engine,
	}
	router := apps.NewRouter(routerParent)

	// Run Server
	if port == "" {
		port = ":3001"
	} else if port[0] != ':' {
		port = ":" + port
	}
	err = router.Engine.Run(port)
	helpers.FatalError(err)
}
