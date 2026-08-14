package chatws

import "github.com/rs/zerolog"

// Hub keeps track of connected clients grouped by chat room (chatID) and
// fans out messages to every client in a room. One Hub per process; start
// it once with `go hub.Run()` at boot.
type Hub struct {
	rooms map[string]map[*Client]bool // chatID -> set of clients

	register   chan *Client
	unregister chan *Client
	broadcast  chan *RoomMessage

	log zerolog.Logger
}

// RoomMessage is an outbound payload destined for every client in a room.
type RoomMessage struct {
	ChatID  string
	Payload []byte // pre-marshaled JSON
}

func NewHub(log zerolog.Logger) *Hub {
	return &Hub{
		rooms:      make(map[string]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *RoomMessage, 256),
		log:        log,
	}
}

// Run owns all room state, so every mutation goes through its channels —
// no mutex needed as long as nothing else touches h.rooms directly.
func (h *Hub) Run() {
	for {
		select {
		case c := <-h.register:
			if h.rooms[c.ChatID] == nil {
				h.rooms[c.ChatID] = make(map[*Client]bool)
			}
			h.rooms[c.ChatID][c] = true

		case c := <-h.unregister:
			if clients, ok := h.rooms[c.ChatID]; ok {
				if _, ok := clients[c]; ok {
					delete(clients, c)
					close(c.send)
					if len(clients) == 0 {
						delete(h.rooms, c.ChatID)
					}
				}
			}

		case m := <-h.broadcast:
			for c := range h.rooms[m.ChatID] {
				select {
				case c.send <- m.Payload:
				default:
					// Slow consumer: don't let one stuck client block the
					// whole room's broadcast. Drop it; readPump/writePump
					// will notice the closed channel and clean up.
					h.log.Warn().Str("user_id", c.viewer.UserID).Str("chat_id", c.ChatID).Msg("chatws: dropping slow client")
					go func(c *Client) { h.unregister <- c }(c)
				}
			}
		}
	}
}

// Publish lets code outside this package (e.g. a REST endpoint that also
// writes chat messages, or a system/event message) push into a room
// without reaching into the channel directly.
func (h *Hub) Publish(chatID string, payload []byte) {
	h.broadcast <- &RoomMessage{ChatID: chatID, Payload: payload}
}

// PublishEvent is the one-call version of Publish for REST handlers: it
// wraps payload in the OutgoingEvent envelope and publishes it. This is
// what a mutating REST endpoint should call right after its usecase call
// succeeds — call it once per request, and only from the transport layer
// that received the request, never from inside the usecase itself (a
// websocket-originated change already broadcasts via Client, so if the
// usecase also published, it would double-broadcast for that path).
func (h *Hub) PublishEvent(chatID string, eventType Event, payload any) error {
	full, err := MarshalEvent(chatID, eventType, payload)
	if err != nil {
		h.log.Error().Err(err).Str("chat_id", chatID).Msg("chatws: failed to marshal event")
		return err
	}
	h.Publish(chatID, full)
	return nil
}