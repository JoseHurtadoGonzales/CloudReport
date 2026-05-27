// Package recipe defines the contract for rendering recipes (the "what's the
// output format" piece) and ships the implementations that run in-process.
package recipe

import (
	"context"
	"encoding/json"
	"time"
)

type Context struct {
	Ctx          context.Context
	HTML         string                       // engine output
	Data         json.RawMessage              // raw request data
	TemplateOpts map[string]json.RawMessage   // per-recipe options
	Timeout      time.Duration
}

type Result struct {
	Content  []byte
	MimeType string
	FileName string
}

type Recipe interface {
	Execute(c *Context) (*Result, error)
}
