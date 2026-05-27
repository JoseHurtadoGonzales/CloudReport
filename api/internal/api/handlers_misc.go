package api

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cloudreport/api/internal/auth"
	"github.com/cloudreport/api/internal/render"
	"github.com/cloudreport/api/internal/scheduler"
	"github.com/cloudreport/api/internal/store"
	"github.com/gofiber/fiber/v2"
)

// ----- Schema -----
func SchemaHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		set := c.Params("set")
		spec, ok := store.EntitySpecs[set]
		if !ok {
			return c.Status(404).JSON(fiber.Map{"error": "unknown entity set"})
		}
		props := fiber.Map{}
		for _, col := range spec.Columns {
			props[col] = fiber.Map{"type": "string"}
		}
		return c.JSON(fiber.Map{
			"$schema":    "http://json-schema.org/draft-07/schema#",
			"type":       "object",
			"properties": props,
		})
	}
}

// ----- Profile -----

func ProfileGetHandler(d *Deps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		spec := store.EntitySpecs["profiles"]
		p, err := d.DB.GetGenericByShortid(c.UserContext(), spec, c.Params("id"))
		if err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "not found"})
		}
		return c.JSON(p)
	}
}

func ProfileEventsHandler(d *Deps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		spec := store.EntitySpecs["profiles"]
		p, err := d.DB.GetGenericByShortid(c.UserContext(), spec, c.Params("id"))
		if err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "not found"})
		}
		key, _ := p["blob_key"].(string)
		if key == "" {
			return c.Status(204).Send(nil)
		}
		data, _, err := d.Blob.Get(c.UserContext(), key)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		c.Set("Content-Type", "text/plain")
		return c.Send(data)
	}
}

// ----- Reports content (already rendered) -----

func ReportContentHandler(d *Deps, publicOnly bool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		spec := store.EntitySpecs["reports"]
		r, err := d.DB.GetGenericByShortid(c.UserContext(), spec, c.Params("shortid"))
		if err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "not found"})
		}
		if publicOnly {
			if v, ok := r["is_public"].(bool); !ok || !v {
				return c.Status(403).JSON(fiber.Map{"error": "not public"})
			}
		}
		key, _ := r["blob_key"].(string)
		if key == "" {
			return c.Status(404).JSON(fiber.Map{"error": "no content"})
		}
		data, ct, err := d.Blob.Get(c.UserContext(), key)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		if mime, ok := r["mime_type"].(string); ok && mime != "" {
			ct = mime
		}
		c.Set("Content-Type", ct)
		return c.Send(data)
	}
}

func ReportStatusHandler(d *Deps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		spec := store.EntitySpecs["reports"]
		r, err := d.DB.GetGenericByShortid(c.UserContext(), spec, c.Params("shortid"))
		if err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "not found"})
		}
		return c.JSON(fiber.Map{
			"state": r["state"],
			"error": r["error"],
		})
	}
}

// ----- Assets content -----

func AssetContentHandler(d *Deps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		spec := store.EntitySpecs["assets"]
		a, err := d.DB.GetGenericByShortid(c.UserContext(), spec, c.Params("shortid"))
		if err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "not found"})
		}
		if inline, ok := a["inline_content"].(string); ok && inline != "" {
			c.Set("Content-Type", coalesceString(a["mime_type"], "text/plain"))
			return c.SendString(inline)
		}
		key, _ := a["blob_key"].(string)
		if key == "" {
			return c.Status(404).JSON(fiber.Map{"error": "no content"})
		}
		data, ct, err := d.Blob.Get(c.UserContext(), key)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		if mime, ok := a["mime_type"].(string); ok && mime != "" {
			ct = mime
		}
		c.Set("Content-Type", ct)
		return c.Send(data)
	}
}

func coalesceString(v any, def string) string {
	if s, ok := v.(string); ok && s != "" {
		return s
	}
	return def
}

// ----- Scheduling -----

func NextRunHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		expr := c.Params("cron")
		t, err := scheduler.NextRun(expr)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"nextRun": t.Format(time.RFC3339)})
	}
}

type runNowReq struct {
	ScheduleShortid string `json:"scheduleShortid"`
}

func RunNowHandler(d *Deps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req runNowReq
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}
		spec := store.EntitySpecs["schedules"]
		s, err := d.DB.GetGenericByShortid(c.UserContext(), spec, req.ScheduleShortid)
		if err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "schedule not found"})
		}
		tpl, _ := s["template_shortid"].(string)
		if tpl == "" {
			return c.Status(400).JSON(fiber.Map{"error": "schedule has no template"})
		}
		go func() {
			ctx := c.UserContext()
			req := &render.Request{}
			req.Template.Shortid = tpl
			req.Data = json.RawMessage("{}")
			_, _ = d.Renderer.Render(ctx, req, auth.UserFrom(c))
		}()
		return c.JSON(fiber.Map{"status": "queued"})
	}
}

// ----- Version control (snapshot the whole DB into a ZIP) -----

type vcsCommitReq struct {
	Message string `json:"message"`
}

func VCSCommitHandler(d *Deps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req vcsCommitReq
		_ = c.BodyParser(&req)
		if req.Message == "" {
			req.Message = "snapshot"
		}
		zipBytes, err := snapshotZip(c, d)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		key, err := d.Blob.Put(c.UserContext(), "versions", "", zipBytes, "application/zip")
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		spec := store.EntitySpecs["versions"]
		u := auth.UserFrom(c)
		payload := map[string]any{
			"message":   req.Message,
			"blob_key":  key,
		}
		if u != nil {
			payload["author_id"] = u.ID
		}
		out, err := d.DB.InsertGeneric(c.UserContext(), spec, payload)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(201).JSON(out)
	}
}

func VCSHistoryHandler(d *Deps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		spec := store.EntitySpecs["versions"]
		items, err := d.DB.ListGeneric(c.UserContext(), spec, "", nil, "created_at DESC", 100, 0)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(items)
	}
}

func VCSLocalChangesHandler(d *Deps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Without a working copy concept, "local changes" is just the count of
		// templates updated since the latest version. Simplified.
		return c.JSON(fiber.Map{"changes": []any{}})
	}
}

// VCSRevertHandler is the public entry point; it dispatches to the proper
// implementation in handlers_extra.go.
func VCSRevertHandler(d *Deps) fiber.Handler {
	return VCSRevertImplHandler(d)
}

// ----- Import / Export -----

func ExportHandler(d *Deps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		data, err := snapshotZip(c, d)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		c.Set("Content-Type", "application/zip")
		c.Set("Content-Disposition", "attachment;filename=export.zip")
		return c.Send(data)
	}
}

func ImportHandler(d *Deps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		body := c.Body()
		zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}
		stats := fiber.Map{}
		for _, f := range zr.File {
			rc, err := f.Open()
			if err != nil {
				continue
			}
			content, _ := readAll(rc)
			rc.Close()

			// Path convention: <entitySet>/<shortid>.json
			parts := strings.SplitN(f.Name, "/", 2)
			if len(parts) != 2 {
				continue
			}
			setName := parts[0]
			spec, ok := store.EntitySpecs[setName]
			if !ok {
				continue
			}
			var payload map[string]any
			if err := json.Unmarshal(content, &payload); err != nil {
				continue
			}
			if _, err := d.DB.InsertGeneric(c.UserContext(), spec, payload); err == nil {
				stats[setName] = increment(stats, setName)
			}
		}
		return c.JSON(stats)
	}
}

func ValidateImportHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		body := c.Body()
		zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}
		names := []string{}
		for _, f := range zr.File {
			names = append(names, f.Name)
		}
		return c.JSON(fiber.Map{"valid": true, "files": names})
	}
}

func increment(m fiber.Map, key string) int {
	if v, ok := m[key].(int); ok {
		return v + 1
	}
	return 1
}

func readAll(r interface{ Read(p []byte) (int, error) }) ([]byte, error) {
	var out bytes.Buffer
	buf := make([]byte, 32*1024)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			out.Write(buf[:n])
		}
		if err != nil {
			if err.Error() == "EOF" {
				return out.Bytes(), nil
			}
			return nil, err
		}
	}
}

func snapshotZip(c *fiber.Ctx, d *Deps) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for setName, spec := range store.EntitySpecs {
		items, err := d.DB.ListGeneric(c.UserContext(), spec, "", nil, "", 10000, 0)
		if err != nil {
			continue
		}
		for _, it := range items {
			shortid, _ := it["shortid"].(string)
			if shortid == "" {
				shortid = fmt.Sprintf("%v", it["id"])
			}
			body, _ := json.MarshalIndent(it, "", "  ")
			w, err := zw.Create(setName + "/" + shortid + ".json")
			if err != nil {
				return nil, err
			}
			w.Write(body)
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ----- Template sharing / public list -----

func TemplateSharingHandler(d *Deps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		access := c.Params("access") // "public" or "deny"
		shortid := c.Params("shortid")
		isPublic := access == "public"
		_, err := d.DB.Pool.Exec(c.UserContext(),
			`UPDATE templates SET is_public = $1, updated_at = NOW() WHERE shortid = $2`,
			isPublic, shortid)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"shortid": shortid, "access": access})
	}
}

func PublicTemplatesHandler(d *Deps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		rows, err := d.DB.Pool.Query(c.UserContext(),
			`SELECT shortid, name, recipe, engine FROM templates WHERE is_public = true ORDER BY name`)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		defer rows.Close()
		var out []fiber.Map
		for rows.Next() {
			var sid, name, recipe, engine string
			if err := rows.Scan(&sid, &name, &recipe, &engine); err == nil {
				out = append(out, fiber.Map{"shortid": sid, "name": name, "recipe": recipe, "engine": engine})
			}
		}
		return c.JSON(out)
	}
}
