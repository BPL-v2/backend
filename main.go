package main

import (
	"bpl/config"
	"bpl/controller"
	_ "bpl/docs"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	ginprometheus "github.com/zsais/go-gin-prometheus"
)

// @title           BPL Backend API
// @version         2.0
// @description     This is the backend API for the BPL project.

// @contact.name   	Liberator
// @contact.email 	Liberatorist@gmail.com

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {

	// Load and validate configuration
	cfg := config.Env()
	db, err := config.InitDB(
		cfg.DatabaseHost,
		cfg.DatabasePort,
		cfg.PostgresUser,
		cfg.PostgresPassword,
		cfg.DatabaseName,
	)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	_ = db
	// autoMigrate(db)
	r := gin.New()
	r.Use(gin.Recovery())
	err = r.SetTrustedProxies(nil)
	if err != nil {
		log.Fatalf("Failed to set trusted proxies: %v", err)
	}
	addLogger(r)
	addMetrics(r)
	addDocs(r)
	setCors(r)
	controller.SetRoutes(r)
	err = r.Run(":8000")
	if err != nil {
		fmt.Println("Failed to start server:", err)
	}
}

func addLogger(r *gin.Engine) {
	r.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		SkipPaths: []string{"/api/metrics"},
		Skip: func(c *gin.Context) bool {
			return c.Request.URL.Query().Get("token") != ""
		},
	}))

}

func addMetrics(r *gin.Engine) {
	p := ginprometheus.NewPrometheus("gin")
	re := regexp.MustCompile(`\d+`)
	charRe := regexp.MustCompile(`characters/[^/]+(/|$)`)
	p.ReqCntURLLabelMappingFn = func(c *gin.Context) string {
		url := strings.Split(c.Request.URL.String(), "?")[0]
		url = strings.ReplaceAll(url, "current", "?")
		url = re.ReplaceAllString(url, "?")
		url = charRe.ReplaceAllString(url, "characters/?$1")
		return strings.TrimPrefix(url, "/api")
	}
	p.MetricsPath = "/api/metrics"
	p.Use(r)
}

func addDocs(r *gin.Engine) {
	r.GET("/api/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))
}

func setCors(r *gin.Engine) {
	corsConfigGetOptions := cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
	corsConfigOtherMethods := cors.Config{
		AllowOriginFunc:  config.IsApprovedOrigin,
		AllowMethods:     []string{"POST", "PUT", "PATCH", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}

	r.Use(func(c *gin.Context) {
		if c.Request.Method == "OPTIONS" {
			// Check the Access-Control-Request-Method header to determine the actual method being preflighted
			requestedMethod := c.GetHeader("Access-Control-Request-Method")
			if requestedMethod == "GET" || requestedMethod == "OPTIONS" {
				cors.New(corsConfigGetOptions)(c)
			} else {
				cors.New(corsConfigOtherMethods)(c)
			}
			c.AbortWithStatus(204) // Respond with 204 No Content for preflight
			return
		}

		if c.Request.Method == "GET" {
			cors.New(corsConfigGetOptions)(c)
		} else {
			cors.New(corsConfigOtherMethods)(c)
		}
	})
}

// func autoMigrate(db *gorm.DB) {
// 	err := db.AutoMigrate(
// 		&repository.Objective{},
// 		&repository.Condition{},
// 		&repository.Event{},
// 		&repository.Team{},
// 		&repository.User{},
// 		&repository.TeamUser{},
// 		&repository.StashChange{},
// 		&repository.ObjectiveMatch{},
// 		&repository.Submission{},
// 		&repository.ClientCredentials{},
// 		&repository.Signup{},
// 		&repository.Oauth{},
// 		&repository.KafkaConsumer{},
// 		&repository.RecurringJob{},
// 		&repository.LadderEntry{},
// 		&repository.ChangeId{},
// 		&repository.GuildStashTab{},
// 	)
// 	if err != nil {
// 		panic(err)
// 	}
// }
