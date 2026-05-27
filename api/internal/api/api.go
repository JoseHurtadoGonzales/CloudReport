// Package api wires all HTTP routes to handlers. It mirrors jsreport's
// public surface (POST /api/report, /odata/*, /api/ping, etc.) as closely as
// is reasonable for a Go implementation.
package api

import (
	"github.com/cloudreport/api/internal/auth"
	"github.com/cloudreport/api/internal/blob"
	"github.com/cloudreport/api/internal/config"
	"github.com/cloudreport/api/internal/odata"
	"github.com/cloudreport/api/internal/queue"
	"github.com/cloudreport/api/internal/render"
	"github.com/cloudreport/api/internal/store"
	"github.com/cloudreport/api/internal/ws"
	"github.com/gofiber/fiber/v2"
	cws "github.com/gofiber/contrib/websocket"
)

type Deps struct {
	Cfg      *config.Config
	DB       *store.Store
	Blob     *blob.Store
	Queue    *queue.Queue
	Auth     *auth.Service
	Renderer *render.Renderer
	Hub      *ws.Hub
}

func Register(app *fiber.App, d *Deps) {
	// Public endpoints
	app.Get("/api/ping", PingHandler())
	app.Get("/api/version", VersionHandler())

	// Auth
	app.Post("/api/auth/register", RegisterHandler(d))
	app.Post("/api/auth/login", LoginHandler(d))
	app.Post("/api/auth/logout", LogoutHandler())

	// API keys (require user JWT, not API key, to prevent infinite recursion of
	// privilege escalation).
	apiKeys := app.Group("/api/apikeys", d.Auth.Required())
	apiKeys.Get("/", ListAPIKeysHandler(d))
	apiKeys.Post("/", CreateAPIKeyHandler(d))
	apiKeys.Delete("/:id", RevokeAPIKeyHandler(d))

	// Authenticated user info
	app.Get("/api/current-user", d.Auth.Required(), CurrentUserHandler())
	app.Post("/api/auth/refresh", d.Auth.Required(), RefreshHandler(d))
	app.Post("/api/users/:shortid/password", d.Auth.Required(), ChangePasswordHandler(d))

	// Admin-only user management. Handlers check `actor.IsAdmin` themselves
	// since we don't have a separate middleware for that yet.
	app.Post("/api/admin/users",            d.Auth.Required(), AdminCreateUserHandler(d))
	app.Patch("/api/admin/users/:shortid",  d.Auth.Required(), AdminPatchUserHandler(d))
	app.Delete("/api/admin/users/:shortid", d.Auth.Required(), AdminDeleteUserHandler(d))

	// Settings & metadata (open with optional auth)
	app.Get("/api/settings", d.Auth.Optional(), SettingsHandler())
	app.Get("/api/recipe", RecipesHandler(d))
	app.Get("/api/engine", EnginesHandler(d))
	app.Get("/api/extensions", ExtensionsHandler())
	app.Get("/api/schema/:set", SchemaHandler())

	// Main rendering endpoint — accepts JWT, API key, or anonymous if template is public
	app.Post("/api/report", d.Auth.Optional(), ReportHandler(d))

	// Profiles
	app.Get("/api/profile/:id", d.Auth.Required(), ProfileGetHandler(d))
	app.Get("/api/profile/:id/events", d.Auth.Required(), ProfileEventsHandler(d))

	// Reports (already-rendered)
	app.Get("/reports/:shortid/content", d.Auth.Optional(), ReportContentHandler(d, false))
	app.Get("/reports/public/:shortid/content", ReportContentHandler(d, true))
	app.Get("/reports/:shortid/status", d.Auth.Optional(), ReportStatusHandler(d))

	// Assets binary content
	app.Get("/assets/:shortid/content", d.Auth.Optional(), AssetContentHandler(d))

	// Scheduling
	sched := app.Group("/api/scheduling", d.Auth.Required())
	sched.Get("/nextRun/:cron", NextRunHandler())
	sched.Post("/runNow", RunNowHandler(d))

	// Version control
	vc := app.Group("/api/version-control", d.Auth.Required())
	vc.Post("/commit", VCSCommitHandler(d))
	vc.Get("/history", VCSHistoryHandler(d))
	vc.Post("/revert", VCSRevertHandler(d))
	vc.Get("/local-changes", VCSLocalChangesHandler(d))

	// Import / Export
	app.Post("/api/export", d.Auth.Required(), ExportHandler(d))
	app.Post("/api/import", d.Auth.Required(), ImportHandler(d))
	app.Post("/api/validate-import", d.Auth.Required(), ValidateImportHandler())

	// Component render (jsreport-compatible)
	app.Post("/api/component", d.Auth.Optional(), ComponentRenderHandler(d))

	// Template sharing
	app.Post("/api/templates/sharing/:shortid/access/:access", d.Auth.Required(), TemplateSharingHandler(d))
	app.Get("/public-templates", PublicTemplatesHandler(d))

	// Asset upload (multipart)
	app.Post("/api/assets/upload", d.Auth.Required(), AssetUploadHandler(d))

	// Folder hierarchy & validation
	app.Post("/studio/hierarchyMove", d.Auth.Required(), HierarchyMoveHandler(d))
	app.Post("/studio/validate-entity-name", d.Auth.Optional(), ValidateEntityNameHandler(d))
	app.Get("/studio/text-search", d.Auth.Required(), TextSearchHandler(d))

	// Health
	app.Get("/api/health", HealthHandler(d))

	// Settings key/value
	app.Get("/api/kv/:key", d.Auth.Required(), KVGetHandler(d))
	app.Put("/api/kv/:key", d.Auth.Required(), KVPutHandler(d))

	// WebSocket events
	app.Get("/ws", cws.New(func(c *cws.Conn) {
		d.Hub.Register(c)
		defer d.Hub.Unregister(c)
		for {
			if _, _, err := c.ReadMessage(); err != nil {
				return
			}
		}
	}))

	// OData CRUD for all entity sets
	odata.Register(app, d.DB)
}
