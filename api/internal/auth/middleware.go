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
