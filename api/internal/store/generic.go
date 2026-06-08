package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

// EntitySpec describes an entity for the generic CRUD layer used by OData.
type EntitySpec struct {
	Table      string
	Columns    []string // ordered column list returned by SELECT
	Insertable []string // columns accepted in INSERT/UPDATE (excludes id, created_at, updated_at)
	HasUpdated bool
	HasShortid bool
	// DefaultOrder is the ORDER BY clause used when the request has no
	// $orderby. Set it for tables that lack both updated_at and created_at
	// (e.g. profiles, which timestamps with started_at instead).
	DefaultOrder string
}

var EntitySpecs = map[string]EntitySpec{
	"templates": {
		Table: "templates",
		Columns: []string{"id", "shortid", "name", "folder_id", "content", "engine", "recipe", "helpers",
			"css", "page_size", "page_orientation", "page_margin",
			"data_shortid", "scripts", "chrome", "weasyprint", "docx", "xlsx", "pptx", "pdf_operations",
			"report_retention_days",
			"is_public", "owner_id", "read_perms", "edit_perms", "created_at", "updated_at"},
		Insertable: []string{"shortid", "name", "folder_id", "content", "engine", "recipe", "helpers",
			"css", "page_size", "page_orientation", "page_margin",
			"data_shortid", "scripts", "chrome", "weasyprint", "docx", "xlsx", "pptx", "pdf_operations",
			"report_retention_days",
			"is_public", "owner_id", "read_perms", "edit_perms"},
		HasUpdated: true, HasShortid: true,
	},
	"folders": {
		Table:      "folders",
		Columns:    []string{"id", "shortid", "name", "parent_id", "owner_id", "read_perms", "edit_perms", "created_at", "updated_at"},
		Insertable: []string{"shortid", "name", "parent_id", "owner_id", "read_perms", "edit_perms"},
		HasUpdated: true, HasShortid: true,
	},
	"assets": {
		Table:      "assets",
		Columns:    []string{"id", "shortid", "name", "folder_id", "blob_key", "inline_content", "mime_type", "size_bytes", "is_shared_helper", "shared_helpers_scope", "link", "owner_id", "created_at", "updated_at"},
		Insertable: []string{"shortid", "name", "folder_id", "blob_key", "inline_content", "mime_type", "size_bytes", "is_shared_helper", "shared_helpers_scope", "link", "owner_id"},
		HasUpdated: true, HasShortid: true,
	},
	"scripts": {
		Table:      "scripts",
		Columns:    []string{"id", "shortid", "name", "folder_id", "content", "scope", "is_global", "owner_id", "created_at", "updated_at"},
		Insertable: []string{"shortid", "name", "folder_id", "content", "scope", "is_global", "owner_id"},
		HasUpdated: true, HasShortid: true,
	},
	"data": {
		Table:      "data_items",
		Columns:    []string{"id", "shortid", "name", "folder_id", "data_json", "owner_id", "created_at", "updated_at"},
		Insertable: []string{"shortid", "name", "folder_id", "data_json", "owner_id"},
		HasUpdated: true, HasShortid: true,
	},
	"components": {
		Table:      "components",
		Columns:    []string{"id", "shortid", "name", "folder_id", "content", "engine", "helpers", "owner_id", "created_at", "updated_at"},
		Insertable: []string{"shortid", "name", "folder_id", "content", "engine", "helpers", "owner_id"},
		HasUpdated: true, HasShortid: true,
	},
	"reports": {
		Table:      "reports",
		Columns:    []string{"id", "shortid", "name", "template_shortid", "state", "error", "mime_type", "blob_key", "size_bytes", "is_public", "task_id", "owner_id", "created_at", "expires_at"},
		Insertable: []string{"shortid", "name", "template_shortid", "state", "error", "mime_type", "blob_key", "size_bytes", "is_public", "task_id", "owner_id", "expires_at"},
		HasUpdated: false, HasShortid: true,
	},
	"schedules": {
		Table:      "schedules",
		Columns:    []string{"id", "shortid", "name", "cron", "template_shortid", "enabled", "state", "next_run", "owner_id", "created_at", "updated_at"},
		Insertable: []string{"shortid", "name", "cron", "template_shortid", "enabled", "state", "next_run", "owner_id"},
		HasUpdated: true, HasShortid: true,
	},
	"profiles": {
		Table:      "profiles",
		Columns:    []string{"id", "shortid", "template_shortid", "state", "mode", "error", "blob_key", "timeout_ms", "started_at", "finished_at", "owner_id"},
		Insertable: []string{"shortid", "template_shortid", "state", "mode", "error", "blob_key", "timeout_ms", "started_at", "finished_at", "owner_id"},
		HasUpdated: false, HasShortid: true,
		// profiles has no created_at/updated_at — it timestamps with started_at.
		DefaultOrder: "started_at DESC",
	},
	"settings": {
		Table:      "settings",
		Columns:    []string{"id", "key", "value", "updated_at"},
		Insertable: []string{"key", "value"},
		HasUpdated: true,
	},
	"versions": {
		Table:      "versions",
		Columns:    []string{"id", "shortid", "message", "blob_key", "author_id", "created_at"},
		Insertable: []string{"shortid", "message", "blob_key", "author_id"},
		HasUpdated: false, HasShortid: true,
	},
	"tags": {
		Table:      "tags",
		Columns:    []string{"id", "shortid", "name", "color", "created_at"},
		Insertable: []string{"shortid", "name", "color"},
		HasUpdated: false, HasShortid: true,
	},
	// Users — exposed read-only via OData so the settings UI can list them.
	// `password_hash` is NEVER in Columns so it can never leak in a list/get
	// response. Writes go through dedicated handlers (register / change
	// password) that hash the secret server-side.
	"users": {
		Table:      "users",
		Columns:    []string{"id", "shortid", "username", "email", "is_admin", "read_all", "edit_all", "created_at", "updated_at"},
		Insertable: []string{"shortid", "username", "email", "is_admin", "read_all", "edit_all"},
		HasUpdated: true, HasShortid: true,
	},
}

// ListGeneric is used by the OData layer. It supports a basic WHERE expression
// (already translated to SQL by the OData parser) plus limit/offset/order.
func (s *Store) ListGeneric(ctx context.Context, spec EntitySpec, where string, args []any, orderBy string, limit, offset int) ([]map[string]any, error) {
	cols := strings.Join(spec.Columns, ", ")
	q := fmt.Sprintf(`SELECT %s FROM %s`, cols, spec.Table)
	if where != "" {
		q += " WHERE " + where
	}
	if orderBy != "" {
		q += " ORDER BY " + orderBy
	} else if spec.DefaultOrder != "" {
		q += " ORDER BY " + spec.DefaultOrder
	} else if spec.HasUpdated {
		q += " ORDER BY updated_at DESC"
	} else {
		q += " ORDER BY created_at DESC"
	}
	if limit > 0 {
		q += fmt.Sprintf(" LIMIT %d", limit)
	}
	if offset > 0 {
		q += fmt.Sprintf(" OFFSET %d", offset)
	}
	rows, err := s.Pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		vals, err := rows.Values()
		if err != nil {
			return nil, err
		}
		m := make(map[string]any, len(spec.Columns))
		for i, c := range spec.Columns {
			m[c] = normalizeJSON(vals[i])
		}
		out = append(out, m)
	}
	return out, nil
}

func (s *Store) CountGeneric(ctx context.Context, spec EntitySpec, where string, args []any) (int, error) {
	q := fmt.Sprintf(`SELECT COUNT(*) FROM %s`, spec.Table)
	if where != "" {
		q += " WHERE " + where
	}
	var n int
	err := s.Pool.QueryRow(ctx, q, args...).Scan(&n)
	return n, err
}

func (s *Store) GetGenericByShortid(ctx context.Context, spec EntitySpec, shortid string) (map[string]any, error) {
	res, err := s.ListGeneric(ctx, spec, "shortid = $1", []any{shortid}, "", 1, 0)
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, ErrNotFound
	}
	return res[0], nil
}

func (s *Store) InsertGeneric(ctx context.Context, spec EntitySpec, payload map[string]any) (map[string]any, error) {
	if spec.HasShortid {
		if _, ok := payload["shortid"]; !ok || payload["shortid"] == "" {
			payload["shortid"] = newShortid()
		}
	}
	cols := []string{}
	placeholders := []string{}
	args := []any{}
	i := 1
	for _, c := range spec.Insertable {
		if v, ok := payload[c]; ok {
			cols = append(cols, c)
			placeholders = append(placeholders, fmt.Sprintf("$%d", i))
			args = append(args, v)
			i++
		}
	}
	if len(cols) == 0 {
		return nil, errors.New("no insertable fields")
	}
	// JSONB columns need their slice/map values pre-encoded so pgx writes a
	// real JSON document, not an array of separate parameters.
	for k, a := range args {
		args[k] = encodeForSQL(a)
	}
	q := fmt.Sprintf(`INSERT INTO %s (%s) VALUES (%s) RETURNING %s`,
		spec.Table, strings.Join(cols, ","), strings.Join(placeholders, ","), strings.Join(spec.Columns, ","))
	row := s.Pool.QueryRow(ctx, q, args...)
	vals := make([]any, len(spec.Columns))
	ptrs := make([]any, len(spec.Columns))
	for i := range vals {
		ptrs[i] = &vals[i]
	}
	if err := row.Scan(ptrs...); err != nil {
		return nil, err
	}
	m := map[string]any{}
	for i, c := range spec.Columns {
		m[c] = normalizeJSON(vals[i])
	}
	return m, nil
}

func (s *Store) UpdateGeneric(ctx context.Context, spec EntitySpec, shortid string, payload map[string]any) (map[string]any, error) {
	sets := []string{}
	args := []any{}
	i := 1
	for _, c := range spec.Insertable {
		if v, ok := payload[c]; ok {
			sets = append(sets, fmt.Sprintf("%s = $%d", c, i))
			args = append(args, encodeForSQL(v))
			i++
		}
	}
	if spec.HasUpdated {
		sets = append(sets, "updated_at = NOW()")
	}
	if len(sets) == 0 {
		return s.GetGenericByShortid(ctx, spec, shortid)
	}
	args = append(args, shortid)
	q := fmt.Sprintf(`UPDATE %s SET %s WHERE shortid = $%d RETURNING %s`,
		spec.Table, strings.Join(sets, ", "), i, strings.Join(spec.Columns, ","))
	row := s.Pool.QueryRow(ctx, q, args...)
	vals := make([]any, len(spec.Columns))
	ptrs := make([]any, len(spec.Columns))
	for i := range vals {
		ptrs[i] = &vals[i]
	}
	if err := row.Scan(ptrs...); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	m := map[string]any{}
	for i, c := range spec.Columns {
		m[c] = normalizeJSON(vals[i])
	}
	return m, nil
}

func (s *Store) DeleteGeneric(ctx context.Context, spec EntitySpec, shortid string) error {
	_, err := s.Pool.Exec(ctx, fmt.Sprintf(`DELETE FROM %s WHERE shortid = $1`, spec.Table), shortid)
	return err
}

// ExpiredReports returns the shortid + blob_key of every report whose
// expires_at is in the past. Used by the retention sweeper to know which
// blobs to delete from SeaweedFS before removing the rows.
func (s *Store) ExpiredReports(ctx context.Context, limit int) ([]struct {
	Shortid string
	BlobKey string
}, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT shortid, COALESCE(blob_key, '')
		FROM reports
		WHERE expires_at IS NOT NULL AND expires_at < NOW()
		ORDER BY expires_at ASC
		LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []struct {
		Shortid string
		BlobKey string
	}
	for rows.Next() {
		var r struct {
			Shortid string
			BlobKey string
		}
		if err := rows.Scan(&r.Shortid, &r.BlobKey); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// DeleteReportsByShortid removes report rows in one statement.
func (s *Store) DeleteReportsByShortid(ctx context.Context, shortids []string) error {
	if len(shortids) == 0 {
		return nil
	}
	_, err := s.Pool.Exec(ctx, `DELETE FROM reports WHERE shortid = ANY($1)`, shortids)
	return err
}

// DeleteProfilesByShortid removes profile rows in one statement.
func (s *Store) DeleteProfilesByShortid(ctx context.Context, shortids []string) error {
	if len(shortids) == 0 {
		return nil
	}
	_, err := s.Pool.Exec(ctx, `DELETE FROM profiles WHERE shortid = ANY($1)`, shortids)
	return err
}

// BlobRef pairs a row's shortid with its blob_key so the caller can purge the
// stored file before deleting the row.
type BlobRef struct {
	Shortid string
	BlobKey string
}

func scanBlobRefs(rows pgx.Rows) ([]BlobRef, error) {
	defer rows.Close()
	var out []BlobRef
	for rows.Next() {
		var r BlobRef
		if err := rows.Scan(&r.Shortid, &r.BlobKey); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// OverflowReports returns every report beyond the newest `keep` (ordered by
// created_at DESC). Used to enforce a hard cap on stored report history.
func (s *Store) OverflowReports(ctx context.Context, keep int) ([]BlobRef, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT shortid, COALESCE(blob_key, '')
		FROM reports
		ORDER BY created_at DESC
		OFFSET $1`, keep)
	if err != nil {
		return nil, err
	}
	return scanBlobRefs(rows)
}

// OverflowProfiles returns every profile beyond the newest `keep` (ordered by
// started_at DESC). Used to enforce a hard cap on stored profile history.
func (s *Store) OverflowProfiles(ctx context.Context, keep int) ([]BlobRef, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT shortid, COALESCE(blob_key, '')
		FROM profiles
		ORDER BY started_at DESC NULLS LAST
		OFFSET $1`, keep)
	if err != nil {
		return nil, err
	}
	return scanBlobRefs(rows)
}

// encodeForSQL turns Go map/slice values into JSON bytes so pgx writes them
// as JSONB literals (rather than trying to encode each element separately).
// Strings and primitives are returned unchanged.
func encodeForSQL(v any) any {
	switch v.(type) {
	case map[string]any, []any, []map[string]any:
		if b, err := json.Marshal(v); err == nil {
			return b
		}
	}
	return v
}

func normalizeJSON(v any) any {
	switch x := v.(type) {
	case []byte:
		// Try to keep JSON columns as objects.
		var raw json.RawMessage = x
		return raw
	case [16]byte:
		// Postgres UUID arrives as a 16-byte array from pgx — render as the
		// canonical 8-4-4-4-12 hex string clients expect.
		return formatUUID(x)
	default:
		return v
	}
}

func formatUUID(b [16]byte) string {
	const hex = "0123456789abcdef"
	out := make([]byte, 36)
	pos := 0
	for i, by := range b {
		out[pos] = hex[by>>4]
		out[pos+1] = hex[by&0x0f]
		pos += 2
		if i == 3 || i == 5 || i == 7 || i == 9 {
			out[pos] = '-'
			pos++
		}
	}
	return string(out)
}
