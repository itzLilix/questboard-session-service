package chatws

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gofiber/contrib/v3/websocket"
	"github.com/rs/zerolog/log"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10 // must stay under pongWait
	maxMessageSize = 16 * 1024           // envelope only (attachments are refs, not bytes) — tune as needed
)

// Client is a single websocket connection bound to one chat room.
type Client struct {
	hub    *Hub
	store  Persister
	perms   PermissionChecker // nil is fine — see PermissionChecker doc
	conn   *websocket.Conn
	send   chan []byte // outbound queue; only the hub/Client write to it, only writePump reads it
	ChatID string
	UserID string
}

// ServeClient upgrades a *websocket.Conn into a registered Client and
// blocks for the connection's lifetime. Call it from inside the handler
// you pass to websocket.New.
func ServeClient(hub *Hub, store Persister, perms PermissionChecker, conn *websocket.Conn, chatID, userID string) {
	c := &Client{
		hub:    hub,
		store:  store,
		perms:   perms,
		conn:   conn,
		send:   make(chan []byte, 32),
		ChatID: chatID,
		UserID: userID,
	}

	hub.register <- c

	// writePump is the connection's sole writer; readPump (this goroutine,
	// entered below) is its sole reader — fasthttp/websocket (like
	// gorilla) only supports one concurrent reader and one concurrent
	// writer per Conn. Must be running before we seed initial state below,
	// since that writes to c.send too and the buffer isn't guaranteed to
	// fit every member's watermark unread.
	go c.writePump()

	// Seed this connection with the room's current read state before it
	// sees any live "read" broadcasts — unicast to just this client, not
	// hub.Publish, since everyone else already has this state.
	if states, err := store.ListReadState(context.Background(), chatID); err != nil {
		log.Warn().Err(err).Str("chat_id", chatID).Msg("chatws: failed to load initial read state")
	} else {
		for _, s := range states {
			c.sendDirect("read", s)
		}
	}

	c.readPump()
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		var in IncomingMessage
		if err := c.conn.ReadJSON(&in); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Error().Err(err).Str("user_id", c.UserID).Str("chat_id", c.ChatID).Msg("chatws: read error")
			}
			return
		}

		switch in.Type {
		case "message":
			c.handleMessage(in)
		case "pin":
			c.handlePin(in, true)
		case "unpin":
			c.handlePin(in, false)
		case "read":
			c.handleRead(in)
		default:
			// unknown/future type — ignore rather than kill the connection
		}
	}
}

func (c *Client) handleMessage(in IncomingMessage) {
	if in.Content == "" && len(in.AttachmentIDs) == 0 {
		return // no bare-empty messages
	}

	payload, err := c.store.SaveMessage(context.Background(), NewMessage{
		ChatID:           c.ChatID,
		SenderID:         c.UserID,
		Content:          in.Content,
		ReplyToID:        in.ReplyToID,
		AttachmentIDs:    in.AttachmentIDs,
		MentionedUserIDs: in.MentionedUserIDs,
	})
	if err != nil {
		// Rejected (bad reply target, attachment not yours, etc). Consider
		// echoing a private "type": "error" event back to just this
		// client via c.sendDirect instead of silently dropping, once you
		// have an error-envelope shape you're happy with.
		log.Warn().Err(err).Str("user_id", c.UserID).Str("chat_id", c.ChatID).Msg("chatws: reject message")
		return
	}
	payload.ClientNonce = in.ClientNonce

	c.broadcastEvent("message", payload)

	// Extension point for notifications (badges + push, eventually): once
	// you have a notifications table, create rows here for chat members
	// who aren't this sender — MarkRead below is the natural place to
	// clear them again when a recipient catches up on this chat.
}

func (c *Client) handlePin(in IncomingMessage, pin bool) {
	if in.MessageID == "" {
		return
	}

	if c.perms != nil {
		allowed, err := c.perms.CanPin(context.Background(), c.ChatID, c.UserID)
		if err != nil || !allowed {
			return
		}
	}

	payload, err := c.store.SetPinned(context.Background(), c.ChatID, in.MessageID, c.UserID, pin)
	if err != nil {
		log.Error().Err(err).Str("chat_id", c.ChatID).Str("message_id", in.MessageID).Msg("chatws: pin update failed")
		return
	}

	eventType := "unpin"
	if pin {
		eventType = "pin"
	}
	c.broadcastEvent(eventType, payload)
}

func (c *Client) handleRead(in IncomingMessage) {
	if in.MessageID == "" {
		return
	}

	payload, err := c.store.MarkRead(context.Background(), c.ChatID, c.UserID, in.MessageID)
	if err != nil {
		log.Error().Err(err).Str("user_id", c.UserID).Str("chat_id", c.ChatID).Msg("chatws: mark-read failed")
		return
	}

	// Broadcasting read state to the whole room is what makes live
	// seen-by lists possible on the client side (see MarkRead doc) —
	// worth keeping at this member scale, unlike in a large chat.
	c.broadcastEvent("read", payload)
}

// marshalEvent wraps payload in the OutgoingEvent envelope. Shared by
// broadcastEvent (hub -> whole room) and sendDirect (this connection only).
func (c *Client) marshalEvent(eventType string, payload any) ([]byte, bool) {
	raw, err := json.Marshal(payload)
	if err != nil {
		log.Error().Err(err).Msg("chatws: marshal payload error")
		return nil, false
	}
	full, err := json.Marshal(OutgoingEvent{Type: eventType, ChatID: c.ChatID, Payload: raw})
	if err != nil {
		log.Error().Err(err).Msg("chatws: marshal event error")
		return nil, false
	}
	return full, true
}

func (c *Client) broadcastEvent(eventType string, payload any) {
	if full, ok := c.marshalEvent(eventType, payload); ok {
		c.hub.Publish(c.ChatID, full)
	}
}

// sendDirect writes only to this client's own queue — used for state that
// doesn't need to (or shouldn't) go to the rest of the room, like the
// initial read-state seed on connect.
func (c *Client) sendDirect(eventType string, payload any) {
	full, ok := c.marshalEvent(eventType, payload)
	if !ok {
		return
	}
	select {
	case c.send <- full:
	default:
		log.Warn().Str("user_id", c.UserID).Str("chat_id", c.ChatID).Msg("chatws: dropped event, send buffer full")
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// hub closed the channel (unregistered) — say goodbye and stop
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}

		case <-ticker.C:
			// fasthttp doesn't enforce idle timeouts for you — app-level
			// pings so dead connections get noticed instead of leaking.
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
