package main

import (
	"github.com/Nysonn/campuscare/internal/chatbot"
	"github.com/Nysonn/campuscare/internal/config"
	"github.com/Nysonn/campuscare/internal/db"
	"github.com/Nysonn/campuscare/internal/handlers"
	"github.com/Nysonn/campuscare/internal/mail"
	"github.com/Nysonn/campuscare/internal/middleware"
	"github.com/Nysonn/campuscare/internal/services"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {

	cfg := config.Load()
	database := db.New(cfg.DatabaseURL)

	sessionService := &services.SessionService{
		DB:         database,
		SessionTTL: cfg.SessionTTL,
	}

	authHandler := &handlers.AuthHandler{
		DB:             database,
		SessionService: sessionService,
	}

	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "https://campuscareug.web.app", "https://campuscareug.firebaseapp.com", "http://192.168.11.23:5173", "http://172.23.0.1:5173", "http://172.19.0.1:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	r.GET("/health", func(c *gin.Context) {
		if err := database.Ping(c.Request.Context()); err != nil {
			c.JSON(503, gin.H{"status": "db_unavailable"})
			return
		}
		c.JSON(200, gin.H{"status": "ok"})
	})

	r.POST("/register", authHandler.Register)
	r.POST("/login", authHandler.Login)
	r.POST("/logout", authHandler.Logout)

	campaignHandler := &handlers.CampaignHandler{DB: database}

	auth := r.Group("/")
	auth.Use(middleware.AuthRequired(sessionService))

	auth.POST("/campaigns", middleware.RequireRole(database, "student"), campaignHandler.Create)
	auth.PUT("/campaigns/:id", middleware.RequireRole(database, "student"), campaignHandler.Update)
	auth.DELETE("/campaigns/:id", middleware.RequireRole(database, "student"), campaignHandler.Delete)
	auth.GET("/campaigns/mine", middleware.RequireRole(database, "student"), campaignHandler.MyCampaigns)

	auth.PUT("/admin/campaigns/:id", middleware.RequireRole(database, "admin"), campaignHandler.Approve)
	auth.GET("/admin/campaigns", middleware.RequireRole(database, "admin"), campaignHandler.ListPending)

	r.GET("/campaigns", campaignHandler.PublicList)

	mailer := mail.NewMailer()
	contributionHandler := &handlers.ContributionHandler{
		DB:     database,
		Mailer: mailer,
	}

	r.POST("/contributions", contributionHandler.Create)

	chatbotHandler := &chatbot.ChatbotHandler{
		Service: &chatbot.Service{DB: database},
	}

	auth.POST("/chatbot",
		middleware.RequireRole(database, "student"),
		chatbotHandler.Ask,
	)
	auth.GET("/chatbot/history",
		middleware.RequireRole(database, "student"),
		chatbotHandler.History,
	)

	bookingHandler := &handlers.BookingHandler{DB: database, Mailer: mailer}

	auth.POST("/bookings", middleware.RequireRole(database, "student"), bookingHandler.Create)
	auth.PUT("/bookings/:id/status", middleware.RequireRole(database, "counselor"), bookingHandler.UpdateStatus)
	auth.GET("/bookings/mine", middleware.RequireRole(database, "student"), bookingHandler.MyBookings)
	auth.GET("/bookings/counselor", middleware.RequireRole(database, "counselor"), bookingHandler.CounselorBookings)
	auth.GET("/counselors", middleware.RequireRole(database, "student"), bookingHandler.ListCounselors)

	auth.GET("/profile", authHandler.Profile)
	auth.PATCH("/profile", authHandler.UpdateProfile)

	admin := r.Group("/admin")
	admin.Use(middleware.AuthRequired(sessionService))
	admin.Use(middleware.RequireRole(database, "admin"))

	adminHandler := &handlers.AdminHandler{DB: database}

	admin.GET("/dashboard", adminHandler.Dashboard)

	admin.GET("/users", adminHandler.ListUsers)
	admin.PUT("/users/:id/status", adminHandler.UpdateUserStatus)

	admin.DELETE("/campaigns/:id", adminHandler.DeleteCampaign)

	admin.GET("/bookings", adminHandler.ListBookings)

	admin.GET("/contributions", adminHandler.ListContributions)
	admin.GET("/contributions/export", adminHandler.ExportContributions)

	r.Run(":" + cfg.AppPort)
}
