package main

import (
	"github.com/Nysonn/campuscare/internal/chatbot"
	"github.com/Nysonn/campuscare/internal/config"
	"github.com/Nysonn/campuscare/internal/db"
	"github.com/Nysonn/campuscare/internal/handlers"
	"github.com/Nysonn/campuscare/internal/mail"
	"github.com/Nysonn/campuscare/internal/middleware"
	"github.com/Nysonn/campuscare/internal/services"
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

	r.POST("/register", authHandler.Register)
	r.POST("/login", authHandler.Login)
	r.POST("/logout", authHandler.Logout)

	campaignHandler := &handlers.CampaignHandler{DB: database}

	auth := r.Group("/")
	auth.Use(middleware.AuthRequired(sessionService))

	auth.POST("/campaigns", middleware.RequireRole(database, "student"), campaignHandler.Create)
	auth.PUT("/campaigns/:id", middleware.RequireRole(database, "student"), campaignHandler.Update)
	auth.DELETE("/campaigns/:id", middleware.RequireRole(database, "student"), campaignHandler.Delete)

	auth.PUT("/admin/campaigns/:id", middleware.RequireRole(database, "admin"), campaignHandler.Approve)
	auth.GET("/admin/campaigns", middleware.RequireRole(database, "admin"), campaignHandler.ListPending)

	r.GET("/campaigns", campaignHandler.PublicList)

	mailer := mail.NewMailer()
	contributionHandler := &handlers.ContributionHandler{
		DB:     database,
		Mailer: mailer,
	}

	r.POST("/contributions", contributionHandler.Create)
	r.POST("/contributions/:id/simulate", contributionHandler.Simulate)

	chatbotHandler := &chatbot.ChatbotHandler{
		Service: &chatbot.Service{DB: database},
	}

	auth.POST("/chatbot",
		middleware.RequireRole(database, "student"),
		chatbotHandler.Ask,
	)

	bookingHandler := &handlers.BookingHandler{DB: database}

	auth.POST("/bookings", middleware.RequireRole(database, "student"), bookingHandler.Create)
	auth.PUT("/bookings/:id/status", middleware.RequireRole(database, "counselor"), bookingHandler.UpdateStatus)

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

	admin.DELETE("/users/:id", adminHandler.AnonymizeUser)

	admin.GET("/crisis-flags", adminHandler.ListCrisisFlags)

	admin.GET("/audit", adminHandler.AuditLogs)

	r.Run(":" + cfg.AppPort)
}
