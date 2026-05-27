package auth

import (
	"testing"
	"time"

	"github.com/cloudreport/api/internal/config"
	"github.com/cloudreport/api/internal/models"
)

func newTestService(secret string, ttl int) *Service {
	return &Service{cfg: &config.Config{JWTSecret: secret, SessionTTLHours: ttl}}
}

func TestSignAndParseJWT(t *testing.T) {
	s := newTestService("test-secret", 24)
	u := &models.User{ID: "u-123", Username: "alice", IsAdmin: true}

	token, err := s.signJWT(u)
	if err != nil {
		t.Fatalf("signJWT: %v", err)
	}
	if token == "" {
		t.Fatal("empty token")
	}

	claims, err := s.ParseJWT(token)
	if err != nil {
		t.Fatalf("ParseJWT: %v", err)
	}
	if claims.UserID != "u-123" {
		t.Errorf("UserID = %q, want u-123", claims.UserID)
	}
	if claims.Username != "alice" {
		t.Errorf("Username = %q, want alice", claims.Username)
	}
	if !claims.IsAdmin {
		t.Errorf("IsAdmin = %v, want true", claims.IsAdmin)
	}
	if claims.Subject != "u-123" {
		t.Errorf("Subject = %q, want u-123", claims.Subject)
	}
	if claims.Issuer != "cloud-report" {
		t.Errorf("Issuer = %q, want cloud-report", claims.Issuer)
	}
	// Expiry should be in the future, roughly ttl hours out.
	if claims.ExpiresAt == nil || !claims.ExpiresAt.After(time.Now()) {
		t.Errorf("ExpiresAt not in future: %v", claims.ExpiresAt)
	}
}

func TestParseJWTWrongSecret(t *testing.T) {
	signer := newTestService("real-secret", 24)
	token, err := signer.signJWT(&models.User{ID: "x", Username: "u"})
	if err != nil {
		t.Fatal(err)
	}
	verifier := newTestService("other-secret", 24)
	if _, err := verifier.ParseJWT(token); err == nil {
		t.Error("expected error parsing with wrong secret")
	}
}

func TestParseJWTGarbage(t *testing.T) {
	s := newTestService("secret", 24)
	if _, err := s.ParseJWT("not.a.jwt"); err == nil {
		t.Error("expected error for malformed token")
	}
}

func TestRefreshProducesParseableToken(t *testing.T) {
	s := newTestService("secret", 12)
	u := &models.User{ID: "u-9", Username: "bob", IsAdmin: false}
	token, err := s.Refresh(u)
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	claims, err := s.ParseJWT(token)
	if err != nil {
		t.Fatalf("ParseJWT(refresh): %v", err)
	}
	if claims.UserID != "u-9" || claims.Username != "bob" || claims.IsAdmin {
		t.Errorf("unexpected claims: %+v", claims)
	}
}

func TestSignJWTZeroTTLFallback(t *testing.T) {
	// SessionTTLHours <= 0 should fall back to a 30-day expiry rather than
	// producing an already-expired token.
	s := newTestService("secret", 0)
	token, err := s.signJWT(&models.User{ID: "z", Username: "z"})
	if err != nil {
		t.Fatal(err)
	}
	claims, err := s.ParseJWT(token)
	if err != nil {
		t.Fatalf("ParseJWT: %v", err)
	}
	if claims.ExpiresAt == nil || !claims.ExpiresAt.After(time.Now().Add(20*24*time.Hour)) {
		t.Errorf("expected ~30d expiry fallback, got %v", claims.ExpiresAt)
	}
}

func TestHashKeyDeterministic(t *testing.T) {
	a := hashKey("cr_abc_def")
	b := hashKey("cr_abc_def")
	if a != b {
		t.Error("hashKey not deterministic")
	}
	if a == hashKey("cr_abc_xyz") {
		t.Error("different inputs hashed equal")
	}
	if len(a) != 64 { // sha256 hex
		t.Errorf("hash len = %d, want 64", len(a))
	}
}

func TestRandHex(t *testing.T) {
	h := randHex(4)
	if len(h) != 8 { // n bytes -> 2n hex chars
		t.Errorf("randHex(4) len = %d, want 8", len(h))
	}
	if randHex(20) == randHex(20) {
		t.Error("randHex should produce distinct values")
	}
}

func TestEmailPtr(t *testing.T) {
	if emailPtr("") != nil {
		t.Error("empty email should be nil")
	}
	p := emailPtr("a@b.com")
	if p == nil || *p != "a@b.com" {
		t.Errorf("emailPtr = %v, want a@b.com", p)
	}
}
