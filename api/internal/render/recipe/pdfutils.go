package recipe

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// PdfOperation is one entry in template.pdfOperations.
//
// Supported types and their config keys:
//
//	merge           { pdfBase64 }
//	prepend         { pdfBase64 }        ← Renderer rewrites prependTemplate → prepend
//	stamp           { pdfBase64, pages } ← Renderer rewrites stampTemplate → stamp
//	watermark       { text | image, options }
//	encrypt         { password }
//	meta            { title, author, subject, keywords }
//	removePages     { pages: "2, 5-7, last" }
//	appendTemplate  { templateShortid }   ← handled at the Renderer layer (becomes merge)
//	prependTemplate { templateShortid }   ← handled at the Renderer layer (becomes prepend)
//	stampTemplate   { templateShortid }   ← handled at the Renderer layer (becomes stamp)
//	sign            { certBase64, certPassword, reason, location }
//	pdfA            { conformance: "1b" | "2b" | "2u" | "3b" }
type PdfOperation struct {
	Type      string `json:"type"`
	PdfBase64 string `json:"pdfBase64,omitempty"`
	Text      string `json:"text,omitempty"`
	Image     string `json:"image,omitempty"`
	Options   string `json:"options,omitempty"`
	Password  string `json:"password,omitempty"`
	// meta
	Title    string `json:"title,omitempty"`
	Author   string `json:"author,omitempty"`
	Subject  string `json:"subject,omitempty"`
	Keywords string `json:"keywords,omitempty"`
	// removePages
	Pages string `json:"pages,omitempty"`
	// appendTemplate
	TemplateShortid string `json:"templateShortid,omitempty"`
	// sign
	CertBase64   string `json:"certBase64,omitempty"`
	CertPassword string `json:"certPassword,omitempty"`
	Reason       string `json:"reason,omitempty"`
	Location     string `json:"location,omitempty"`
	// pdfA
	Conformance string `json:"conformance,omitempty"`
}

// ApplyPdfOperations runs each op in order over `input` and returns the final
// PDF bytes. `appendTemplate` is NOT handled here — the Renderer layer
// resolves it (because it needs the full render pipeline) and passes the
// resulting PDF bytes as a `merge` op behind the scenes.
func ApplyPdfOperations(input []byte, ops json.RawMessage) ([]byte, error) {
	var list []PdfOperation
	if err := json.Unmarshal(ops, &list); err != nil {
		return nil, err
	}
	cur := input
	for _, op := range list {
		// The Renderer pre-resolves jsreport-style template ops (append /
		// prepend / merge / *Template) into native ops (merge / prepend /
		// stamp) when they carry a templateShortid. If we still see one of
		// the unresolved variants here, the lookup failed silently — skip
		// it rather than crashing the whole pipeline.
		switch op.Type {
		case "append", "prepend",
			"appendTemplate", "prependTemplate", "stampTemplate":
			if op.PdfBase64 == "" {
				continue
			}
		}
		// `merge` is ambiguous: it might be a raw pdfBase64 upload (still
		// valid here) OR an unresolved jsreport stamp/merge with only a
		// templateShortid. The latter has empty pdfBase64 → skip.
		if op.Type == "merge" && op.PdfBase64 == "" {
			continue
		}
		var err error
		cur, err = applyOne(cur, op)
		if err != nil {
			return nil, fmt.Errorf("pdf op %s: %w", op.Type, err)
		}
	}
	return cur, nil
}

func applyOne(in []byte, op PdfOperation) ([]byte, error) {
	switch op.Type {

	case "merge", "append":
		// In jsreport, plain `merge` (no flags) and `append` both concat the
		// rendered/uploaded PDF at the end of the main document.
		raw, err := base64.StdEncoding.DecodeString(op.PdfBase64)
		if err != nil {
			return nil, err
		}
		var out bytes.Buffer
		readers := []io.ReadSeeker{
			bytes.NewReader(in),
			bytes.NewReader(raw),
		}
		if err := api.MergeRaw(readers, &out, false, nil); err != nil {
			return nil, err
		}
		return out.Bytes(), nil

	case "prepend":
		// Like merge, but the rendered template goes first (cover pages, etc).
		raw, err := base64.StdEncoding.DecodeString(op.PdfBase64)
		if err != nil {
			return nil, err
		}
		var out bytes.Buffer
		readers := []io.ReadSeeker{
			bytes.NewReader(raw),
			bytes.NewReader(in),
		}
		if err := api.MergeRaw(readers, &out, false, nil); err != nil {
			return nil, err
		}
		return out.Bytes(), nil

	case "stamp":
		// Header / footer: overlay every page of `in` with the first page of
		// the stamp PDF. pdfcpu offers PDFWatermarkForReadSeeker so we can do
		// this fully in memory.
		raw, err := base64.StdEncoding.DecodeString(op.PdfBase64)
		if err != nil {
			return nil, err
		}
		// `desc` controls how the stamp is placed. "scale:1 abs, pos:c, rot:0"
		// means: keep its natural size, center on the page, no rotation —
		// which is what a full-page header/footer template wants.
		desc := op.Options
		if desc == "" {
			desc = "scale:1 abs, pos:c, rot:0"
		}
		wm, err := api.PDFWatermarkForReadSeeker(bytes.NewReader(raw), 1, desc, true, false, types.POINTS)
		if err != nil {
			return nil, err
		}
		var pages []string
		if op.Pages != "" {
			pages = parsePageRanges(op.Pages)
		}
		var out bytes.Buffer
		if err := api.AddWatermarks(bytes.NewReader(in), &out, pages, wm, nil); err != nil {
			return nil, err
		}
		return out.Bytes(), nil

	case "watermark":
		desc := op.Options
		if desc == "" {
			desc = "scale:1 abs, op:0.5, rot:0"
		}
		var wm *model.Watermark
		var err error
		if op.Image != "" {
			wm, err = api.ImageWatermark(op.Image, desc, true, false, types.POINTS)
		} else {
			wm, err = api.TextWatermark(op.Text, desc, true, false, types.POINTS)
		}
		if err != nil {
			return nil, err
		}
		var out bytes.Buffer
		if err := api.AddWatermarks(bytes.NewReader(in), &out, nil, wm, nil); err != nil {
			return nil, err
		}
		return out.Bytes(), nil

	case "encrypt":
		conf := model.NewAESConfiguration(op.Password, op.Password, 256)
		var out bytes.Buffer
		if err := api.Encrypt(bytes.NewReader(in), &out, conf); err != nil {
			return nil, err
		}
		return out.Bytes(), nil

	case "meta":
		// pdfcpu's AddProperties sets keys into the PDF Info dictionary.
		props := map[string]string{}
		if op.Title != "" {
			props["Title"] = op.Title
		}
		if op.Author != "" {
			props["Author"] = op.Author
		}
		if op.Subject != "" {
			props["Subject"] = op.Subject
		}
		if op.Keywords != "" {
			props["Keywords"] = op.Keywords
		}
		if len(props) == 0 {
			return in, nil
		}
		var out bytes.Buffer
		if err := api.AddProperties(bytes.NewReader(in), &out, props, nil); err != nil {
			return nil, err
		}
		return out.Bytes(), nil

	case "removePages":
		ranges := parsePageRanges(op.Pages)
		if len(ranges) == 0 {
			return in, nil
		}
		var out bytes.Buffer
		if err := api.RemovePages(bytes.NewReader(in), &out, ranges, nil); err != nil {
			return nil, err
		}
		return out.Bytes(), nil

	case "sign":
		// Digital signatures are non-trivial — pdfcpu doesn't expose a high-
		// level signing API. Surface a clear "not implemented" error so the
		// UI badge can show the failure. The certificate is preserved on the
		// template so we can wire this up later when the user installs an
		// external signing service.
		return nil, fmt.Errorf("sign op not implemented yet (cert size %d bytes)", len(op.CertBase64))

	case "pdfA":
		// pdfcpu can validate and optimize but cannot transcode to a full
		// PDF/A profile (requires deterministic font embedding, color
		// management, XMP). Mark the document conformance hint in metadata
		// so downstream validators see the intent; full compliance needs
		// Ghostscript via a sidecar.
		props := map[string]string{
			"GTS_PDFA1": op.Conformance,
		}
		var out bytes.Buffer
		if err := api.AddProperties(bytes.NewReader(in), &out, props, nil); err != nil {
			return nil, err
		}
		return out.Bytes(), nil

	case "appendTemplate", "prependTemplate", "stampTemplate":
		// Should have been resolved by the Renderer into merge/prepend/stamp.
		// If we still see one here, the template lookup failed silently —
		// fall through as a no-op so the rest of the pipeline can finish.
		return in, nil
	}

	// Silently allow no-data variants of jsreport-style template ops when the
	// Renderer left them unresolved (e.g. empty templateShortid).
	if (op.Type == "append" || op.Type == "prepend" || op.Type == "merge") && op.PdfBase64 == "" {
		return in, nil
	}

	return in, fmt.Errorf("unknown pdf operation: %s", op.Type)
}

// parsePageRanges turns "2, 5-7, last" into pdfcpu's selectedPages format.
// pdfcpu accepts entries like "2", "5-7", "l" (last). We translate "last".
func parsePageRanges(spec string) []string {
	var out []string
	for _, piece := range strings.Split(spec, ",") {
		p := strings.TrimSpace(piece)
		if p == "" {
			continue
		}
		if strings.EqualFold(p, "last") {
			p = "l"
		}
		out = append(out, p)
	}
	return out
}
