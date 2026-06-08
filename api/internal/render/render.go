// Package render is the rendering orchestrator. It resolves the template,
// evaluates data, runs the engine, dispatches to a recipe (in-process for Go
// recipes, via Redis for Python workers), and applies PDF post-processing.
package render

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	pdfapi "github.com/pdfcpu/pdfcpu/pkg/api"

	"github.com/cloudreport/api/internal/blob"
	"github.com/cloudreport/api/internal/config"
	"github.com/cloudreport/api/internal/models"
	"github.com/cloudreport/api/internal/queue"
	"github.com/cloudreport/api/internal/render/engine"
	"github.com/cloudreport/api/internal/render/recipe"
	"github.com/cloudreport/api/internal/render/sandbox"
	"github.com/cloudreport/api/internal/store"
)

type Renderer struct {
	cfg     *config.Config
	db      *store.Store
	blob    *blob.Store
	queue   *queue.Queue
	sandbox *sandbox.Sandbox

	recipes map[string]recipe.Recipe
}

var componentRegex = regexp.MustCompile(`\{\{\s*>\s*([A-Za-z0-9_\-]+)\s*\}\}`)

// assetRegex matches jsreport's {#asset name [@encoding=utf8|base64|dataURI|link|string]}
// syntax. The inner capture is the raw param string, parsed by evaluateAssets.
var assetRegex = regexp.MustCompile(`\{#asset ([^{}]{1,500})\}`)

func New(cfg *config.Config, db *store.Store, bs *blob.Store, q *queue.Queue) *Renderer {
	r := &Renderer{
		cfg: cfg, db: db, blob: bs, queue: q,
		sandbox: sandbox.New(5 * time.Second),
	}
	r.recipes = map[string]recipe.Recipe{
		"html":       recipe.HTML{},
		"text":       recipe.Text{},
		"static-pdf": recipe.StaticPDF{Blob: bs},
		"xlsx":       recipe.Xlsx{},
		// recipes that delegate to Python workers:
		"chrome-pdf":   recipe.WorkerProxy{Recipe: "chrome-pdf", Mime: "application/pdf", Queue: q},
		"weasyprint":   recipe.WorkerProxy{Recipe: "weasyprint", Mime: "application/pdf", Queue: q},
		"docx":         recipe.WorkerProxy{Recipe: "docx", Mime: "application/vnd.openxmlformats-officedocument.wordprocessingml.document", Queue: q, NeedsTemplateAsset: true},
		"pptx":         recipe.WorkerProxy{Recipe: "pptx", Mime: "application/vnd.openxmlformats-officedocument.presentationml.presentation", Queue: q, NeedsTemplateAsset: true},
		"html-to-xlsx": recipe.WorkerProxy{Recipe: "html-to-xlsx", Mime: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", Queue: q},
	}
	return r
}

func (r *Renderer) ListRecipes() []string {
	out := make([]string, 0, len(r.recipes))
	for k := range r.recipes {
		out = append(out, k)
	}
	return out
}

func (r *Renderer) ListEngines() []string {
	return []string{"handlebars", "none"}
}

// Request mirrors jsreport's POST /api/report body.
type Request struct {
	Template struct {
		Shortid string          `json:"shortid,omitempty"`
		Name    string          `json:"name,omitempty"`
		Content string          `json:"content,omitempty"`
		Engine  string          `json:"engine,omitempty"`
		Recipe  string          `json:"recipe,omitempty"`
		Helpers string          `json:"helpers,omitempty"`
		Chrome      json.RawMessage `json:"chrome,omitempty"`
		WeasyPrint  json.RawMessage `json:"weasyprint,omitempty"`
		Docx        json.RawMessage `json:"docx,omitempty"`
		Xlsx        json.RawMessage `json:"xlsx,omitempty"`
		Pptx        json.RawMessage `json:"pptx,omitempty"`
		PdfOperations json.RawMessage `json:"pdfOperations,omitempty"`
	} `json:"template"`
	Data    json.RawMessage `json:"data,omitempty"`
	Options json.RawMessage `json:"options,omitempty"`
	Context json.RawMessage `json:"context,omitempty"`

	// pageOverride is set internally by per-page stamp rendering so the
	// header/footer template inherits the parent document's page size.
	// Not exposed in the public JSON body (unexported → invisible to
	// json.Unmarshal).
	pageOverride *pageOverride
}

type pageOverride struct {
	size, orientation, margin string
}

// Result is what the renderer produces.
type Result struct {
	Content   []byte
	MimeType  string
	FileName  string
	ProfileID string
}

func (r *Renderer) Render(ctx context.Context, req *Request, user *models.User) (*Result, error) {
	tpl, err := r.resolveTemplate(ctx, req)
	if err != nil {
		return nil, err
	}

	// Profile (record the run).
	prof, err := r.startProfile(ctx, tpl, user)
	if err != nil {
		// non-fatal
		prof = nil
	}

	eng, err := engine.ByName(tpl.Engine)
	if err != nil {
		r.finishProfile(ctx, prof, "error", err)
		return nil, err
	}

	var data any = map[string]any{}
	if len(req.Data) > 0 {
		if err := json.Unmarshal(req.Data, &data); err != nil {
			// Allow string input
			data = string(req.Data)
		}
	}

	// Compile user helpers (if any) in the sandbox. Failure is non-fatal: the
	// template may still render without them. The returned HelperSet keeps a
	// goja VM alive, so we always defer-close it to avoid leaks.
	var helpers map[string]sandbox.HelperFn
	if tpl.Helpers != "" {
		if hs, herr := r.sandbox.RunHelpers(ctx, tpl.Helpers); herr == nil && hs != nil {
			defer hs.Close()
			helpers = hs.Helpers
		}
	}

	// Run beforeRender scripts. Scripts are linked via tpl.Scripts (JSON array
	// of {shortid, content}). For inline content we run it; for shortids we
	// resolve from the DB.
	mutatedData, err := r.runBeforeRenderScripts(ctx, tpl, req.Data)
	if err != nil {
		r.finishProfile(ctx, prof, "error", err)
		return nil, fmt.Errorf("script error: %w", err)
	}
	if mutatedData != nil {
		data = mutatedData
	}

	// Strip any `@page { size: ... }` rules from user content + CSS when
	// the Página tab has a PageSize set. Chromium honours CSS @page size
	// in some scenarios even with preferCSSPageSize=false, which silently
	// overrides the user's choice in the Página tab.
	if tpl.PageSize != "" {
		tpl.Content = stripAtPageSize(tpl.Content)
		tpl.CSS = stripAtPageSize(tpl.CSS)
	}

	// Inline-expand components referenced as {{> componentName}} (partials).
	// We pull them eagerly so raymond can register them before parsing.
	contentToRender, err := r.expandComponents(ctx, tpl.Content)
	if err != nil {
		r.finishProfile(ctx, prof, "error", err)
		return nil, err
	}

	// Resolve {#asset ...} references in template content / CSS / helpers
	// BEFORE the engine runs, mirroring jsreport-assets. This lets the user
	// inline CSS, JS and base64 images at template authoring time.
	contentToRender, err = r.evaluateAssets(ctx, contentToRender)
	if err != nil {
		r.finishProfile(ctx, prof, "error", err)
		return nil, fmt.Errorf("assets (content): %w", err)
	}
	cssResolved, err := r.evaluateAssets(ctx, tpl.CSS)
	if err != nil {
		r.finishProfile(ctx, prof, "error", err)
		return nil, fmt.Errorf("assets (css): %w", err)
	}

	rendered, err := eng.Render(contentToRender, data, helpers)
	if err != nil {
		r.finishProfile(ctx, prof, "error", err)
		return nil, fmt.Errorf("engine: %w", err)
	}

	// Second pass on the rendered output — handlebars may have produced more
	// {#asset ...} tokens dynamically.
	rendered, err = r.evaluateAssets(ctx, rendered)
	if err != nil {
		r.finishProfile(ctx, prof, "error", err)
		return nil, fmt.Errorf("assets (rendered): %w", err)
	}

	// Inject the template's CSS + page settings into the rendered HTML. For
	// PDF recipes the @page rules control physical size & margins; for HTML
	// recipes they're harmless. We only do this when the rendered output
	// looks like HTML so non-HTML templates (xlsx JSON, text) aren't touched.
	if isHTMLOutput(tpl.Recipe, rendered) {
		rendered = injectStyles(rendered, cssResolved, tpl.PageSize, tpl.PageOrientation, tpl.PageMargin)
	}

	rec, ok := r.recipes[tpl.Recipe]
	if !ok {
		err := fmt.Errorf("unknown recipe: %s", tpl.Recipe)
		r.finishProfile(ctx, prof, "error", err)
		return nil, err
	}

	templateOpts := pickTemplateOptions(tpl)
	// For office recipes the template binary lives in the assets table; the
	// worker needs the SeaweedFS object key, not the asset shortid. Resolve
	// it here so each recipe stays oblivious to the asset table.
	if err := r.resolveTemplateAssets(ctx, tpl.Recipe, templateOpts); err != nil {
		r.finishProfile(ctx, prof, "error", err)
		return nil, err
	}
	// For chrome-pdf: when the user picks "From template" for header/footer,
	// the option carries headerTemplateShortid / footerTemplateShortid. We
	// render those templates here so the worker receives plain HTML.
	if err := r.resolveHeaderFooterTemplates(ctx, tpl.Recipe, templateOpts, req.Data); err != nil {
		r.finishProfile(ctx, prof, "error", err)
		return nil, err
	}

	rctx := &recipe.Context{
		Ctx:          ctx,
		HTML:         rendered,
		Data:         req.Data,
		TemplateOpts: templateOpts,
		Timeout:      time.Duration(r.cfg.JobTimeoutSeconds) * time.Second,
	}

	out, err := rec.Execute(rctx)
	if err != nil {
		r.finishProfile(ctx, prof, "error", err)
		return nil, fmt.Errorf("recipe %s: %w", tpl.Recipe, err)
	}

	// Worker proxies return a sentinel "s3://<key>" instead of inline bytes.
	if strings.HasPrefix(string(out.Content), "s3://") {
		key := strings.TrimPrefix(string(out.Content), "s3://")
		data, ct, err := r.blob.Get(ctx, key)
		if err != nil {
			r.finishProfile(ctx, prof, "error", err)
			return nil, fmt.Errorf("fetch worker output: %w", err)
		}
		out.Content = data
		if out.MimeType == "" && ct != "" {
			out.MimeType = ct
		}
	}

	// PDF post-processing — only if the recipe output is PDF.
	if len(tpl.PdfOperations) > 0 && strings.HasPrefix(out.MimeType, "application/pdf") {
		// Count pages of the main PDF so we can support jsreport-style
		// {{$pdf.pageNumber}} / {{$pdf.pages.length}} substitution in stamp
		// templates (rendered once per page with those vars injected).
		totalPages, _ := pdfapi.PageCount(bytes.NewReader(out.Content), nil)
		parentOverride := &pageOverride{
			size:        tpl.PageSize,
			orientation: tpl.PageOrientation,
			margin:      "0", // stamps always render edge-to-edge so position:fixed inside the HTML lands on the page edge
		}
		resolved, rerr := r.resolveAppendTemplates(ctx, tpl.PdfOperations, req.Data, totalPages, parentOverride)
		if rerr != nil {
			r.finishProfile(ctx, prof, "error", rerr)
			return nil, fmt.Errorf("appendTemplate: %w", rerr)
		}
		processed, perr := recipe.ApplyPdfOperations(out.Content, resolved)
		if perr != nil {
			r.finishProfile(ctx, prof, "error", perr)
			return nil, fmt.Errorf("pdfOperations: %w", perr)
		}
		out.Content = processed
	}

	r.finishProfile(ctx, prof, "success", nil)

	result := &Result{
		Content:  out.Content,
		MimeType: out.MimeType,
		FileName: out.FileName,
	}
	if prof != nil {
		result.ProfileID = prof.Shortid
	}

	// Persist the rendered file as a `report` so it shows up in history and
	// can be re-downloaded. Stored in SeaweedFS with an expiry derived from
	// the template's retention policy; the sweeper deletes it after that.
	// Done on a detached context so a client disconnect doesn't abort the
	// save, and best-effort (a storage hiccup must not fail the render).
	r.persistReport(tpl, out, user)

	return result, nil
}

// persistReport stores the rendered output in the blob store and inserts a
// `reports` row with an expires_at computed from the template's retention.
// Retention 0 = keep forever (expires_at stays NULL).
func (r *Renderer) persistReport(tpl *models.Template, out *recipe.Result, user *models.User) {
	if r.blob == nil || len(out.Content) == 0 {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		shortid := store.NewShortid()
		key, err := r.blob.Put(ctx, "reports", shortid, out.Content, out.MimeType)
		if err != nil {
			return // best-effort
		}
		payload := map[string]any{
			"shortid":          shortid,
			"name":             out.FileName,
			"template_shortid": tpl.Shortid,
			"state":            "success",
			"mime_type":        out.MimeType,
			"blob_key":         key,
			"size_bytes":       int64(len(out.Content)),
		}
		if user != nil {
			payload["owner_id"] = user.ID
		}
		// retention <= 0 means keep forever → leave expires_at NULL.
		if tpl.ReportRetentionDays > 0 {
			payload["expires_at"] = time.Now().Add(time.Duration(tpl.ReportRetentionDays) * 24 * time.Hour)
		}
		_, _ = r.db.InsertGeneric(ctx, store.EntitySpecs["reports"], payload)
		// Enforce the hard cap so history never grows past MaxHistoryRows.
		r.capReports(ctx)
	}()
}

// MaxHistoryRows is the hard cap on how many reports / profiles are kept.
// Once exceeded, the oldest rows (and their blobs) are deleted so only the
// newest MaxHistoryRows remain.
const MaxHistoryRows = 200

// capReports trims the `reports` table to the newest MaxHistoryRows rows,
// purging each deleted row's blob first. Best-effort.
func (r *Renderer) capReports(ctx context.Context) {
	over, err := r.db.OverflowReports(ctx, MaxHistoryRows)
	if err != nil || len(over) == 0 {
		return
	}
	shortids := make([]string, 0, len(over))
	for _, e := range over {
		if e.BlobKey != "" && r.blob != nil {
			_ = r.blob.Delete(ctx, e.BlobKey)
		}
		shortids = append(shortids, e.Shortid)
	}
	_ = r.db.DeleteReportsByShortid(ctx, shortids)
}

// capProfiles trims the `profiles` table to the newest MaxHistoryRows rows.
func (r *Renderer) capProfiles(ctx context.Context) {
	over, err := r.db.OverflowProfiles(ctx, MaxHistoryRows)
	if err != nil || len(over) == 0 {
		return
	}
	shortids := make([]string, 0, len(over))
	for _, e := range over {
		if e.BlobKey != "" && r.blob != nil {
			_ = r.blob.Delete(ctx, e.BlobKey)
		}
		shortids = append(shortids, e.Shortid)
	}
	_ = r.db.DeleteProfilesByShortid(ctx, shortids)
}

// EnforceHistoryCap trims both reports and profiles to MaxHistoryRows. Invoked
// by the cleanup ticker (boot + hourly) as a backstop to the per-insert caps.
func (r *Renderer) EnforceHistoryCap(ctx context.Context) {
	r.capReports(ctx)
	r.capProfiles(ctx)
}

// SweepExpiredReports deletes reports whose retention window has elapsed:
// first the SeaweedFS blob, then the DB row. Returns how many were removed.
// Safe to call repeatedly; the cleanup ticker invokes it on a schedule.
func (r *Renderer) SweepExpiredReports(ctx context.Context) (int, error) {
	expired, err := r.db.ExpiredReports(ctx, 500)
	if err != nil {
		return 0, err
	}
	if len(expired) == 0 {
		return 0, nil
	}
	shortids := make([]string, 0, len(expired))
	for _, e := range expired {
		if e.BlobKey != "" && r.blob != nil {
			_ = r.blob.Delete(ctx, e.BlobKey) // best-effort; row removal proceeds regardless
		}
		shortids = append(shortids, e.Shortid)
	}
	if err := r.db.DeleteReportsByShortid(ctx, shortids); err != nil {
		return 0, err
	}
	return len(shortids), nil
}

func (r *Renderer) resolveTemplate(ctx context.Context, req *Request) (*models.Template, error) {
	if req.Template.Shortid != "" {
		t, err := r.db.GetTemplateByShortid(ctx, req.Template.Shortid)
		if err == nil {
			overlay(t, req)
			return t, nil
		}
		if !errors.Is(err, store.ErrNotFound) {
			return nil, err
		}
	}
	if req.Template.Name != "" {
		t, err := r.db.GetTemplateByName(ctx, req.Template.Name)
		if err == nil {
			overlay(t, req)
			return t, nil
		}
		if !errors.Is(err, store.ErrNotFound) {
			return nil, err
		}
	}
	if req.Template.Content == "" && req.Template.Recipe == "" {
		return nil, errors.New("no template (shortid/name/content) provided")
	}
	// Inline template (not persisted)
	return &models.Template{
		Content:    req.Template.Content,
		Engine:     defaultStr(req.Template.Engine, "handlebars"),
		Recipe:     defaultStr(req.Template.Recipe, "html"),
		Helpers:    req.Template.Helpers,
		Chrome:     req.Template.Chrome,
		WeasyPrint: req.Template.WeasyPrint,
		Docx:       req.Template.Docx,
		Xlsx:       req.Template.Xlsx,
		Pptx:       req.Template.Pptx,
		PdfOperations: defaultJSON(req.Template.PdfOperations, "[]"),
	}, nil
}

func overlay(t *models.Template, req *Request) {
	if req.Template.Engine != "" {
		t.Engine = req.Template.Engine
	}
	if req.Template.Recipe != "" {
		t.Recipe = req.Template.Recipe
	}
	if req.Template.Content != "" {
		t.Content = req.Template.Content
	}
}

func defaultStr(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func defaultJSON(v json.RawMessage, def string) json.RawMessage {
	if len(v) == 0 {
		return json.RawMessage(def)
	}
	return v
}

// isHTMLOutput is a cheap heuristic: HTML-producing recipes get style injection,
// non-HTML ones (xlsx, text, static-pdf) don't.
func isHTMLOutput(recipe, rendered string) bool {
	switch recipe {
	case "html", "weasyprint", "chrome-pdf", "html-to-xlsx":
		return true
	}
	// Fallback sniff: starts with <! or contains <html
	r := strings.TrimSpace(rendered)
	return strings.HasPrefix(r, "<") && (strings.Contains(r, "<html") || strings.Contains(r, "<!DOCTYPE") || strings.Contains(r, "<body"))
}

// injectStyles weaves the template's CSS + page settings into the HTML head.
// If the HTML already has a <head>, we append a <style> block; otherwise we
// wrap the rendered HTML in a minimal HTML5 envelope so WeasyPrint always has
// proper @page rules.
func injectStyles(html, css, size, orientation, margin string) string {
	if size == "" {
		size = "A4"
	}
	if margin == "" {
		margin = "1cm"
	}
	pageRule := "@page { size: " + size
	if orientation == "landscape" {
		pageRule += " landscape"
	}
	pageRule += "; margin: " + margin + "; }"

	styleBlock := "<style>" + pageRule + "\n" + css + "</style>"
	if strings.Contains(html, "</head>") {
		return strings.Replace(html, "</head>", styleBlock+"</head>", 1)
	}
	// No head — wrap the content.
	return "<!DOCTYPE html><html><head><meta charset=\"utf-8\">" + styleBlock + "</head><body>" + html + "</body></html>"
}

// resolveAppendTemplates walks the pdfOperations array and rewrites every op
// that references a `templateShortid` into a backend-native op carrying the
// rendered PDF in base64. Mirrors jsreport-pdf-utils' canonical model:
//
//	{ type: 'append',  templateShortid }                       → merge (concat at end)
//	{ type: 'prepend', templateShortid }                       → prepend
//	{ type: 'merge',   templateShortid, renderForEveryPage } → stamp (overlay every page)
//	{ type: 'merge',   templateShortid }                       → merge (concat – fallback)
//
// Legacy aliases used by older saved data are accepted too:
//
//	appendTemplate → append, prependTemplate → prepend, stampTemplate → merge+stamp
//
// When the stamp template contains jsreport-style {{$pdf.pageNumber}} or
// {{$pdf.pages.length}} placeholders AND we know the total page count of the
// main document, the op is expanded into N stamp ops — one per page — each
// rendered with the appropriate $pdf vars injected into the data. This
// mirrors jsreport-pdf-utils' per-page rendering for header/footer use-cases.
//
// Keeps the pdfutils package free of any knowledge of the render pipeline.
func (r *Renderer) resolveAppendTemplates(ctx context.Context, ops json.RawMessage, data json.RawMessage, totalPages int, parentOverride *pageOverride) (json.RawMessage, error) {
	if len(ops) == 0 {
		return ops, nil
	}
	var list []map[string]any
	if err := json.Unmarshal(ops, &list); err != nil {
		return ops, nil
	}
	newList := make([]map[string]any, 0, len(list))
	changed := false
	for _, op := range list {
		t, _ := op["type"].(string)
		shortid, _ := op["templateShortid"].(string)
		if shortid == "" {
			// Nothing to render; leave the op alone (might be a static merge
			// with a pre-uploaded pdfBase64).
			newList = append(newList, op)
			continue
		}

		// Decide target backend op type using both the canonical jsreport
		// type and any flags it carries.
		var nextType string
		switch t {
		case "append", "appendTemplate":
			nextType = "merge"
		case "prepend", "prependTemplate":
			nextType = "prepend"
		case "stampTemplate":
			nextType = "stamp"
		case "merge":
			// In jsreport, `merge` with a templateShortid is a stamp/overlay
			// unless the user explicitly chose mergeWholeDocument (which we
			// approximate as a concat). renderForEveryPage further confirms
			// the stamp intent for header/footer.
			if rfe, _ := op["renderForEveryPage"].(bool); rfe {
				nextType = "stamp"
			} else if mwd, _ := op["mergeWholeDocument"].(bool); mwd {
				nextType = "merge"
			} else {
				// Default for a templated merge is the header/footer use-case.
				nextType = "stamp"
			}
		default:
			newList = append(newList, op)
			continue
		}

		// jsreport-style {{$pdf.*}} per-page expansion. Only applies to stamp
		// ops where we know the page count. We peek at the template content
		// once to decide whether per-page rendering is needed; for templates
		// that don't reference $pdf, a single render suffices (matches the
		// original behaviour).
		needsPerPage := false
		if nextType == "stamp" && totalPages > 0 {
			if tpl, err := r.db.GetTemplateByShortid(ctx, shortid); err == nil && tpl != nil {
				if templateReferencesPdfVars(tpl.Content) || templateReferencesPdfVars(tpl.CSS) {
					needsPerPage = true
				}
			}
		}

		if needsPerPage {
			pageOps, err := r.renderPerPageStamps(ctx, shortid, data, totalPages, parentOverride)
			if err != nil {
				return nil, fmt.Errorf("template %q: %w", shortid, err)
			}
			newList = append(newList, pageOps...)
			changed = true
			continue
		}

		// Default path: single render, same data, no profiling, no further
		// pdfOps to avoid infinite recursion if a template references itself.
		// Stamps still inherit the parent's page size so the overlay
		// dimensions match the target document.
		req := &Request{}
		req.Template.Shortid = shortid
		req.Data = data
		if nextType == "stamp" {
			req.pageOverride = parentOverride
		}
		sub, err := r.renderLeaf(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("template %q: %w", shortid, err)
		}
		if !strings.HasPrefix(sub.MimeType, "application/pdf") {
			return nil, fmt.Errorf("template %q did not produce PDF (got %s)", shortid, sub.MimeType)
		}
		b64 := base64.StdEncoding.EncodeToString(sub.Content)
		next := map[string]any{
			"type":      nextType,
			"pdfBase64": b64,
		}
		if pages, ok := op["pages"].(string); ok && pages != "" {
			next["pages"] = pages
		}
		newList = append(newList, next)
		changed = true
	}
	if !changed {
		return ops, nil
	}
	return json.Marshal(newList)
}

// templateReferencesPdfVars reports whether the template content uses any of
// the jsreport-style {{$pdf.*}} placeholders. We check the raw template
// instead of waiting for the engine to error, because raymond doesn't
// recognise `$pdf` as a variable name and would silently emit empty strings.
func templateReferencesPdfVars(content string) bool {
	if content == "" {
		return false
	}
	return strings.Contains(content, "{{$pdf.pageNumber}}") ||
		strings.Contains(content, "{{$pdf.pages.length}}") ||
		strings.Contains(content, "{{$pdf.pages.pageNumber}}")
}

// renderPerPageStamps renders the stamp template once per page with $pdf vars
// injected, returning N pdfutils stamp ops targeted to each page individually.
// parentOverride propagates the main document's page size into each per-page
// render so the overlay PDF matches the target page dimensions exactly.
func (r *Renderer) renderPerPageStamps(ctx context.Context, shortid string, data json.RawMessage, totalPages int, parentOverride *pageOverride) ([]map[string]any, error) {
	out := make([]map[string]any, 0, totalPages)
	for i := 1; i <= totalPages; i++ {
		perPageData, err := injectPdfVarsIntoData(data, i, totalPages)
		if err != nil {
			return nil, err
		}
		req := &Request{}
		req.Template.Shortid = shortid
		req.Data = perPageData
		req.pageOverride = parentOverride
		sub, err := r.renderLeaf(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("render page %d: %w", i, err)
		}
		if !strings.HasPrefix(sub.MimeType, "application/pdf") {
			return nil, fmt.Errorf("per-page stamp must produce PDF (got %s)", sub.MimeType)
		}
		out = append(out, map[string]any{
			"type":      "stamp",
			"pdfBase64": base64.StdEncoding.EncodeToString(sub.Content),
			"pages":     fmt.Sprintf("%d", i),
		})
	}
	return out, nil
}

// injectPdfVarsIntoData merges {$pdf: {pageNumber, pages: {length, pageNumber}}}
// into the user data so per-page stamp templates can substitute them via the
// preprocessor in renderLeaf.
func injectPdfVarsIntoData(data json.RawMessage, pageNum, totalPages int) (json.RawMessage, error) {
	m := map[string]any{}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &m); err != nil {
			// Data wasn't an object — preserve under a key so we don't lose it.
			m = map[string]any{}
		}
	}
	m["$pdf"] = map[string]any{
		"pageNumber": pageNum,
		"pages": map[string]any{
			"length":     totalPages,
			"pageNumber": pageNum,
		},
	}
	return json.Marshal(m)
}

// substitutePdfVars replaces jsreport-style {{$pdf.*}} placeholders in a raw
// template string with concrete values. Called before the engine sees the
// content so raymond never has to interpret `$pdf` as a variable name.
func substitutePdfVars(content string, pageNum, totalPages int) string {
	if content == "" {
		return content
	}
	pn := fmt.Sprintf("%d", pageNum)
	tp := fmt.Sprintf("%d", totalPages)
	content = strings.ReplaceAll(content, "{{$pdf.pageNumber}}", pn)
	content = strings.ReplaceAll(content, "{{$pdf.pages.pageNumber}}", pn)
	content = strings.ReplaceAll(content, "{{$pdf.pages.length}}", tp)
	return content
}

// asInt coerces an arbitrary JSON-decoded value into an int. JSON numbers
// arrive as float64 from encoding/json by default; the pdf-vars map carries
// integers semantically, so we cast appropriately.
func asInt(v any) int {
	switch x := v.(type) {
	case int:
		return x
	case int64:
		return int(x)
	case float64:
		return int(x)
	case json.Number:
		i, _ := x.Int64()
		return int(i)
	}
	return 0
}

// renderLeaf runs the pipeline but skips pdfOperations (to avoid recursion).
// Used by appendTemplate resolution.
func (r *Renderer) renderLeaf(ctx context.Context, req *Request) (*recipe.Result, error) {
	tpl, err := r.resolveTemplate(ctx, req)
	if err != nil {
		return nil, err
	}
	tpl.PdfOperations = nil // hard guard against recursion

	// Per-page stamp rendering inherits the parent's page size so the
	// overlay PDF always matches the target page dimensions. We mutate the
	// resolved template's fields BEFORE option-picking so chrome.format,
	// chrome.landscape and chrome.margin all derive from the override.
	if req.pageOverride != nil {
		if req.pageOverride.size != "" {
			tpl.PageSize = req.pageOverride.size
		}
		if req.pageOverride.orientation != "" {
			tpl.PageOrientation = req.pageOverride.orientation
		}
		if req.pageOverride.margin != "" {
			tpl.PageMargin = req.pageOverride.margin
		}
	}

	// Strip any `@page { size: ... }` rules from the user-supplied content
	// and CSS when the template-level PageSize is set. Chromium honours
	// CSS @page size even with preferCSSPageSize=false in some scenarios,
	// causing the rendered PDF to ignore the format the user picked in the
	// Página tab. We remove only the `size` descriptor — `margin` and other
	// @page rules survive.
	if tpl.PageSize != "" {
		tpl.Content = stripAtPageSize(tpl.Content)
		tpl.CSS = stripAtPageSize(tpl.CSS)
	}

	eng, err := engine.ByName(tpl.Engine)
	if err != nil {
		return nil, err
	}

	var data any = map[string]any{}
	if len(req.Data) > 0 {
		_ = json.Unmarshal(req.Data, &data)
	}
	// jsreport pdf-utils compat: when the caller injected {$pdf: {pageNumber,
	// pages: {length}}} into the data (per-page stamp renders), substitute
	// {{$pdf.pageNumber}} / {{$pdf.pages.length}} in the template + CSS
	// BEFORE Handlebars sees them — raymond doesn't recognise `$pdf` as a
	// variable name and would silently emit empty strings.
	if dataMap, ok := data.(map[string]any); ok {
		if pdfVars, ok := dataMap["$pdf"].(map[string]any); ok {
			pageNum := asInt(pdfVars["pageNumber"])
			totalPages := 0
			if pages, ok := pdfVars["pages"].(map[string]any); ok {
				totalPages = asInt(pages["length"])
			}
			tpl.Content = substitutePdfVars(tpl.Content, pageNum, totalPages)
			tpl.CSS = substitutePdfVars(tpl.CSS, pageNum, totalPages)
		}
	}
	var helpers map[string]sandbox.HelperFn
	if tpl.Helpers != "" {
		if hs, herr := r.sandbox.RunHelpers(ctx, tpl.Helpers); herr == nil && hs != nil {
			defer hs.Close()
			helpers = hs.Helpers
		}
	}
	content, _ := r.expandComponents(ctx, tpl.Content)
	content, _ = r.evaluateAssets(ctx, content)
	resolvedCSS, _ := r.evaluateAssets(ctx, tpl.CSS)
	tpl.CSS = resolvedCSS
	rendered, err := eng.Render(content, data, helpers)
	if err != nil {
		return nil, err
	}
	rendered, _ = r.evaluateAssets(ctx, rendered)
	if isHTMLOutput(tpl.Recipe, rendered) {
		rendered = injectStyles(rendered, tpl.CSS, tpl.PageSize, tpl.PageOrientation, tpl.PageMargin)
	}
	rec, ok := r.recipes[tpl.Recipe]
	if !ok {
		return nil, fmt.Errorf("unknown recipe: %s", tpl.Recipe)
	}
	opts := pickTemplateOptions(tpl)
	_ = r.resolveTemplateAssets(ctx, tpl.Recipe, opts)
	out, err := rec.Execute(&recipe.Context{
		Ctx: ctx, HTML: rendered, Data: req.Data,
		TemplateOpts: opts,
		Timeout:      time.Duration(r.cfg.JobTimeoutSeconds) * time.Second,
	})
	if err != nil {
		return nil, err
	}
	if strings.HasPrefix(string(out.Content), "s3://") {
		key := strings.TrimPrefix(string(out.Content), "s3://")
		data, ct, err := r.blob.Get(ctx, key)
		if err != nil {
			return nil, err
		}
		out.Content = data
		if out.MimeType == "" && ct != "" {
			out.MimeType = ct
		}
	}
	return out, nil
}

// expandComponents replaces {{> componentName}} references with the component's
// content, looked up by name in the components table. We do this string-level
// (rather than via raymond's PartialFunc) so the template can be rendered with
// helpers in a single pass.
//
// The regex matches Handlebars partial syntax: {{> name}} or {{>name}}.
func (r *Renderer) expandComponents(ctx context.Context, content string) (string, error) {
	re := componentRegex
	if !re.MatchString(content) {
		return content, nil
	}
	return re.ReplaceAllStringFunc(content, func(match string) string {
		sub := re.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		name := sub[1]
		spec := store.EntitySpecs["components"]
		items, err := r.db.ListGeneric(ctx, spec, "name = $1", []any{name}, "", 1, 0)
		if err != nil || len(items) == 0 {
			return fmt.Sprintf("<!-- component %q not found -->", name)
		}
		body, _ := items[0]["content"].(string)
		return body
	}), nil
}

// evaluateAssets walks `content` looking for jsreport's `{#asset name @encoding=…}`
// references and replaces every occurrence with the resolved asset payload.
//
// Supported encodings (mirrors jsreport-assets):
//
//	utf8 / string  → raw inline text (CSS, JS, HTML, JSON, …) — DEFAULT
//	base64         → base64 of the raw bytes
//	dataURI        → "data:<mime>;base64,…" — useful for <img src> and CSS url()
//	link           → "/assets/<shortid>/content" — the API endpoint URL
//
// Lookup is by asset NAME (not shortid), matching jsreport. If the same name
// exists multiple times the most recently updated wins. To avoid runaway
// recursion we cap the substitution loop at 100 passes — same limit jsreport
// uses (`evaluateAssetsCounter < 100`).
func (r *Renderer) evaluateAssets(ctx context.Context, content string) (string, error) {
	if content == "" || !strings.Contains(content, "{#asset ") {
		return content, nil
	}
	cur := content
	for pass := 0; pass < 100; pass++ {
		if !strings.Contains(cur, "{#asset ") {
			return cur, nil
		}
		var firstErr error
		replaced := assetRegex.ReplaceAllStringFunc(cur, func(match string) string {
			sub := assetRegex.FindStringSubmatch(match)
			if len(sub) < 2 {
				return match
			}
			name, encoding, perr := parseAssetParam(sub[1])
			if perr != nil {
				if firstErr == nil {
					firstErr = perr
				}
				return match
			}
			body, mime, rerr := r.resolveAssetByName(ctx, name)
			if rerr != nil {
				if firstErr == nil {
					firstErr = fmt.Errorf("asset %q: %w", name, rerr)
				}
				return match
			}
			return encodeAsset(body, mime, name, encoding)
		})
		if firstErr != nil {
			return "", firstErr
		}
		if replaced == cur {
			// No more progress — bail (otherwise we'd loop forever on a
			// reference to a missing asset that we deliberately left in place).
			return replaced, nil
		}
		cur = replaced
	}
	return cur, nil
}

// parseAssetParam splits "name @encoding=base64" into ("name", "base64"). When
// no `@encoding=` is given the default is "utf8" (jsreport behaviour).
func parseAssetParam(raw string) (name, encoding string, err error) {
	encoding = "utf8"
	idx := strings.Index(raw, " @")
	if idx == -1 {
		return strings.TrimSpace(raw), encoding, nil
	}
	name = strings.TrimSpace(raw[:idx])
	params := strings.TrimSpace(raw[idx+2:])
	parts := strings.SplitN(params, "=", 2)
	if len(parts) != 2 {
		return name, encoding, fmt.Errorf("wrong asset param spec, expected {#asset name @encoding=base64}")
	}
	if strings.TrimSpace(parts[0]) != "encoding" {
		return name, encoding, fmt.Errorf("unsupported asset param %q", parts[0])
	}
	encoding = strings.TrimSpace(parts[1])
	switch encoding {
	case "utf8", "string", "base64", "dataURI", "link":
	default:
		return name, encoding, fmt.Errorf("unsupported encoding %q (use utf8, base64, dataURI, link, string)", encoding)
	}
	return name, encoding, nil
}

// resolveAssetByName fetches the first asset whose `name` matches `target`.
// Returns the raw payload plus its mime type. Inline-content assets bypass
// the blob store entirely.
func (r *Renderer) resolveAssetByName(ctx context.Context, target string) (data []byte, mime string, err error) {
	spec := store.EntitySpecs["assets"]
	rows, lerr := r.db.ListGeneric(ctx, spec, "name = $1", []any{target}, "", 1, 0)
	if lerr != nil {
		return nil, "", lerr
	}
	if len(rows) == 0 {
		return nil, "", fmt.Errorf("not found")
	}
	a := rows[0]
	if mime, _ = a["mime_type"].(string); mime == "" {
		mime = guessMimeFromName(target)
	}
	if inline, ok := a["inline_content"].(string); ok && inline != "" {
		return []byte(inline), mime, nil
	}
	key, _ := a["blob_key"].(string)
	if key == "" {
		return nil, mime, fmt.Errorf("asset has no content")
	}
	body, ct, gerr := r.blob.Get(ctx, key)
	if gerr != nil {
		return nil, mime, gerr
	}
	if mime == "" {
		mime = ct
	}
	return body, mime, nil
}

func encodeAsset(body []byte, mime, name, encoding string) string {
	switch encoding {
	case "utf8", "string":
		return string(body)
	case "base64":
		return base64.StdEncoding.EncodeToString(body)
	case "dataURI":
		if mime == "" {
			mime = guessMimeFromName(name)
		}
		if mime == "" {
			mime = "application/octet-stream"
		}
		// Match jsreport-assets format: text types get an explicit UTF-8
		// charset declaration so the browser decodes them correctly.
		charset := ""
		if strings.HasPrefix(mime, "text/") {
			charset = "; charset=UTF-8"
		}
		return "data:" + mime + charset + ";base64," + base64.StdEncoding.EncodeToString(body)
	case "link":
		// Best-effort URL — the API serves this regardless of auth so it can
		// be fetched by the chromium worker.
		return "/assets/by-name/" + name
	}
	return string(body)
}

// guessMimeFromName is a tiny mime sniff covering the file types most often
// referenced from a template — images, fonts, web text formats. We avoid
// pulling in net/http for the std mime.TypeByExtension since this list is
// short and predictable.
func guessMimeFromName(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.HasSuffix(lower, ".css"):
		return "text/css"
	case strings.HasSuffix(lower, ".js"):
		return "application/javascript"
	case strings.HasSuffix(lower, ".json"):
		return "application/json"
	case strings.HasSuffix(lower, ".html"), strings.HasSuffix(lower, ".htm"):
		return "text/html"
	case strings.HasSuffix(lower, ".svg"):
		return "image/svg+xml"
	case strings.HasSuffix(lower, ".png"):
		return "image/png"
	case strings.HasSuffix(lower, ".jpg"), strings.HasSuffix(lower, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(lower, ".gif"):
		return "image/gif"
	case strings.HasSuffix(lower, ".webp"):
		return "image/webp"
	case strings.HasSuffix(lower, ".woff"):
		return "font/woff"
	case strings.HasSuffix(lower, ".woff2"):
		return "font/woff2"
	case strings.HasSuffix(lower, ".ttf"):
		return "font/ttf"
	case strings.HasSuffix(lower, ".otf"):
		return "font/otf"
	}
	return ""
}

// runBeforeRenderScripts executes the inline / referenced scripts attached
// to the template and returns the (possibly mutated) data object. Each script
// runs in its own goja VM with a wall-clock timeout. Errors abort rendering.
func (r *Renderer) runBeforeRenderScripts(ctx context.Context, tpl *models.Template, rawData json.RawMessage) (map[string]any, error) {
	if len(tpl.Scripts) == 0 {
		return nil, nil
	}
	var entries []struct {
		Shortid string `json:"shortid"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(tpl.Scripts, &entries); err != nil {
		return nil, nil // malformed scripts: ignore silently
	}
	if len(entries) == 0 {
		return nil, nil
	}
	var data map[string]any
	if len(rawData) > 0 {
		_ = json.Unmarshal(rawData, &data)
	}
	if data == nil {
		data = map[string]any{}
	}
	request := map[string]any{"data": data}
	for _, e := range entries {
		body := e.Content
		if body == "" && e.Shortid != "" {
			s, err := r.db.GetGenericByShortid(ctx, store.EntitySpecs["scripts"], e.Shortid)
			if err != nil {
				continue
			}
			body, _ = s["content"].(string)
		}
		if body == "" {
			continue
		}
		updated, err := r.sandbox.RunBeforeRender(ctx, body, request)
		if err != nil {
			return nil, err
		}
		if updated != nil {
			request = updated
		}
	}
	if d, ok := request["data"].(map[string]any); ok {
		return d, nil
	}
	return nil, nil
}

// resolveHeaderFooterTemplates pre-renders the chrome-pdf
// headerTemplateShortid / footerTemplateShortid into plain HTML and stores it
// in headerTemplate / footerTemplate so the chromium worker can pass them to
// Playwright as-is.
//
// We do this here (instead of the worker) for two reasons:
//   (1) the worker doesn't have access to the template store.
//   (2) renderLeaf() reuses the same render pipeline including helpers and
//       scripts, so the user's beforeRender logic also runs on the header.
func (r *Renderer) resolveHeaderFooterTemplates(ctx context.Context, recipe string, opts map[string]json.RawMessage, data json.RawMessage) error {
	// WeasyPrint uses a different mechanism (CSS @page regions). Translate the
	// {header,footer} bag into CSS and bail before the chrome-specific logic.
	if recipe == "weasyprint" {
		raw, ok := opts["weasyprint"]
		if !ok || len(raw) == 0 {
			return nil
		}
		var bag map[string]any
		if err := json.Unmarshal(raw, &bag); err != nil {
			return nil
		}
		translateWeasyHF(bag)
		if out, err := json.Marshal(bag); err == nil {
			opts["weasyprint"] = out
		}
		return nil
	}
	if recipe != "chrome-pdf" {
		return nil
	}
	// We look at both "chrome" and "chrome-pdf" keys because pickTemplateOptions
	// exposes the chrome bag under both.
	for _, key := range []string{"chrome", "chrome-pdf"} {
		raw, ok := opts[key]
		if !ok || len(raw) == 0 {
			continue
		}
		var bag map[string]any
		if err := json.Unmarshal(raw, &bag); err != nil {
			continue
		}
		changed := false
		for _, pair := range []struct {
			shortidKey string
			htmlKey    string
		}{
			{"headerTemplateShortid", "headerTemplate"},
			{"footerTemplateShortid", "footerTemplate"},
		} {
			sid, _ := bag[pair.shortidKey].(string)
			if sid == "" {
				continue
			}
			req := &Request{}
			req.Template.Shortid = sid
			req.Data = data
			sub, err := r.renderLeaf(ctx, req)
			if err != nil {
				return fmt.Errorf("%s %q: %w", pair.shortidKey, sid, err)
			}
			html := string(sub.Content)
			// Strip <html><body> envelope if the rendered output is a full
			// page — Chromium expects a "fragment" for header/footer.
			html = stripHTMLEnvelope(html)
			bag[pair.htmlKey] = html
			changed = true
		}
		if changed {
			if out, err := json.Marshal(bag); err == nil {
				opts[key] = out
			}
		}
	}
	return nil
}

// translateWeasyHF turns the simple {header: {left,center,right}, footer: …}
// shape from the UI into a CSS @page block that WeasyPrint understands. We
// append it to the options.css so the worker's "extra CSS" path picks it up.
//
// Special tokens supported:
//
//	{page}   → counter(page)
//	{pages}  → counter(pages)
//	{title}  → string(title) — requires the template to set running titles
func translateWeasyHF(bag map[string]any) {
	type triple = map[string]string
	collect := func(key string) triple {
		out := triple{}
		raw, _ := bag[key].(map[string]any)
		for _, p := range []string{"left", "center", "right"} {
			if v, ok := raw[p].(string); ok && v != "" {
				out[p] = v
			}
		}
		return out
	}
	header := collect("header")
	footer := collect("footer")
	if len(header) == 0 && len(footer) == 0 {
		return
	}

	pageRegion := func(side string, m triple, isFooter bool) string {
		val, ok := m[side]
		if !ok {
			return ""
		}
		// Escape double quotes for CSS string content.
		val = strings.ReplaceAll(val, `"`, `\"`)
		// Replace placeholders with CSS counter() calls.
		var content strings.Builder
		i := 0
		for i < len(val) {
			if strings.HasPrefix(val[i:], "{page}") {
				content.WriteString(`" counter(page) "`)
				i += len("{page}")
				continue
			}
			if strings.HasPrefix(val[i:], "{pages}") {
				content.WriteString(`" counter(pages) "`)
				i += len("{pages}")
				continue
			}
			content.WriteByte(val[i])
			i++
		}
		region := "@top-" + side
		if isFooter {
			region = "@bottom-" + side
		}
		return region + ` { content: "` + content.String() + `"; }`
	}

	var sb strings.Builder
	sb.WriteString("@page {")
	for _, side := range []string{"left", "center", "right"} {
		if r := pageRegion(side, header, false); r != "" {
			sb.WriteString(" ")
			sb.WriteString(r)
		}
		if r := pageRegion(side, footer, true); r != "" {
			sb.WriteString(" ")
			sb.WriteString(r)
		}
	}
	sb.WriteString(" }")

	// Append to existing css (so user's overrides still apply).
	prev, _ := bag["css"].(string)
	if prev != "" {
		prev += "\n"
	}
	bag["css"] = prev + sb.String()
}

// stripHTMLEnvelope is a tiny helper that pulls the contents of <body> if the
// rendered HTML is a full document. Chromium's headerTemplate/footerTemplate
// expects a fragment.
func stripHTMLEnvelope(html string) string {
	low := strings.ToLower(html)
	bs := strings.Index(low, "<body")
	be := strings.Index(low, "</body>")
	if bs < 0 || be < 0 {
		return html
	}
	// move past the closing > of the body open tag
	bsEnd := strings.Index(html[bs:], ">")
	if bsEnd < 0 {
		return html
	}
	start := bs + bsEnd + 1
	if start >= be {
		return html
	}
	return html[start:be]
}

// resolveTemplateAssets looks at template options for the office recipes and
// rewrites any `templateAsset: {shortid: "..."}` into `templateAssetKey:
// "<blob-key>"` so workers can fetch the file directly from SeaweedFS.
func (r *Renderer) resolveTemplateAssets(ctx context.Context, recipe string, opts map[string]json.RawMessage) error {
	key := ""
	switch recipe {
	case "docx", "pptx", "xlsx":
		key = recipe
	default:
		return nil
	}
	raw, ok := opts[key]
	if !ok || len(raw) == 0 {
		return nil
	}
	var holder struct {
		TemplateAsset *struct {
			Shortid string `json:"shortid"`
		} `json:"templateAsset"`
		TemplateAssetKey string `json:"templateAssetKey"`
	}
	if err := json.Unmarshal(raw, &holder); err != nil {
		return nil // unparseable — leave options as-is
	}
	if holder.TemplateAssetKey != "" || holder.TemplateAsset == nil || holder.TemplateAsset.Shortid == "" {
		return nil
	}
	asset, err := r.db.GetGenericByShortid(ctx, store.EntitySpecs["assets"], holder.TemplateAsset.Shortid)
	if err != nil {
		return fmt.Errorf("templateAsset %q not found", holder.TemplateAsset.Shortid)
	}
	blobKey, _ := asset["blob_key"].(string)
	if blobKey == "" {
		return fmt.Errorf("templateAsset %q has no blob_key", holder.TemplateAsset.Shortid)
	}

	// Merge templateAssetKey into the existing options object.
	var merged map[string]any
	_ = json.Unmarshal(raw, &merged)
	if merged == nil {
		merged = map[string]any{}
	}
	merged["templateAssetKey"] = blobKey
	if out, err := json.Marshal(merged); err == nil {
		opts[key] = out
	}
	return nil
}

// pickTemplateOptions exposes the per-recipe option bag keyed by both the
// recipe name AND the historical short key, so the worker proxy + recipes
// can always find their options regardless of how they look it up.
func pickTemplateOptions(t *models.Template) map[string]json.RawMessage {
	m := map[string]json.RawMessage{}

	// Template-level page settings (Página tab) are the single source of
	// truth for format / orientation / margins. We merge them INTO the
	// recipe-specific JSONB so the worker sees a single, consistent set of
	// options regardless of where the user edited them.
	if t.Recipe == "chrome-pdf" {
		merged := mergeChromeWithPageSettings(t.Chrome, t.PageSize, t.PageOrientation, t.PageMargin)
		if len(merged) > 0 {
			m["chrome"] = merged
			m["chrome-pdf"] = merged // worker proxy lookup
		}
	} else if len(t.Chrome) > 0 {
		// For non-chrome recipes, expose the JSONB as-is so other workers can
		// still read it if they cross-reference (rare).
		m["chrome"] = t.Chrome
		m["chrome-pdf"] = t.Chrome
	}

	if len(t.WeasyPrint) > 0 {
		m["weasyprint"] = t.WeasyPrint
	}
	if len(t.Docx) > 0 {
		m["docx"] = t.Docx
	}
	if len(t.Xlsx) > 0 {
		m["xlsx"] = t.Xlsx
	}
	if len(t.Pptx) > 0 {
		m["pptx"] = t.Pptx
	}
	return m
}

// mergeChromeWithPageSettings folds the template-level Página fields
// (PageSize, PageOrientation, PageMargin) into the recipe's chrome JSONB.
// Página tab wins over whatever may have been left in the JSONB by older
// versions of the UI — there is only one place to edit these now.
func mergeChromeWithPageSettings(chromeJSON json.RawMessage, pageSize, orientation, margin string) json.RawMessage {
	var c map[string]any
	if len(chromeJSON) > 0 {
		_ = json.Unmarshal(chromeJSON, &c)
	}
	if c == nil {
		c = map[string]any{}
	}

	if pageSize != "" {
		c["format"] = pageSize
		// When the Página tab specifies a size, it MUST win over whatever
		// the user wrote in `@page { size: ... }` inside the template's
		// CSS. Otherwise Chrome silently honours the CSS rule and the page
		// renders in the wrong format (e.g. A4 instead of Letter).
		c["preferCSSPageSize"] = false
	}
	// orientation is always set (even when "portrait" → landscape=false) so
	// switching from landscape back to portrait actually flips the bit.
	if orientation == "landscape" {
		c["landscape"] = true
	} else if orientation != "" {
		c["landscape"] = false
	}

	if margin != "" {
		top, right, bottom, left := parseFourSided(margin)
		c["margin"] = map[string]any{
			"top":    top,
			"right":  right,
			"bottom": bottom,
			"left":   left,
		}
	}

	out, _ := json.Marshal(c)
	return out
}

// stripAtPageSizeRE matches `size: <whatever>;` (with optional surrounding
// whitespace) inside CSS, so we can drop only the `size` descriptor from
// `@page` blocks without touching `margin`, `marks`, etc.
var stripAtPageSizeRE = regexp.MustCompile(`(?i)\bsize\s*:[^;{}]*;?`)

// atPageBlockRE finds @page blocks (and named variants like @page :first) so
// we can rewrite only the body, not random `size:` in unrelated selectors.
var atPageBlockRE = regexp.MustCompile(`(?is)@page\b[^{]*\{[^}]*\}`)

// stripAtPageSize removes any `size: <foo>;` rules from inside @page blocks.
// The Página tab is the source of truth for page format; leaving the user's
// `@page { size: A4 }` in the HTML/CSS causes Chromium to honour it and
// silently override the Página tab choice.
func stripAtPageSize(s string) string {
	if s == "" || !strings.Contains(s, "@page") {
		return s
	}
	return atPageBlockRE.ReplaceAllStringFunc(s, func(block string) string {
		return stripAtPageSizeRE.ReplaceAllString(block, "")
	})
}

// parseFourSided accepts CSS-style margin shorthand:
//
//	"1cm"             → 1cm on every side
//	"1cm 2cm"         → 1cm vertical, 2cm horizontal
//	"1cm 2cm 3cm"     → top=1cm, sides=2cm, bottom=3cm
//	"1cm 2cm 3cm 4cm" → top, right, bottom, left
//
// Anything that doesn't match the 1-4 token shape falls through as the raw
// value on every side, which matches how the chromium worker tolerates
// "1cm" passed as a string in the older flow.
func parseFourSided(s string) (string, string, string, string) {
	parts := strings.Fields(strings.TrimSpace(s))
	switch len(parts) {
	case 1:
		return parts[0], parts[0], parts[0], parts[0]
	case 2:
		return parts[0], parts[1], parts[0], parts[1]
	case 3:
		return parts[0], parts[1], parts[2], parts[1]
	case 4:
		return parts[0], parts[1], parts[2], parts[3]
	}
	return s, s, s, s
}

// ---- profile lifecycle ----

func (r *Renderer) startProfile(ctx context.Context, t *models.Template, u *models.User) (*models.Profile, error) {
	spec := store.EntitySpecs["profiles"]
	payload := map[string]any{
		"template_shortid": t.Shortid,
		"state":            "running",
		"mode":             "standard",
		"timeout_ms":       r.cfg.JobTimeoutSeconds * 1000,
		"started_at":       time.Now(),
	}
	if u != nil {
		payload["owner_id"] = u.ID
	}
	out, err := r.db.InsertGeneric(ctx, spec, payload)
	if err != nil {
		return nil, err
	}
	p := &models.Profile{}
	if v, ok := out["shortid"].(string); ok {
		p.Shortid = v
	}
	// Enforce the hard cap on profile history (detached so it never blocks or
	// fails the render that just started).
	go func() {
		cctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		r.capProfiles(cctx)
	}()
	return p, nil
}

func (r *Renderer) finishProfile(ctx context.Context, p *models.Profile, state string, e error) {
	if p == nil {
		return
	}
	spec := store.EntitySpecs["profiles"]
	now := time.Now()
	payload := map[string]any{
		"state":       state,
		"finished_at": now,
	}
	if e != nil {
		s := e.Error()
		payload["error"] = s
	}
	_, _ = r.db.UpdateGeneric(ctx, spec, p.Shortid, payload)
}
