package recipe

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudreport/api/internal/blob"
)

// StaticPDF returns a pre-existing PDF (passthrough). The PDF source can come
// from: options.staticPdf.content (base64), or options.staticPdf.assetBlobKey
// (S3 key).
type StaticPDF struct {
	Blob *blob.Store
}

type staticPdfOpts struct {
	Content      string `json:"content"` // base64
	AssetBlobKey string `json:"assetBlobKey"`
}

func (s StaticPDF) Execute(c *Context) (*Result, error) {
	raw, ok := c.TemplateOpts["staticPdf"]
	if !ok {
		// the HTML output itself might be base64
		if strings.HasPrefix(c.HTML, "%PDF-") {
			return &Result{Content: []byte(c.HTML), MimeType: "application/pdf", FileName: "report.pdf"}, nil
		}
		b, err := base64.StdEncoding.DecodeString(c.HTML)
		if err == nil && bytes.HasPrefix(b, []byte("%PDF-")) {
			return &Result{Content: b, MimeType: "application/pdf", FileName: "report.pdf"}, nil
		}
		return nil, fmt.Errorf("static-pdf: no source provided")
	}
	var opts staticPdfOpts
	if err := json.Unmarshal(raw, &opts); err != nil {
		return nil, err
	}
	if opts.Content != "" {
		b, err := base64.StdEncoding.DecodeString(opts.Content)
		if err != nil {
			return nil, fmt.Errorf("staticPdf.content base64: %w", err)
		}
		return &Result{Content: b, MimeType: "application/pdf", FileName: "report.pdf"}, nil
	}
	if opts.AssetBlobKey != "" {
		data, _, err := s.Blob.Get(c.Ctx, opts.AssetBlobKey)
		if err != nil {
			return nil, err
		}
		return &Result{Content: data, MimeType: "application/pdf", FileName: "report.pdf"}, nil
	}
	return nil, fmt.Errorf("static-pdf: empty options")
}
