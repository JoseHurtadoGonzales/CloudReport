// Package sandbox executes user-supplied JavaScript (helpers + lifecycle
// scripts) in an isolated goja VM with a wall-clock timeout.
//
// goja is a pure-Go JS interpreter (~ES2015). It is intentionally less
// capable than V8 — there is no `require`, no filesystem, no network.
package sandbox

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/dop251/goja"
)

var ErrTimeout = errors.New("script timeout")

type Sandbox struct {
	timeout time.Duration
}

func New(timeout time.Duration) *Sandbox {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &Sandbox{timeout: timeout}
}

// HelperFn represents a user helper plucked out of the sandbox. `Arity` is the
// declared parameter count of the JS function (`Function.length`), used by
// the template engine to register an adapter with the right signature.
type HelperFn struct {
	Name  string
	Arity int
	Call  func(args ...any) (any, error)
}

// HelperSet bundles a live VM with the helper closures that reach into it.
// Callers MUST invoke Close() when the helpers are no longer needed (e.g.
// after the render finishes) — that releases the VM and stops the watchdog.
type HelperSet struct {
	Helpers map[string]HelperFn
	close   func()
	once    sync.Once
}

func (h *HelperSet) Close() {
	if h == nil {
		return
	}
	h.once.Do(func() {
		if h.close != nil {
			h.close()
		}
	})
}

// RunHelpers compiles a helpers script and returns the set of callable
// helpers. The VM stays alive until HelperSet.Close() is invoked, which
// must happen before the parent context goes away.
func (s *Sandbox) RunHelpers(ctx context.Context, helpers string) (*HelperSet, error) {
	if helpers == "" {
		return nil, nil
	}
	rt, cancel, err := s.newRuntime(ctx, helpers)
	if err != nil {
		return nil, err
	}

	set := &HelperSet{
		Helpers: map[string]HelperFn{},
		close:   cancel,
	}

	globals := rt.GlobalObject()
	for _, key := range globals.Keys() {
		v := globals.Get(key)
		if v == nil {
			continue
		}
		cb, ok := goja.AssertFunction(v)
		if !ok {
			continue
		}
		// Detect declared arity (Function.length). Defaults to 1 if unreadable.
		arity := 1
		if obj := v.ToObject(rt); obj != nil {
			if lv := obj.Get("length"); lv != nil {
				arity = int(lv.ToInteger())
			}
		}
		name := key
		callable := cb
		set.Helpers[name] = HelperFn{
			Name:  name,
			Arity: arity,
			Call: func(args ...any) (any, error) {
				jsArgs := make([]goja.Value, len(args))
				for i, a := range args {
					jsArgs[i] = rt.ToValue(a)
				}
				res, err := callable(goja.Undefined(), jsArgs...)
				if err != nil {
					return nil, err
				}
				if res == nil {
					return nil, nil
				}
				return res.Export(), nil
			},
		}
	}

	return set, nil
}

// RunBeforeRender executes the `beforeRender(req, res, done)` style script on
// the given request, returning the mutated data object. Single-shot — VM is
// disposed before this returns.
func (s *Sandbox) RunBeforeRender(ctx context.Context, script string, request map[string]any) (map[string]any, error) {
	if script == "" {
		return request, nil
	}
	rt, cancel, err := s.newRuntime(ctx, script)
	if err != nil {
		return nil, err
	}
	defer cancel()

	fn := rt.Get("beforeRender")
	cb, ok := goja.AssertFunction(fn)
	if !ok {
		// Script may simply mutate `req` at the top level. Return it as-is.
		return request, nil
	}

	reqVal := rt.ToValue(request)
	resVal := rt.ToValue(map[string]any{})
	if _, err := cb(goja.Undefined(), reqVal, resVal); err != nil {
		return nil, err
	}
	return mapFromValue(rt, reqVal), nil
}

func (s *Sandbox) newRuntime(parent context.Context, script string) (*goja.Runtime, func(), error) {
	rt := goja.New()
	rt.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))

	jctx, cancel := context.WithTimeout(parent, s.timeout)
	stopped := make(chan struct{})
	go func() {
		select {
		case <-jctx.Done():
			rt.Interrupt(ErrTimeout)
		case <-stopped:
		}
	}()

	if _, err := rt.RunString(script); err != nil {
		close(stopped)
		cancel()
		return nil, nil, fmt.Errorf("sandbox: %w", err)
	}

	return rt, func() {
		close(stopped)
		cancel()
	}, nil
}

func mapFromValue(rt *goja.Runtime, v goja.Value) map[string]any {
	if v == nil {
		return nil
	}
	obj := v.ToObject(rt)
	if obj == nil {
		return nil
	}
	out := map[string]any{}
	for _, k := range obj.Keys() {
		out[k] = obj.Get(k).Export()
	}
	return out
}
