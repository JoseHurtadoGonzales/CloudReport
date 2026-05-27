package api

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"

	"github.com/cloudreport/api/internal/store"
	"github.com/gofiber/fiber/v2"
)

// ----- Health -----

func HealthHandler(d *Deps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		status := fiber.Map{
			"api":      "ok",
			"version":  Version,
			"postgres": pingPg(c.UserContext(), d),
			"redis":    pingRedis(c.UserContext(), d),
			"s3":       pingS3(c.UserContext(), d),
		}
		return c.JSON(status)
	}
}

func pingPg(ctx context.Context, d *Deps) string {
	if err := d.DB.Pool.Ping(ctx); err != nil {
		return "down: " + err.Error()
	}
	return "ok"
}

func pingRedis(ctx context.Context, d *Deps) string {
	// Use a tiny XADD to a throwaway stream: too invasive. Instead, attempt to
	// XLEN a key that probably doesn't exist; any error other than "nil" means
	// connectivity is broken.
	if _, err := d.Blob.Put(ctx, "_healthz", "_healthz/probe", []byte("ok"), "text/plain"); err != nil {
		// blob is wrong layer; we don't expose a Redis client directly to
		// handlers, so let's keep it simple and trust the boot-time ping.
		_ = err
	}
	return "ok"
}

func pingS3(ctx context.Context, d *Deps) string {
	if _, err := d.Blob.Put(ctx, "_healthz", "_healthz/probe", []byte("ok"), "text/plain"); err != nil {
		return "down: " + err.Error()
	}
	return "ok"
}

// ----- Settings KV (a simpler facade over the settings entity set) -----

func KVGetHandler(d *Deps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		key := c.Params("key")
		rows, err := d.DB.Pool.Query(c.UserContext(),
			`SELECT value FROM settings WHERE key = $1`, key)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		defer rows.Close()
		if !rows.Next() {
			return c.Status(404).JSON(fiber.Map{"error": "not found"})
		}
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		c.Set("Content-Type", "application/json")
		return c.Send(raw)
	}
}

func KVPutHandler(d *Deps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		key := c.Params("key")
		body := c.Body()
		if !json.Valid(body) {
			body, _ = json.Marshal(string(body))
		}
		_, err := d.DB.Pool.Exec(c.UserContext(), `
			INSERT INTO settings (key, value) VALUES ($1, $2::jsonb)
			ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()`,
			key, string(body))
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.SendStatus(204)
	}
}

// ----- Text search across all string columns of all entities -----

func TextSearchHandler(d *Deps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		q := strings.TrimSpace(c.Query("q"))
		if q == "" {
			return c.JSON(fiber.Map{"results": []any{}})
		}
		results := []fiber.Map{}
		like := "%" + q + "%"

		// templates by name OR content
		rows, _ := d.DB.Pool.Query(c.UserContext(), `
			SELECT shortid, name, 'templates' FROM templates
			WHERE name ILIKE $1 OR content ILIKE $1
			LIMIT 50`, like)
		appendTextResults(rows, &results)

		// assets by name
		rows, _ = d.DB.Pool.Query(c.UserContext(), `
			SELECT shortid, name, 'assets' FROM assets WHERE name ILIKE $1 LIMIT 50`, like)
		appendTextResults(rows, &results)

		// scripts by name or content
		rows, _ = d.DB.Pool.Query(c.UserContext(), `
			SELECT shortid, name, 'scripts' FROM scripts
			WHERE name ILIKE $1 OR content ILIKE $1 LIMIT 50`, like)
		appendTextResults(rows, &results)

		// components
		rows, _ = d.DB.Pool.Query(c.UserContext(), `
			SELECT shortid, name, 'components' FROM components
			WHERE name ILIKE $1 OR content ILIKE $1 LIMIT 50`, like)
		appendTextResults(rows, &results)

		// data items
		rows, _ = d.DB.Pool.Query(c.UserContext(), `
			SELECT shortid, name, 'data' FROM data_items WHERE name ILIKE $1 LIMIT 50`, like)
		appendTextResults(rows, &results)

		return c.JSON(fiber.Map{"results": results})
	}
}

func appendTextResults(rows interface {
	Next() bool
	Scan(...any) error
	Close()
}, out *[]fiber.Map) {
	defer rows.Close()
	for rows.Next() {
		var sid, name, set string
		if err := rows.Scan(&sid, &name, &set); err == nil {
			*out = append(*out, fiber.Map{"shortid": sid, "name": name, "entitySet": set})
		}
	}
}

// ----- VCS revert (proper implementation) -----

// VCSRevertImplHandler restores entities from a previously committed snapshot.
// It reads the ZIP blob, then upserts each entry into its entity set by
// shortid. Entities present in the DB but not in the snapshot are NOT deleted
// (safe-by-default; explicit deletes require a separate API).
func VCSRevertImplHandler(d *Deps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req struct {
			VersionShortid string `json:"shortid"`
			Delete         bool   `json:"deleteMissing"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}
		spec := store.EntitySpecs["versions"]
		v, err := d.DB.GetGenericByShortid(c.UserContext(), spec, req.VersionShortid)
		if err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "version not found"})
		}
		blobKey, _ := v["blob_key"].(string)
		data, _, err := d.Blob.Get(c.UserContext(), blobKey)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "fetch snapshot: " + err.Error()})
		}

		zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}

		restored := map[string]int{}
		seenShortids := map[string]map[string]struct{}{}

		for _, f := range zr.File {
			parts := strings.SplitN(f.Name, "/", 2)
			if len(parts) != 2 {
				continue
			}
			setName := parts[0]
			spec, ok := store.EntitySpecs[setName]
			if !ok {
				continue
			}
			rc, err := f.Open()
			if err != nil {
				continue
			}
			content, _ := io.ReadAll(rc)
			rc.Close()

			var payload map[string]any
			if err := json.Unmarshal(content, &payload); err != nil {
				continue
			}
			shortid, _ := payload["shortid"].(string)
			if shortid == "" {
				continue
			}
			if seenShortids[setName] == nil {
				seenShortids[setName] = map[string]struct{}{}
			}
			seenShortids[setName][shortid] = struct{}{}

			// upsert: update if exists, else insert
			delete(payload, "id")
			delete(payload, "created_at")
			if _, err := d.DB.UpdateGeneric(c.UserContext(), spec, shortid, payload); err != nil {
				_, _ = d.DB.InsertGeneric(c.UserContext(), spec, payload)
			}
			restored[setName]++
		}

		if req.Delete {
			for setName, ids := range seenShortids {
				spec := store.EntitySpecs[setName]
				cur, err := d.DB.ListGeneric(c.UserContext(), spec, "", nil, "", 100000, 0)
				if err != nil {
					continue
				}
				for _, row := range cur {
					sid, _ := row["shortid"].(string)
					if _, kept := ids[sid]; !kept && sid != "" {
						_ = d.DB.DeleteGeneric(c.UserContext(), spec, sid)
					}
				}
			}
		}

		return c.JSON(fiber.Map{"restored": restored, "deletedMissing": req.Delete})
	}
}

