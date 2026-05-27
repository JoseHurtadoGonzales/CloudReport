// Package queue implements a request/reply pattern over Redis Streams.
//
// Producer (this package) XADDs to "renders:<recipe>" and waits on
// "renders:replies:<jobId>". Workers consume the request stream as part of a
// consumer group, do the work, XADD the reply, and XACK.
package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/cloudreport/api/internal/config"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/redis/go-redis/v9"
)

type Queue struct {
	r   *redis.Client
	cfg *config.Config
}

type Job struct {
	ID         string          `json:"id"`
	Recipe     string          `json:"recipe"`
	TemplateID string          `json:"templateId,omitempty"`
	Engine     string          `json:"engine,omitempty"`
	InputBlob  string          `json:"inputBlob,omitempty"`  // S3 key to template file (docx/pptx/xlsx)
	HTMLBlob   string          `json:"htmlBlob,omitempty"`   // S3 key to rendered HTML
	Data       json.RawMessage `json:"data,omitempty"`
	Options    json.RawMessage `json:"options,omitempty"`
	ReplyTo    string          `json:"replyTo"`
	DeadlineMs int64           `json:"deadlineMs"`
}

type Reply struct {
	ID          string `json:"id"`
	Status      string `json:"status"` // success | error
	Error       string `json:"error,omitempty"`
	OutputBlob  string `json:"outputBlob,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
	DurationMs  int64  `json:"durationMs"`
	Logs        string `json:"logs,omitempty"`
}

func New(ctx context.Context, cfg *config.Config) (*Queue, error) {
	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		return nil, err
	}
	r := redis.NewClient(opt)
	if err := r.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	return &Queue{r: r, cfg: cfg}, nil
}

func (q *Queue) Close() error { return q.r.Close() }

// Submit pushes a job onto the recipe stream and blocks until reply arrives or
// the timeout elapses.
func (q *Queue) Submit(ctx context.Context, recipe string, job *Job, timeout time.Duration) (*Reply, error) {
	if job.ID == "" {
		id, _ := gonanoid.New(16)
		job.ID = id
	}
	if timeout <= 0 {
		timeout = time.Duration(q.cfg.JobTimeoutSeconds) * time.Second
	}
	job.Recipe = recipe
	job.ReplyTo = "renders:replies:" + job.ID
	job.DeadlineMs = time.Now().Add(timeout).UnixMilli()

	body, err := json.Marshal(job)
	if err != nil {
		return nil, err
	}

	stream := "renders:" + recipe

	// Ensure consumer group exists is the worker's responsibility, but we
	// guarantee MKSTREAM via XADD which auto-creates the stream.
	if err := q.r.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		Values: map[string]any{"payload": body},
	}).Err(); err != nil {
		return nil, fmt.Errorf("xadd: %w", err)
	}

	// Block waiting for the reply on a per-job stream.
	deadline := time.Now().Add(timeout)
	lastID := "0-0"
	for time.Now().Before(deadline) {
		remaining := time.Until(deadline)
		if remaining < 200*time.Millisecond {
			remaining = 200 * time.Millisecond
		}
		res, err := q.r.XRead(ctx, &redis.XReadArgs{
			Streams: []string{job.ReplyTo, lastID},
			Count:   1,
			Block:   remaining,
		}).Result()
		if err == redis.Nil {
			continue
		}
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil, err
			}
			// transient
			time.Sleep(100 * time.Millisecond)
			continue
		}
		for _, s := range res {
			for _, m := range s.Messages {
				lastID = m.ID
				raw, _ := m.Values["payload"].(string)
				var rep Reply
				if err := json.Unmarshal([]byte(raw), &rep); err != nil {
					return nil, err
				}
				// cleanup
				_ = q.r.Del(ctx, job.ReplyTo).Err()
				return &rep, nil
			}
		}
	}
	return nil, fmt.Errorf("job %s timeout after %s", job.ID, timeout)
}

// PublishReply is exposed for tests / in-process workers.
func (q *Queue) PublishReply(ctx context.Context, replyTo string, r *Reply) error {
	body, _ := json.Marshal(r)
	return q.r.XAdd(ctx, &redis.XAddArgs{
		Stream: replyTo,
		Values: map[string]any{"payload": body},
	}).Err()
}
