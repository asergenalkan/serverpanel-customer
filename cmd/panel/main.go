package main

import (
	"log"
	"os"

	"github.com/asergenalkan/serverpanel/internal/api"
	"github.com/asergenalkan/serverpanel/internal/config"
	"github.com/asergenalkan/serverpanel/internal/database"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize database
	db, err := database.Initialize(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("Database initialization failed: %v", err)
	}
	defer db.Close()

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName:               "ServerPanel v1.0.0",
		ServerHeader:          "ServerPanel",
		BodyLimit:             512 * 1024 * 1024, // 512MB upload limit
		StreamRequestBody:     true,
		DisableStartupMessage: false,
	})

	// Middleware
	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${method} ${path} - ${latency}\n",
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
	}))

	// WebSocket route (must be before API routes to avoid JWT middleware)
	app.Use("/api/v1/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})
	app.Get("/api/v1/ws/tasks/:task_id", websocket.New(api.HandleTaskWebSocketDirect(db, cfg)))

	// API routes
	apiRouter := app.Group("/api/v1")
	api.SetupRoutes(apiRouter, db)

	// Serve static files (frontend)
	app.Static("/", "./public")

	// SPA fallback - serve index.html for all non-API routes
	app.Get("/*", func(c *fiber.Ctx) error {
		return c.SendFile("./public/index.html")
	})

	// Get port from config or environment
	port := cfg.Port
	if envPort := os.Getenv("PORT"); envPort != "" {
		port = envPort
	}

	log.Printf("🚀 ServerPanel starting on http://localhost:%s", port)
	log.Printf("� API: http://localhost:%s/api/v1", port)
	log.Printf("� Default login: admin / admin123")

	if err := app.Listen(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
