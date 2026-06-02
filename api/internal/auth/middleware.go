package auth

import (
	"strings"

	"github.com/cloudreport/api/internal/models"
	"github.com/gofiber/fiber/v2"
)

const ContextUserKey = "auth:user"
const ContextAPIKeyKey = "auth:apikey"

// Required is a Fiber middleware that accepts either:
//   - Authorization: Bearer <jwt>
//   - x-api-key: cr_<prefix>_<secret>
//
// On success it stores *models.User in Locals(ContextUserKey).
func (s *Service) Required() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if u, ok := s.resolve(c); ok {
			c.Locals(ContextUserKey, u)
			return c.Next()
		}
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}
}

// Optional populates the user if present but never blocks.
func (s *Service) Optional() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if u, ok := s.resolve(c); ok {
			c.Locals(ContextUserKey, u)
		}
		return c.Next()
	}
}

func (s *Service) AdminOnly() fiber.Handler {
	return func(c *fiber.Ctx) error {
		u := UserFrom(c)
		if u == nil || !u.IsAdmin {
			return c.Status(403).JSON(fiber.Map{"error": "admin only"})
		}
		return c.Next()
	}
}

func UserFrom(c *fiber.Ctx) *models.User {
	v := c.Locals(ContextUserKey)
	if v == nil {
		return nil
	}
	u, _ := v.(*models.User)
	return u
}

// IsAPIKey reports whether the current request was authenticated with an API
// key (rather than a Bearer JWT). Only meaningful after Required()/Optional()
// has run on the route.
func IsAPIKey(c *fiber.Ctx) bool {
	return c.Locals(ContextAPIKeyKey) != nil
}

// RejectAPIKey blocks requests authenticated via API key. Used to keep
// user-management endpoints reachable only with a real logged-in session
// (JWT) — an API key must not be able to create/edit/delete users. Place it
// AFTER Required() so the auth context is populated.
func (s *Service) RejectAPIKey() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if IsAPIKey(c) {
			return c.Status(403).JSON(fiber.Map{"error": "las API keys no pueden gestionar usuarios; usá una sesión con login"})
		}
		return c.Next()
	}
}

func (s *Service) resolve(c *fiber.Ctx) (*models.User, bool) {
	ctx := c.UserContext()

	// API key
	if k := strings.TrimSpace(c.Get("x-api-key")); k != "" {
		u, apiKey, err := s.ResolveAPIKey(ctx, k)
		if err == nil {
			c.Locals(ContextAPIKeyKey, apiKey)
			return u, true
		}
	}

	// Bearer JWT
	authH := c.Get("Authorization")
	if strings.HasPrefix(authH, "Bearer ") {
		token := strings.TrimSpace(strings.TrimPrefix(authH, "Bearer "))
		claims, err := s.ParseJWT(token)
		if err == nil {
			u, err := s.db.GetUserByID(ctx, claims.UserID)
			if err == nil {
				return u, true
			}
		}
	}

	return nil, false
}
