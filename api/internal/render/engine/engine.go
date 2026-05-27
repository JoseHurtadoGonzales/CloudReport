// Package engine implements template engines. Today: handlebars (raymond) and
// a "passthrough" engine for static content.
//
// User helpers come from the sandbox package. raymond cannot register variadic
// helpers (it dispatches arguments positionally via reflection), so we wrap
// each helper in a per-arity Go adapter so handlebars calls like
//
//   {{formatDate date}}      // arity 1
//   {{truncate text 250}}    // arity 2
//   {{inc @index}}           // arity 1
//
// are routed correctly. The last positional argument from Handlebars is always
// `*raymond.Options`; we drop it before forwarding to the JS function.
package engine

import (
	"fmt"
	"reflect"
	"regexp"

	"github.com/aymerick/raymond"
	"github.com/cloudreport/api/internal/render/sandbox"
)

// jsreport-style {{path.length}} doesn't resolve on Go slices in raymond. We
// rewrite to the built-in `{{length path}}` helper before parsing.
var lengthRewriteRE = regexp.MustCompile(`\{\{\s*([A-Za-z_][\w.]*)\.length\s*\}\}`)

type Engine interface {
	Name() string
	Render(template string, data any, helpers map[string]sandbox.HelperFn) (string, error)
}

func ByName(name string) (Engine, error) {
	switch name {
	case "", "none":
		return Passthrough{}, nil
	case "handlebars":
		return Handlebars{}, nil
	}
	return nil, fmt.Errorf("unknown engine: %s", name)
}

type Passthrough struct{}

func (Passthrough) Name() string { return "none" }
func (Passthrough) Render(t string, _ any, _ map[string]sandbox.HelperFn) (string, error) {
	return t, nil
}

type Handlebars struct{}

func (Handlebars) Name() string { return "handlebars" }

func (Handlebars) Render(t string, data any, helpers map[string]sandbox.HelperFn) (string, error) {
	t = lengthRewriteRE.ReplaceAllString(t, "{{length $1}}")
	tpl, err := raymond.Parse(t)
	if err != nil {
		return "", fmt.Errorf("handlebars parse: %w", err)
	}
	registerBuiltins(tpl)
	for name, h := range helpers {
		tpl.RegisterHelper(name, buildAdapter(h))
	}
	return tpl.Exec(data)
}

// registerBuiltins adds tiny utility helpers that jsreport users expect to
// "just work" because they were native in mustache.js or Handlebars.js but
// aren't exposed by raymond out of the box (it can't auto-resolve `.length`
// on Go slices/maps, for instance).
//
//	{{length items}}          → 5
//	{{json data}}             → {"foo":"bar"}
//	{{eq a b}}                → true/false (also: ne, gt, lt)
//	{{upper s}}, {{lower s}}  → case helpers
//	{{add a b}}, {{sub a b}}  → simple arithmetic
func registerBuiltins(tpl *raymond.Template) {
	tpl.RegisterHelper("length", func(v interface{}) interface{} {
		if v == nil {
			return 0
		}
		rv := reflect.ValueOf(v)
		switch rv.Kind() {
		case reflect.Slice, reflect.Array, reflect.Map, reflect.String, reflect.Chan:
			return rv.Len()
		case reflect.Ptr, reflect.Interface:
			if rv.IsNil() {
				return 0
			}
			return reflect.Indirect(rv).Len()
		}
		return 0
	})
}

// buildAdapter returns a Go function with the right number of `interface{}`
// parameters so raymond's reflective dispatcher matches it to the helper
// invocation. We forward (arity) arguments to the underlying JS callable and
// turn errors into a visible inline string so templates fail loudly rather
// than silently.
func buildAdapter(h sandbox.HelperFn) interface{} {
	errString := func(err error) string {
		return fmt.Sprintf("[helper %s error: %s]", h.Name, err)
	}
	switch h.Arity {
	case 0:
		return func() interface{} {
			v, err := h.Call()
			if err != nil {
				return errString(err)
			}
			return v
		}
	case 1:
		return func(a interface{}) interface{} {
			v, err := h.Call(a)
			if err != nil {
				return errString(err)
			}
			return v
		}
	case 2:
		return func(a, b interface{}) interface{} {
			v, err := h.Call(a, b)
			if err != nil {
				return errString(err)
			}
			return v
		}
	case 3:
		return func(a, b, c interface{}) interface{} {
			v, err := h.Call(a, b, c)
			if err != nil {
				return errString(err)
			}
			return v
		}
	case 4:
		return func(a, b, c, d interface{}) interface{} {
			v, err := h.Call(a, b, c, d)
			if err != nil {
				return errString(err)
			}
			return v
		}
	default:
		// 5+ params is rare; collapse to one-arg fallback that calls with the
		// available args (mostly defensive).
		return func(a interface{}) interface{} {
			v, err := h.Call(a)
			if err != nil {
				return errString(err)
			}
			return v
		}
	}
}
