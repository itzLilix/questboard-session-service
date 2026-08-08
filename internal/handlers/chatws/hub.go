package chatws

import "github.com/rs/zerolog/log"

// Hub keeps track of connected clients grouped by chat room (chatID) and
// fans out messages to every client in a room. One Hub per process; start
// it once with `go hub.Run()` at boot.
type Hub struct {
	rooms map[string]map[*Client]bool // chatID -> set of clients

	register   chan *Client
	unregister chan *Client
	broadcast  chan *RoomMessage
}

// RoomMessage is an outbound payload destined for every client in a room.
type RoomMessage struct {
	ChatID  string
	Payload []byte // pre-marshaled JSON
}

func NewHub() *Hub {
	return &Hub{
		rooms:      make(map[string]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *RoomMessage, 256),
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
					log.Warn().Str("user_id", c.UserID).Str("chat_id", c.ChatID).Msg("chatws: dropping slow client")
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
