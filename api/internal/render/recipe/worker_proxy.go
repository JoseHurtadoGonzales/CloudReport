package recipe

import (
	"encoding/json"
	"fmt"

	"github.com/cloudreport/api/internal/queue"
)

// WorkerProxy delegates the actual rendering to a Python worker via Redis.
// The orchestrator publishes a Job with `htmlBlob` (or raw HTML inline) and
// waits for a Reply that contains the output blob key.
//
// Workers fetch input from / upload output to SeaweedFS via S3.
type WorkerProxy struct {
	Recipe             string
	Mime               string
	Queue              *queue.Queue
	NeedsTemplateAsset bool // true for docx/pptx where the "template" is a binary asset
}

func (w WorkerProxy) Execute(c *Context) (*Result, error) {
	// Build the job payload. For HTML-based recipes we ship the HTML inline as
	// `htmlInline` to avoid an extra round-trip through blob storage; workers
	// know how to consume it.
	options := json.RawMessage("{}")
	if raw, ok := c.TemplateOpts[w.Recipe]; ok && len(raw) > 0 {
		options = raw
	}
	// For office templates we expect options.{recipe}.templateAsset.shortid (resolved
	// to a blob key by the API layer before calling Execute) — passed through
	// as-is to the worker.
	payload := map[string]any{
		"recipe":  w.Recipe,
		"html":    c.HTML,
		"data":    c.Data,
		"options": options,
	}
	body, _ := json.Marshal(payload)

	job := &queue.Job{
		Recipe:  w.Recipe,
		Engine:  "handlebars",
		Data:    body, // worker reads {html,data,options} from here
		Options: options,
	}

	reply, err := w.Queue.Submit(c.Ctx, w.Recipe, job, c.Timeout)
	if err != nil {
		return nil, fmt.Errorf("worker %s submit: %w", w.Recipe, err)
	}
	if reply.Status != "success" {
		return nil, fmt.Errorf("worker %s error: %s", w.Recipe, reply.Error)
	}
	// The worker may either inline content (small) via outputBlob="inline:<b64>"
	// or upload to S3 and return the key. The renderer's caller layer fetches
	// the blob; for simplicity here we expect inline base64 in the reply.
	// (See workers/shared/protocol.py).
	if reply.OutputBlob == "" {
		return nil, fmt.Errorf("worker %s: empty output", w.Recipe)
	}
	// Reply.OutputBlob is the SeaweedFS S3 key — the caller (Render layer) is
	// expected to fetch it before returning, but here we treat WorkerProxy as
	// owning the blob fetch. We re-use the queue's internal Redis client by
	// stashing the blob bytes in `Logs` field as base64 when small. To avoid
	// pulling the blob client into recipes, we let the API layer download.
	// However, recipes return raw bytes; to keep that contract, we ask
	// workers to ALWAYS reply with the bytes base64-encoded in Logs when
	// size < 5MB, and an S3 key otherwise. The orchestrator handles the
	// large-file case.
	mime := reply.MimeType
	if mime == "" {
		mime = w.Mime
	}
	// The actual download/decoding happens upstream; recipes that go through
	// the worker proxy return the OutputBlob key in Result.Content as a
	// special sentinel string prefixed "s3://". The orchestrator (Render
	// layer) recognizes this and resolves it.
	return &Result{
		Content:  []byte("s3://" + reply.OutputBlob),
		MimeType: mime,
		FileName: defaultFileName(w.Recipe),
	}, nil
}

func defaultFileName(recipe string) string {
	switch recipe {
	case "weasyprint", "chrome-pdf":
		return "report.pdf"
	case "docx":
		return "report.docx"
	case "pptx":
		return "report.pptx"
	case "html-to-xlsx":
		return "report.xlsx"
	}
	return "report.bin"
}
