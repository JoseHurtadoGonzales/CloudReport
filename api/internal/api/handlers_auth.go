package api

import (
	"errors"
	"time"

	"github.com/cloudreport/api/internal/auth"
	"github.com/cloudreport/api/internal/store"
	"github.com/gofiber/fiber/v2"
)

type registerReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
}

func RegisterHandler(d *Deps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if !d.Cfg.AllowRegistration {
			return c.Status(403).JSON(fiber.Map{"error": "registration disabled"})
		}
		var req registerReq
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}
		if req.Username == "" || len(req.Password) < 6 {
			return c.Status(400).JSON(fiber.Map{"error": "username and password (>=6) required"})
		}
		u, err := d.Auth.Register(c.UserContext(), req.Username, req.Password, req.Email)
		if errors.Is(err, auth.ErrUserExists) {
			return c.Status(409).JSON(fiber.Map{"error": "user exists"})
		}
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		_, token, _ := d.Auth.Login(c.UserContext(), req.Username, req.Password)
		return c.Status(201).JSON(fiber.Map{
			"user":  u,
			"token": token,
		})
	}
}

type loginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func LoginHandler(d *Deps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req loginReq
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}
		u, token, err := d.Auth.Login(c.UserContext(), req.Username, req.Password)
		if err != nil {
			return c.Status(401).JSON(fiber.Map{"error": "invalid credentials"})
		}
		c.Cookie(&fiber.Cookie{
			Name: "session", Value: token, HTTPOnly: true, Secure: false, Path: "/",
			Expires: time.Now().Add(12 * time.Hour),
		})
		return c.JSON(fiber.Map{
			"user":  u,
			"token": token,
		})
	}
}

func LogoutHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Cookie(&fiber.Cookie{Name: "session", Value: "", Expires: time.Now().Add(-time.Hour), Path: "/"})
		return c.SendStatus(204)
	}
}

func CurrentUserHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		u := auth.UserFrom(c)
		return c.JSON(fiber.Map{"username": u.Username, "isAdmin": u.IsAdmin, "shortid": u.Shortid})
	}
}

// RefreshHandler hands back a freshly-signed token for the current user. The
// request must carry a still-valid token (enforced by d.Auth.Required()).
// The frontend calls this on load / focus / on a timer so the session keeps
// sliding forward and the user is never unexpectedly logged out.
func RefreshHandler(d *Deps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		u := auth.UserFrom(c)
		if u == nil {
			return c.Status(401).JSON(fiber.Map{"error": "not authenticated"})
		}
		token, err := d.Auth.Refresh(u)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		c.Cookie(&fiber.Cookie{
			Name: "session", Value: token, HTTPOnly: true, Secure: false, Path: "/",
			Expires: time.Now().Add(time.Duration(d.Cfg.SessionTTLHours) * time.Hour),
		})
		return c.JSON(fiber.Map{
			"user":  fiber.Map{"username": u.Username, "isAdmin": u.IsAdmin, "shortid": u.Shortid},
			"token": token,
		})
	}
}

type changePasswordReq struct {
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
}

func ChangePasswordHandler(d *Deps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		u := auth.UserFrom(c)
		if u.Shortid != c.Params("shortid") && !u.IsAdmin {
			return c.Status(403).JSON(fiber.Map{"error": "forbidden"})
		}
		var req changePasswordReq
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}
		if err := d.Auth.ChangePassword(c.UserContext(), u, req.OldPassword, req.NewPassword); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}
		return c.SendStatus(204)
	}
}

// ---------- API KEYS ----------

type createKeyReq struct {
	Name      string   `json:"name"`
	Scopes    []string `json:"scopes"`
	TTLDays   int      `json:"ttlDays"`
}

func CreateAPIKeyHandler(d *Deps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		u := auth.UserFrom(c)
		var req createKeyReq
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}
		var ttl time.Duration
		if req.TTLDays > 0 {
			ttl = time.Duration(req.TTLDays) * 24 * time.Hour
		}
		k, raw, err := d.Auth.CreateAPIKey(c.UserContext(), u.ID, req.Name, req.Scopes, ttl)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(201).JSON(fiber.Map{
			"apiKey": k,
			"key":    raw,
			"note":   "store this key now — it will not be shown again",
		})
	}
}

func ListAPIKeysHandler(d *Deps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		u := auth.UserFrom(c)
		keys, err := d.Auth.ListAPIKeys(c.UserContext(), u.ID)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(keys)
	}
}

func RevokeAPIKeyHandler(d *Deps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		u := auth.UserFrom(c)
		id := c.Params("id")
		if err := d.Auth.RevokeAPIKey(c.UserContext(), u.ID, id); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.SendStatus(204)
	}
}

// ─── Admin: user management ───────────────────────────────────────────────
// These endpoints sit behind d.Auth.Required() and an in-handler isAdmin
// check (we don't have a separate "RequireAdmin" middleware yet — folding
// the check in keeps the router config simple).

type adminCreateUserReq struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	IsAdmin  bool   `json:"isAdmin"`
}

func AdminCreateUserHandler(d *Deps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		actor := auth.UserFrom(c)
		if actor == nil || !actor.IsAdmin {
			return c.Status(403).JSON(fiber.Map{"error": "admin required"})
		}
		var req adminCreateUserReq
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}
		if req.Username == "" || len(req.Password) < 6 {
			return c.Status(400).JSON(fiber.Map{"error": "username and password (>=6) required"})
		}
		u, err := d.Auth.Register(c.UserContext(), req.Username, req.Password, req.Email)
		if errors.Is(err, auth.ErrUserExists) {
			return c.Status(409).JSON(fiber.Map{"error": "user exists"})
		}
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		// Optionally promote the freshly-created user — Register() always
		// creates non-admin accounts, so we patch the flag afterwards.
		if req.IsAdmin {
			spec := store.EntitySpecs["users"]
			_, _ = d.DB.UpdateGeneric(c.UserContext(), spec, u.Shortid, map[string]any{
				"is_admin": true,
			})
			u.IsAdmin = true
		}
		return c.Status(201).JSON(fiber.Map{"user": u})
	}
}

func AdminDeleteUserHandler(d *Deps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		actor := auth.UserFrom(c)
		if actor == nil || !actor.IsAdmin {
			return c.Status(403).JSON(fiber.Map{"error": "admin required"})
		}
		shortid := c.Params("shortid")
		if shortid == "" {
			return c.Status(400).JSON(fiber.Map{"error": "shortid required"})
		}
		// Block self-delete — otherwise an admin can lock themselves out
		// of the system with a single click.
		if shortid == actor.Shortid {
			return c.Status(400).JSON(fiber.Map{"error": "cannot delete yourself"})
		}
		spec := store.EntitySpecs["users"]
		if err := d.DB.DeleteGeneric(c.UserContext(), spec, shortid); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.SendStatus(204)
	}
}

type adminPatchUserReq struct {
	IsAdmin *bool `json:"isAdmin,omitempty"`
	Email   *string `json:"email,omitempty"`
}

func AdminPatchUserHandler(d *Deps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		actor := auth.UserFrom(c)
		if actor == nil || !actor.IsAdmin {
			return c.Status(403).JSON(fiber.Map{"error": "admin required"})
		}
		shortid := c.Params("shortid")
		var req adminPatchUserReq
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}
		patch := map[string]any{}
		if req.IsAdmin != nil {
			// Block self-demote — same reasoning as the delete handler.
			if shortid == actor.Shortid && !*req.IsAdmin {
				return c.Status(400).JSON(fiber.Map{"error": "cannot demote yourself"})
			}
			patch["is_admin"] = *req.IsAdmin
		}
		if req.Email != nil {
			patch["email"] = *req.Email
		}
		if len(patch) == 0 {
			return c.Status(400).JSON(fiber.Map{"error": "nothing to update"})
		}
		spec := store.EntitySpecs["users"]
		out, err := d.DB.UpdateGeneric(c.UserContext(), spec, shortid, patch)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(out)
	}
}
