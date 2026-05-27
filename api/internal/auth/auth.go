// Package auth implements user registration, login (JWT), and API key handling.
//
// We support two credentials:
//
//   - Bearer JWT: short-lived (12h), issued at /api/auth/login.
//   - API key: long-lived static string of the form "cr_<prefix>_<secret>".
//     Sent via header `x-api-key`. Only its SHA-256 hash is stored in DB.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cloudreport/api/internal/config"
	"github.com/cloudreport/api/internal/models"
	"github.com/cloudreport/api/internal/store"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	cfg *config.Config
	db  *store.Store
}

func New(cfg *config.Config, db *store.Store) *Service {
	return &Service{cfg: cfg, db: db}
}

// ---------- USERS ----------

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserExists         = errors.New("user already exists")
	ErrUnauthorized       = errors.New("unauthorized")
)

func (s *Service) Register(ctx context.Context, username, password, email string) (*models.User, error) {
	if _, err := s.db.GetUserByUsername(ctx, username); err == nil {
		return nil, ErrUserExists
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	// First user gets admin.
	count, _ := s.db.CountUsers(ctx)
	u := &models.User{
		Username:     username,
		Email:        emailPtr(email),
		PasswordHash: string(hash),
		IsAdmin:      count == 0,
	}
	if err := s.db.CreateUser(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

func emailPtr(e string) *string {
	if e == "" {
		return nil
	}
	return &e
}

func (s *Service) Login(ctx context.Context, username, password string) (*models.User, string, error) {
	u, err := s.db.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, "", ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return nil, "", ErrInvalidCredentials
	}
	token, err := s.signJWT(u)
	if err != nil {
		return nil, "", err
	}
	return u, token, nil
}

func (s *Service) ChangePassword(ctx context.Context, u *models.User, oldP, newP string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(oldP)); err != nil {
		return ErrInvalidCredentials
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newP), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.db.UpdateUserPassword(ctx, u.Shortid, string(hash))
}

// ---------- JWT ----------

type Claims struct {
	UserID   string `json:"uid"`
	Username string `json:"u"`
	IsAdmin  bool   `json:"adm"`
	jwt.RegisteredClaims
}

func (s *Service) signJWT(u *models.User) (string, error) {
	ttl := time.Duration(s.cfg.SessionTTLHours) * time.Hour
	if ttl <= 0 {
		ttl = 720 * time.Hour // 30-day fallback
	}
	c := Claims{
		UserID:   u.ID,
		Username: u.Username,
		IsAdmin:  u.IsAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   u.ID,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "cloud-report",
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	return t.SignedString([]byte(s.cfg.JWTSecret))
}

// Refresh issues a brand-new token for an already-authenticated user. The
// caller must have validated the existing token (via middleware). This gives
// us a sliding session: every refresh resets the expiry clock, so an active
// user never gets logged out, while an idle one eventually expires.
func (s *Service) Refresh(u *models.User) (string, error) {
	return s.signJWT(u)
}

func (s *Service) ParseJWT(token string) (*Claims, error) {
	c := &Claims{}
	_, err := jwt.ParseWithClaims(token, c, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(s.cfg.JWTSecret), nil
	})
	return c, err
}

// ---------- API KEYS ----------

// CreateAPIKey returns the API key model AND the raw token (only available
// at this moment). Format: cr_<prefix8>_<random40>.
func (s *Service) CreateAPIKey(ctx context.Context, userID, name string, scopes []string, ttl time.Duration) (*models.APIKey, string, error) {
	if name == "" {
		return nil, "", errors.New("name required")
	}
	if len(scopes) == 0 {
		scopes = []string{"render", "read", "write"}
	}
	prefix := randHex(4) // 8 chars
	secret := randHex(20) // 40 chars
	raw := fmt.Sprintf("cr_%s_%s", prefix, secret)
	hash := hashKey(raw)
	var expiresAt *time.Time
	if ttl > 0 {
		t := time.Now().Add(ttl)
		expiresAt = &t
	}
	k, err := s.db.CreateAPIKey(ctx, userID, name, prefix, hash, scopes, expiresAt)
	if err != nil {
		return nil, "", err
	}
	k.Raw = raw
	return k, raw, nil
}

func (s *Service) ResolveAPIKey(ctx context.Context, raw string) (*models.User, *models.APIKey, error) {
	if !strings.HasPrefix(raw, "cr_") {
		return nil, nil, ErrUnauthorized
	}
	hash := hashKey(raw)
	k, u, err := s.db.GetAPIKeyByHash(ctx, hash)
	if err != nil {
		return nil, nil, ErrUnauthorized
	}
	if k.RevokedAt != nil {
		return nil, nil, ErrUnauthorized
	}
	if k.ExpiresAt != nil && time.Now().After(*k.ExpiresAt) {
		return nil, nil, ErrUnauthorized
	}
	go s.db.TouchAPIKey(context.Background(), k.ID)
	return u, k, nil
}

func (s *Service) ListAPIKeys(ctx context.Context, userID string) ([]*models.APIKey, error) {
	return s.db.ListAPIKeys(ctx, userID)
}

func (s *Service) RevokeAPIKey(ctx context.Context, userID, id string) error {
	return s.db.RevokeAPIKey(ctx, userID, id)
}

func hashKey(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

func randHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
