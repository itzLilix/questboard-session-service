package handlers

import "github.com/rs/zerolog"

type Conn interface {
	WriteMessage(messageType int, data []byte) error
	ReadMessage() (messageType int, p []byte, err error)
	Close() error
}

type Event struct {
	Type   string `json:"type"` // e.g. "message.created", "typing", "read"
	ChatID string `json:"chatId"`
	Data   any    `json:"data,omitempty"`
}

type Client struct {
	userID string
	conn   Conn
	send   chan []byte
	chats  map[string]struct{}
}

type Hub struct {
	register   chan *Client
	unregister chan *Client
	broadcast  chan Event
	chats      map[string]map[*Client]struct{}
	log        zerolog.Logger
}

func NewHub(log zerolog.Logger) *Hub {
	return &Hub{
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan Event),
		chats:      make(map[string]map[*Client]struct{}),
		log:        log,
	}
}
