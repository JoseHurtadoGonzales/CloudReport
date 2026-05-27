package odata

import (
	"encoding/json"

	"github.com/cloudreport/api/internal/auth"
	"github.com/cloudreport/api/internal/store"
	"github.com/gofiber/fiber/v2"
)

// Handler registers `/odata/:entitySet` routes for every known entity.
func Register(app fiber.Router, db *store.Store) {
	app.Get("/odata/:set", listHandler(db))
	app.Get("/odata/:set/:shortid", getHandler(db))
	app.Post("/odata/:set", insertHandler(db))
	app.Patch("/odata/:set/:shortid", updateHandler(db))
	app.Put("/odata/:set/:shortid", updateHandler(db))
	app.Delete("/odata/:set/:shortid", deleteHandler(db))
}

func specFor(c *fiber.Ctx) (store.EntitySpec, bool) {
	name := c.Params("set")
	spec, ok := store.EntitySpecs[name]
	return spec, ok
}

func listHandler(db *store.Store) fiber.Handler {
	return func(c *fiber.Ctx) error {
		spec, ok := specFor(c)
		if !ok {
			return c.Status(404).JSON(fiber.Map{"error": "unknown entity set"})
		}
		raw := c.Queries()
		q, err := Parse(raw, spec.Columns, jsreportAliases)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}
		items, err := db.ListGeneric(c.UserContext(), spec, q.Filter, q.Args, q.OrderBy, q.Top, q.Skip)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		// Permission scoping: admins (or users with read_all) see everything.
		// Otherwise the user must own the row, OR be listed in read_perms,
		// OR belong to a group listed in read_perms.
		u := auth.UserFrom(c)
		if u != nil && !u.IsAdmin && !u.ReadAll {
			principals, _ := db.UserPrincipals(c.UserContext(), u.ID)
			principals = append(principals, u.ID)
			items = scopeByPermissions(items, u.ID, principals, "read_perms")
		}

		resp := fiber.Map{"value": toJSReport(items)}
		if q.Count || q.InlineCount {
			n, _ := db.CountGeneric(c.UserContext(), spec, q.Filter, q.Args)
			resp["@odata.count"] = n
		}
		return c.JSON(resp)
	}
}

func getHandler(db *store.Store) fiber.Handler {
	return func(c *fiber.Ctx) error {
		spec, ok := specFor(c)
		if !ok {
			return c.Status(404).JSON(fiber.Map{"error": "unknown entity set"})
		}
		shortid := c.Params("shortid")
		item, err := db.GetGenericByShortid(c.UserContext(), spec, shortid)
		if err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "not found"})
		}
		return c.JSON(toJSReportOne(item))
	}
}

func insertHandler(db *store.Store) fiber.Handler {
	return func(c *fiber.Ctx) error {
		spec, ok := specFor(c)
		if !ok {
			return c.Status(404).JSON(fiber.Map{"error": "unknown entity set"})
		}
		body := map[string]any{}
		if err := json.Unmarshal(c.Body(), &body); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}
		payload := fromJSReport(body)
		if u := auth.UserFrom(c); u != nil {
			if _, hasOwner := payload["owner_id"]; !hasOwner {
				for _, col := range spec.Insertable {
					if col == "owner_id" {
						payload["owner_id"] = u.ID
						break
					}
				}
			}
		}
		item, err := db.InsertGeneric(c.UserContext(), spec, payload)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(201).JSON(toJSReportOne(item))
	}
}

func updateHandler(db *store.Store) fiber.Handler {
	return func(c *fiber.Ctx) error {
		spec, ok := specFor(c)
		if !ok {
			return c.Status(404).JSON(fiber.Map{"error": "unknown entity set"})
		}
		shortid := c.Params("shortid")
		body := map[string]any{}
		if err := json.Unmarshal(c.Body(), &body); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}

		// Authorization: must hold edit perm or be owner / admin.
		if err := checkEditPerm(c, db, spec, shortid); err != nil {
			return c.Status(403).JSON(fiber.Map{"error": err.Error()})
		}

		payload := fromJSReport(body)
		item, err := db.UpdateGeneric(c.UserContext(), spec, shortid, payload)
		if err != nil {
			return c.Status(404).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(toJSReportOne(item))
	}
}

func checkEditPerm(c *fiber.Ctx, db *store.Store, spec store.EntitySpec, shortid string) error {
	u := auth.UserFrom(c)
	if u == nil {
		return nil // optional auth: let DB layer decide
	}
	if u.IsAdmin || u.EditAll {
		return nil
	}
	cur, err := db.GetGenericByShortid(c.UserContext(), spec, shortid)
	if err != nil {
		return nil
	}
	principals, _ := db.UserPrincipals(c.UserContext(), u.ID)
	if allowed(cur, u.ID, principals, "edit_perms") {
		return nil
	}
	return fiber.NewError(403, "edit forbidden")
}

func deleteHandler(db *store.Store) fiber.Handler {
	return func(c *fiber.Ctx) error {
		spec, ok := specFor(c)
		if !ok {
			return c.Status(404).JSON(fiber.Map{"error": "unknown entity set"})
		}
		shortid := c.Params("shortid")
		if err := checkEditPerm(c, db, spec, shortid); err != nil {
			return c.Status(403).JSON(fiber.Map{"error": err.Error()})
		}
		if err := db.DeleteGeneric(c.UserContext(), spec, shortid); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.SendStatus(204)
	}
}

// jsreportAliases maps the public JSON keys (camelCase) to internal SQL columns.
var jsreportAliases = map[string]string{
	"_id":                  "id",
	"folder":               "folder_id",
	"templateShortid":      "template_shortid",
	"isSharedHelper":       "is_shared_helper",
	"sharedHelpersScope":   "shared_helpers_scope",
	"mimeType":             "mime_type",
	"isGlobal":             "is_global",
	"isPublic":             "is_public",
	"dataJson":             "data_json",
	"sizeBytes":            "size_bytes",
	"creationDate":         "created_at",
	"modificationDate":     "updated_at",
	"timestamp":            "started_at",
	"finishedOn":           "finished_at",
	"nextRun":              "next_run",
	"timeout":              "timeout_ms",
	"readPermissions":      "read_perms",
	"editPermissions":      "edit_perms",
	"ownerId":              "owner_id",
	"pdfOperations":        "pdf_operations",
	"pageSize":             "page_size",
	"pageOrientation":      "page_orientation",
	"pageMargin":           "page_margin",
	"htmlToXlsx":           "html_to_xlsx",
	// users entity
	"isAdmin":              "is_admin",
	"readAll":              "read_all",
	"editAll":              "edit_all",
	// reports retention
	"reportRetentionDays":  "report_retention_days",
	"expiresAt":            "expires_at",
}

// scopeByPermissions filters a list of rows down to those the user can access.
// A row is included when ANY of these is true:
//
//   - the row has no owner_id and no perms column (truly public)
//   - the row is owned by the user
//   - the user (or one of their groups) appears in the perms column
//
// permsCol is either "read_perms" or "edit_perms".
func scopeByPermissions(items []map[string]any, userID string, principals []string, permsCol string) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, it := range items {
		if allowed(it, userID, principals, permsCol) {
			out = append(out, it)
		}
	}
	return out
}

func allowed(it map[string]any, userID string, principals []string, permsCol string) bool {
	if v, ok := it["owner_id"].(string); ok && v == userID {
		return true
	}
	if v, ok := it[permsCol]; ok && v != nil {
		// pgx returns []string for text[] columns, fall back to []any if not.
		switch perms := v.(type) {
		case []string:
			if intersects(perms, principals) {
				return true
			}
		case []any:
			conv := make([]string, 0, len(perms))
			for _, p := range perms {
				if s, ok := p.(string); ok {
					conv = append(conv, s)
				}
			}
			if intersects(conv, principals) {
				return true
			}
		}
		// Row has perms but user is not listed: explicit denial.
		// But: rows with EMPTY perms are open to everyone authenticated.
		switch perms := v.(type) {
		case []string:
			return len(perms) == 0
		case []any:
			return len(perms) == 0
		}
	}
	// No owner, no perms column → public.
	_, hasOwner := it["owner_id"]
	return !hasOwner
}

func intersects(a, b []string) bool {
	if len(a) == 0 || len(b) == 0 {
		return false
	}
	set := make(map[string]struct{}, len(a))
	for _, x := range a {
		set[x] = struct{}{}
	}
	for _, y := range b {
		if _, ok := set[y]; ok {
			return true
		}
	}
	return false
}

// toJSReport renames SQL columns back to camelCase for the response.
func toJSReport(items []map[string]any) []map[string]any {
	out := make([]map[string]any, len(items))
	for i, it := range items {
		out[i] = toJSReportOne(it)
	}
	return out
}

var reverseAlias = func() map[string]string {
	m := map[string]string{}
	for k, v := range jsreportAliases {
		m[v] = k
	}
	return m
}()

func toJSReportOne(it map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range it {
		if alias, ok := reverseAlias[k]; ok {
			out[alias] = v
		} else {
			out[k] = v
		}
	}
	return out
}

func fromJSReport(body map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range body {
		if alias, ok := jsreportAliases[k]; ok {
			out[alias] = v
		} else {
			out[k] = v
		}
	}
	return out
}
