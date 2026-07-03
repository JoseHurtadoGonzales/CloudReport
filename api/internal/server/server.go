package server

import (
	"time"

	"github.com/cloudreport/api/internal/config"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func New(cfg *config.Config) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:   "cloud-report",
		BodyLimit: 50 * 1024 * 1024, // 50MB like jsreport
		// fasthttp's default header buffer is 4KB. This app shares the host IP
		// with n8n (:5678) and the SeaweedFS UIs, and browser cookies are scoped
		// by host (not port), so their cookies pile up on the same domain and the
		// combined Cookie header can blow past 4KB → HTTP 431. Give headers room.
		ReadBufferSize:        32 * 1024,
		ReadTimeout:           60 * time.Second,
		WriteTimeout:          120 * time.Second,
		DisableStartupMessage: false,
	})

	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} ${method} ${path} ${latency}\n",
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "*",
		AllowMethods:     "GET,POST,PUT,DELETE,PATCH,OPTIONS,MERGE",
		AllowHeaders:     "Origin,Content-Type,Accept,Authorization,x-api-key,Profile-Id",
		ExposeHeaders:    "Profile-Id,Profile-Location,Content-Disposition,Content-Type",
		AllowCredentials: false,
	}))

	return app
}
