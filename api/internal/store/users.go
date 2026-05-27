package store

import (
	"context"
	"errors"
	"time"

	"github.com/cloudreport/api/internal/models"
	"github.com/jackc/pgx/v5"
)

func (s *Store) CreateUser(ctx context.Context, u *models.User) error {
	if u.Shortid == "" {
		u.Shortid = newShortid()
	}
	row := s.Pool.QueryRow(ctx, `
		INSERT INTO users (shortid, username, email, password_hash, is_admin, read_all, edit_all)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING id, created_at, updated_at`,
		u.Shortid, u.Username, u.Email, u.PasswordHash, u.IsAdmin, u.ReadAll, u.EditAll)
	return row.Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
}

func (s *Store) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	var u models.User
	err := s.Pool.QueryRow(ctx, `
		SELECT id, shortid, username, email, password_hash, is_admin, read_all, edit_all, created_at, updated_at
		FROM users WHERE username = $1`, username).
		Scan(&u.ID, &u.Shortid, &u.Username, &u.Email, &u.PasswordHash, &u.IsAdmin, &u.ReadAll, &u.EditAll, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &u, err
}

func (s *Store) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	var u models.User
	err := s.Pool.QueryRow(ctx, `
		SELECT id, shortid, username, email, password_hash, is_admin, read_all, edit_all, created_at, updated_at
		FROM users WHERE id = $1`, id).
		Scan(&u.ID, &u.Shortid, &u.Username, &u.Email, &u.PasswordHash, &u.IsAdmin, &u.ReadAll, &u.EditAll, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &u, err
}

func (s *Store) CountUsers(ctx context.Context) (int, error) {
	var n int
	err := s.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&n)
	return n, err
}

func (s *Store) UpdateUserPassword(ctx context.Context, shortid, hash string) error {
	_, err := s.Pool.Exec(ctx, `UPDATE users SET password_hash=$2, updated_at=NOW() WHERE shortid=$1`, shortid, hash)
	return err
}

// ----- API KEYS -----

func (s *Store) CreateAPIKey(ctx context.Context, userID, name, prefix, hash string, scopes []string, expiresAt *time.Time) (*models.APIKey, error) {
	k := &models.APIKey{
		UserID:    userID,
		Name:      name,
		KeyPrefix: prefix,
		Scopes:    scopes,
		ExpiresAt: expiresAt,
	}
	row := s.Pool.QueryRow(ctx, `
		INSERT INTO api_keys (user_id, name, key_prefix, key_hash, scopes, expires_at)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING id, created_at`,
		userID, name, prefix, hash, scopes, expiresAt)
	if err := row.Scan(&k.ID, &k.CreatedAt); err != nil {
		return nil, err
	}
	return k, nil
}

func (s *Store) GetAPIKeyByHash(ctx context.Context, hash string) (*models.APIKey, *models.User, error) {
	var k models.APIKey
	var u models.User
	err := s.Pool.QueryRow(ctx, `
		SELECT k.id, k.user_id, k.name, k.key_prefix, k.scopes, k.last_used_at, k.expires_at, k.revoked_at, k.created_at,
		       u.id, u.shortid, u.username, u.email, u.is_admin, u.read_all, u.edit_all, u.created_at, u.updated_at
		FROM api_keys k JOIN users u ON u.id = k.user_id
		WHERE k.key_hash = $1`, hash).
		Scan(&k.ID, &k.UserID, &k.Name, &k.KeyPrefix, &k.Scopes, &k.LastUsedAt, &k.ExpiresAt, &k.RevokedAt, &k.CreatedAt,
			&u.ID, &u.Shortid, &u.Username, &u.Email, &u.IsAdmin, &u.ReadAll, &u.EditAll, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, ErrNotFound
	}
	if err != nil {
		return nil, nil, err
	}
	return &k, &u, nil
}

func (s *Store) TouchAPIKey(ctx context.Context, id string) {
	_, _ = s.Pool.Exec(ctx, `UPDATE api_keys SET last_used_at=NOW() WHERE id=$1`, id)
}

func (s *Store) ListAPIKeys(ctx context.Context, userID string) ([]*models.APIKey, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id, user_id, name, key_prefix, scopes, last_used_at, expires_at, revoked_at, created_at
		FROM api_keys WHERE user_id = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*models.APIKey
	for rows.Next() {
		var k models.APIKey
		if err := rows.Scan(&k.ID, &k.UserID, &k.Name, &k.KeyPrefix, &k.Scopes, &k.LastUsedAt, &k.ExpiresAt, &k.RevokedAt, &k.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &k)
	}
	return out, nil
}

func (s *Store) RevokeAPIKey(ctx context.Context, userID, id string) error {
	_, err := s.Pool.Exec(ctx, `UPDATE api_keys SET revoked_at=NOW() WHERE id=$1 AND user_id=$2`, id, userID)
	return err
}
