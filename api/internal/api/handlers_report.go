package api

import (
	"encoding/json"

	"github.com/cloudreport/api/internal/auth"
	"github.com/cloudreport/api/internal/render"
	"github.com/gofiber/fiber/v2"
)

func ReportHandler(d *Deps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req render.Request
		if err := json.Unmarshal(c.Body(), &req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}
		u := auth.UserFrom(c)
		// Anonymous renders are only allowed for templates marked isPublic.
		if u == nil {
			if req.Template.Shortid == "" && req.Template.Name == "" {
				return c.Status(401).JSON(fiber.Map{"error": "auth required for inline templates"})
			}
		}
		res, err := d.Renderer.Render(c.UserContext(), &req, u)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		if res.ProfileID != "" {
			c.Set("Profile-Id", res.ProfileID)
			c.Set("Profile-Location", "/api/profile/"+res.ProfileID)
		}
		c.Set("Content-Type", res.MimeType)
		c.Set("Content-Disposition", "inline; filename="+res.FileName)
		return c.Send(res.Content)
	}
}

func ComponentRenderHandler(d *Deps) fiber.Handler {
	// Component render is a thin wrapper that always returns HTML.
	return func(c *fiber.Ctx) error {
		var body struct {
			Component struct {
				Shortid string `json:"shortid"`
				Content string `json:"content"`
				Engine  string `json:"engine"`
			} `json:"component"`
			Data json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(c.Body(), &body); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}
		req := &render.Request{}
		req.Template.Content = body.Component.Content
		req.Template.Engine = body.Component.Engine
		req.Template.Recipe = "html"
		req.Data = body.Data
		u := auth.UserFrom(c)
		res, err := d.Renderer.Render(c.UserContext(), req, u)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		c.Set("Content-Type", res.MimeType)
		return c.Send(res.Content)
	}
}
