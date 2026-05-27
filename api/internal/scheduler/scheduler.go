// Package scheduler loads enabled schedules from the DB and runs them through
// the renderer at the configured cron times.
package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/cloudreport/api/internal/render"
	"github.com/cloudreport/api/internal/store"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog/log"
)

type Scheduler struct {
	db       *store.Store
	renderer *render.Renderer
	c        *cron.Cron
	mu       sync.Mutex
	entries  map[string]cron.EntryID
}

func New(db *store.Store, r *render.Renderer) *Scheduler {
	return &Scheduler{
		db: db, renderer: r,
		c:       cron.New(),
		entries: map[string]cron.EntryID{},
	}
}

func (s *Scheduler) Start(ctx context.Context) error {
	s.c.Start()
	return s.Reload(ctx)
}

func (s *Scheduler) Stop() {
	if s.c != nil {
		s.c.Stop()
	}
}

func (s *Scheduler) Reload(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for shortid, id := range s.entries {
		s.c.Remove(id)
		delete(s.entries, shortid)
	}
	spec := store.EntitySpecs["schedules"]
	items, err := s.db.ListGeneric(ctx, spec, "enabled = true", nil, "", 1000, 0)
	if err != nil {
		return err
	}
	for _, it := range items {
		expr, _ := it["cron"].(string)
		shortid, _ := it["shortid"].(string)
		tplShortid, _ := it["template_shortid"].(string)
		if expr == "" || tplShortid == "" {
			continue
		}
		id, err := s.c.AddFunc(expr, s.runJob(tplShortid, shortid))
		if err != nil {
			log.Warn().Err(err).Str("cron", expr).Msg("invalid cron")
			continue
		}
		s.entries[shortid] = id
	}
	log.Info().Int("schedules", len(s.entries)).Msg("scheduler loaded")
	return nil
}

func (s *Scheduler) runJob(templateShortid, scheduleShortid string) func() {
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		req := &render.Request{}
		req.Template.Shortid = templateShortid
		req.Data = json.RawMessage("{}")
		_, err := s.renderer.Render(ctx, req, nil)
		if err != nil {
			log.Error().Err(err).Str("schedule", scheduleShortid).Msg("scheduled render failed")
		}
	}
}

// NextRun computes the next time a cron expression will fire.
func NextRun(expr string) (time.Time, error) {
	sched, err := cron.ParseStandard(expr)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse cron: %w", err)
	}
	return sched.Next(time.Now()), nil
}
