package api

import (
	"github.com/cloudreport/api/internal/auth"
	"github.com/gofiber/fiber/v2"
)

const Version = "1.0.0"

func PingHandler() fiber.Handler {
	return func(c *fiber.Ctx) error { return c.SendString("pong") }
}

func VersionHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"version": Version, "product": "cloud-report"})
	}
}

func SettingsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		u := auth.UserFrom(c)
		resp := fiber.Map{
			"version":      Version,
			"product":      "cloud-report",
			"tenant":       nil,
			"isTenantAdmin": false,
		}
		if u != nil {
			resp["username"] = u.Username
			resp["isAdmin"] = u.IsAdmin
		}
		return c.JSON(resp)
	}
}

func RecipesHandler(d *Deps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.JSON(d.Renderer.ListRecipes())
	}
}

func EnginesHandler(d *Deps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.JSON(d.Renderer.ListEngines())
	}
}

func ExtensionsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Static list mirroring jsreport's extensions shape.
		return c.JSON([]fiber.Map{
			{"name": "core", "version": Version, "main": "true"},
			{"name": "express", "version": Version},
			{"name": "auth", "version": Version},
			{"name": "handlebars", "version": Version},
			{"name": "weasyprint", "version": Version},
			{"name": "docx", "version": Version},
			{"name": "pptx", "version": Version},
			{"name": "xlsx", "version": Version},
			{"name": "pdf-utils", "version": Version},
			{"name": "scheduling", "version": Version},
			{"name": "scripts", "version": Version},
			{"name": "version-control", "version": Version},
		})
	}
}
