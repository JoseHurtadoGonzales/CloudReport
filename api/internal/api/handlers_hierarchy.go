package api

import (
	"github.com/cloudreport/api/internal/store"
	"github.com/gofiber/fiber/v2"
)

type hierarchyMoveReq struct {
	Source struct {
		EntitySet string `json:"entitySet"`
		Shortid   string `json:"shortid"`
	} `json:"source"`
	Target struct {
		Shortid string `json:"shortid"` // target folder shortid (or "" for root)
	} `json:"target"`
}

// HierarchyMoveHandler relocates an entity to a different folder. Compatible
// with jsreport's /studio/hierarchyMove endpoint.
func HierarchyMoveHandler(d *Deps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req hierarchyMoveReq
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}
		spec, ok := store.EntitySpecs[req.Source.EntitySet]
		if !ok {
			return c.Status(400).JSON(fiber.Map{"error": "unknown entitySet"})
		}
		var folderID any = nil
		if req.Target.Shortid != "" {
			f, err := d.DB.GetGenericByShortid(c.UserContext(), store.EntitySpecs["folders"], req.Target.Shortid)
			if err != nil {
				return c.Status(404).JSON(fiber.Map{"error": "target folder not found"})
			}
			folderID = f["id"]
		}
		updated, err := d.DB.UpdateGeneric(c.UserContext(), spec, req.Source.Shortid, map[string]any{
			"folder_id": folderID,
		})
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(updated)
	}
}

// ValidateEntityNameHandler ensures the requested name is unique within the
// target folder for the given entity set. Used by Studio-style UIs before
// saving.
func ValidateEntityNameHandler(d *Deps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req struct {
			EntitySet string `json:"entitySet"`
			Name      string `json:"name"`
			FolderID  string `json:"folder"` // shortid
			Shortid   string `json:"shortid"` // existing entity, to exclude itself
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}
		spec, ok := store.EntitySpecs[req.EntitySet]
		if !ok {
			return c.Status(400).JSON(fiber.Map{"error": "unknown entitySet"})
		}
		// Check uniqueness inside folder (folder_id IS NULL when at root).
		var folderID any
		if req.FolderID != "" {
			if f, err := d.DB.GetGenericByShortid(c.UserContext(), store.EntitySpecs["folders"], req.FolderID); err == nil {
				folderID = f["id"]
			}
		}
		where := "name = $1 AND folder_id IS NOT DISTINCT FROM $2"
		args := []any{req.Name, folderID}
		if req.Shortid != "" {
			where += " AND shortid <> $3"
			args = append(args, req.Shortid)
		}
		n, err := d.DB.CountGeneric(c.UserContext(), spec, where, args)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"valid": n == 0})
	}
}
