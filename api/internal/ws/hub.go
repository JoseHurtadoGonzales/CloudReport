// Package ws implements a fan-out WebSocket hub for entity-change events.
// Clients connect to /ws and receive a stream of JSON messages of the form:
//
//   { "type": "entityChange", "entitySet": "templates", "action": "update",
//     "shortid": "abc", "userId": "...", "at": "RFC3339" }
//
// The hub also broadcasts render-lifecycle events (running/success/error).
package ws

import (
	"encoding/json"
	"sync"
	"time"

	cws "github.com/gofiber/contrib/websocket"
	"github.com/rs/zerolog/log"
)

type Event struct {
	Type      string `json:"type"`
	EntitySet string `json:"entitySet,omitempty"`
	Action    string `json:"action,omitempty"`
	Shortid   string `json:"shortid,omitempty"`
	UserID    string `json:"userId,omitempty"`
	At        string `json:"at"`
	Extra     any    `json:"extra,omitempty"`
}

type Hub struct {
	mu      sync.Mutex
	clients map[*cws.Conn]struct{}
}

func NewHub() *Hub {
	return &Hub{clients: map[*cws.Conn]struct{}{}}
}

func (h *Hub) Register(c *cws.Conn) {
	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()
}

func (h *Hub) Unregister(c *cws.Conn) {
	h.mu.Lock()
	delete(h.clients, c)
	h.mu.Unlock()
}

// Broadcast sends the event to every connected client. Failures drop the
// client silently.
func (h *Hub) Broadcast(e Event) {
	if e.At == "" {
		e.At = time.Now().UTC().Format(time.RFC3339Nano)
	}
	payload, err := json.Marshal(e)
	if err != nil {
		log.Warn().Err(err).Msg("ws marshal")
		return
	}
	h.mu.Lock()
	clients := make([]*cws.Conn, 0, len(h.clients))
	for c := range h.clients {
		clients = append(clients, c)
	}
	h.mu.Unlock()
	for _, c := range clients {
		if err := c.WriteMessage(cws.TextMessage, payload); err != nil {
			h.Unregister(c)
			_ = c.Close()
		}
	}
}

// Emit is a convenience helper used by handlers after a DB mutation.
func (h *Hub) Emit(entitySet, action, shortid, userID string) {
	h.Broadcast(Event{
		Type: "entityChange", EntitySet: entitySet, Action: action,
		Shortid: shortid, UserID: userID,
	})
}
