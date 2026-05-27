package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/cloudreport/api/internal/models"
	"github.com/jackc/pgx/v5"
	gonanoid "github.com/matoous/go-nanoid/v2"
)

var ErrNotFound = errors.New("not found")

func newShortid() string {
	id, _ := gonanoid.New(10)
	return id
}

// NewShortid is the exported entry point for callers outside the store
// package (e.g. the renderer persisting reports) that need a fresh shortid.
func NewShortid() string { return newShortid() }

// ----- TEMPLATES -----

func (s *Store) CreateTemplate(ctx context.Context, t *models.Template) error {
	if t.Shortid == "" {
		t.Shortid = newShortid()
	}
	if t.Engine == "" {
		t.Engine = "handlebars"
	}
	if t.Recipe == "" {
		t.Recipe = "html"
	}
	if len(t.Scripts) == 0 {
		t.Scripts = json.RawMessage("[]")
	}
	if len(t.PdfOperations) == 0 {
		t.PdfOperations = json.RawMessage("[]")
	}
	row := s.Pool.QueryRow(ctx, `
		INSERT INTO templates
		(shortid, name, folder_id, content, engine, recipe, helpers, data_shortid,
		 scripts, chrome, weasyprint, docx, xlsx, pptx, pdf_operations,
		 is_public, owner_id, read_perms, edit_perms)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)
		RETURNING id, created_at, updated_at`,
		t.Shortid, t.Name, t.FolderID, t.Content, t.Engine, t.Recipe, t.Helpers, t.DataShortid,
		t.Scripts, nullableJSON(t.Chrome), nullableJSON(t.WeasyPrint), nullableJSON(t.Docx),
		nullableJSON(t.Xlsx), nullableJSON(t.Pptx), t.PdfOperations,
		t.IsPublic, t.OwnerID, t.ReadPerms, t.EditPerms,
	)
	return row.Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
}

func nullableJSON(j json.RawMessage) interface{} {
	if len(j) == 0 {
		return nil
	}
	return j
}

func (s *Store) GetTemplateByShortid(ctx context.Context, shortid string) (*models.Template, error) {
	return s.queryTemplate(ctx, `WHERE shortid = $1`, shortid)
}

func (s *Store) GetTemplateByName(ctx context.Context, name string) (*models.Template, error) {
	return s.queryTemplate(ctx, `WHERE name = $1`, name)
}

func (s *Store) GetTemplateByID(ctx context.Context, id string) (*models.Template, error) {
	return s.queryTemplate(ctx, `WHERE id = $1`, id)
}

func (s *Store) queryTemplate(ctx context.Context, where, arg string) (*models.Template, error) {
	var t models.Template
	var chrome, weasy, docx, xlsx, pptx []byte
	err := s.Pool.QueryRow(ctx, `
		SELECT id, shortid, name, folder_id, content, engine, recipe, helpers, css,
		       page_size, page_orientation, page_margin, data_shortid,
		       scripts, chrome, weasyprint, docx, xlsx, pptx, pdf_operations,
		       report_retention_days,
		       is_public, owner_id, read_perms, edit_perms, created_at, updated_at
		FROM templates `+where, arg).
		Scan(&t.ID, &t.Shortid, &t.Name, &t.FolderID, &t.Content, &t.Engine, &t.Recipe, &t.Helpers, &t.CSS,
			&t.PageSize, &t.PageOrientation, &t.PageMargin, &t.DataShortid,
			&t.Scripts, &chrome, &weasy, &docx, &xlsx, &pptx, &t.PdfOperations,
			&t.ReportRetentionDays,
			&t.IsPublic, &t.OwnerID, &t.ReadPerms, &t.EditPerms, &t.CreatedAt, &t.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	t.Chrome = chrome
	t.WeasyPrint = weasy
	t.Docx = docx
	t.Xlsx = xlsx
	t.Pptx = pptx
	return &t, nil
}

func (s *Store) ListTemplates(ctx context.Context, ownerID *string, limit, offset int) ([]*models.Template, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id, shortid, name, folder_id, content, engine, recipe, helpers, data_shortid,
		       scripts, chrome, weasyprint, docx, xlsx, pptx, pdf_operations,
		       is_public, owner_id, read_perms, edit_perms, created_at, updated_at
		FROM templates
		WHERE ($1::uuid IS NULL OR owner_id = $1)
		ORDER BY updated_at DESC LIMIT $2 OFFSET $3`, ownerID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*models.Template
	for rows.Next() {
		var t models.Template
		var chrome, weasy, docx, xlsx, pptx []byte
		if err := rows.Scan(&t.ID, &t.Shortid, &t.Name, &t.FolderID, &t.Content, &t.Engine, &t.Recipe, &t.Helpers, &t.DataShortid,
			&t.Scripts, &chrome, &weasy, &docx, &xlsx, &pptx, &t.PdfOperations,
			&t.IsPublic, &t.OwnerID, &t.ReadPerms, &t.EditPerms, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		t.Chrome, t.WeasyPrint, t.Docx, t.Xlsx, t.Pptx = chrome, weasy, docx, xlsx, pptx
		out = append(out, &t)
	}
	return out, nil
}

func (s *Store) UpdateTemplate(ctx context.Context, t *models.Template) error {
	t.UpdatedAt = time.Now()
	_, err := s.Pool.Exec(ctx, `
		UPDATE templates SET
		  name=$2, folder_id=$3, content=$4, engine=$5, recipe=$6, helpers=$7, data_shortid=$8,
		  scripts=$9, chrome=$10, weasyprint=$11, docx=$12, xlsx=$13, pptx=$14, pdf_operations=$15,
		  is_public=$16, read_perms=$17, edit_perms=$18, updated_at=NOW()
		WHERE shortid=$1`,
		t.Shortid, t.Name, t.FolderID, t.Content, t.Engine, t.Recipe, t.Helpers, t.DataShortid,
		t.Scripts, nullableJSON(t.Chrome), nullableJSON(t.WeasyPrint), nullableJSON(t.Docx),
		nullableJSON(t.Xlsx), nullableJSON(t.Pptx), t.PdfOperations,
		t.IsPublic, t.ReadPerms, t.EditPerms,
	)
	return err
}

func (s *Store) DeleteTemplate(ctx context.Context, shortid string) error {
	_, err := s.Pool.Exec(ctx, `DELETE FROM templates WHERE shortid=$1`, shortid)
	return err
}
