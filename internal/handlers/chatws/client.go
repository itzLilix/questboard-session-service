package chatws

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/contrib/v3/websocket"
	"github.com/itzLilix/questboard-session-service/internal/entities"
	"github.com/itzLilix/questboard-session-service/internal/usecase"
	"github.com/itzLilix/questboard-shared/dtos"
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
	uc     Usecase
	conn   *websocket.Conn
	send   chan []byte // outbound queue; only the hub/Client write to it, only writePump reads it
	ChatID string
	viewer *entities.Viewer

	ctx context.Context
	cancel context.CancelFunc
}

// ServeClient upgrades a *websocket.Conn into a registered Client and
// blocks for the connection's lifetime. Call it from inside the handler
// you pass to websocket.New. Access to the chat itself is already gated
// by PermissionChecker.CanAccessChat before the upgrade happens (see
// handler.go) — per-action authorization (who may pin, who may edit what)
// lives inside Persister's usecase implementation instead, same as the
// REST handlers; Client doesn't need its own copy of those rules.
func ServeClient(hub *Hub, uc Usecase, conn *websocket.Conn, chatID string, viewer *entities.Viewer) {
	ctx, cancel := context.WithCancel(context.Background())
	c := &Client{
		hub:    hub,
		uc:  uc,
		conn:   conn,
		send:   make(chan []byte, 32),
		ChatID: chatID,
		viewer: viewer,
		ctx: ctx,
		cancel: cancel,
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
	if states, err := uc.ListReadState(c.ctx, chatID, viewer); err != nil {
		c.hub.log.Warn().Err(err).Str("chat_id", chatID).Msg("chatws: failed to load initial read state")
	} else {
		for _, s := range states {
			c.sendDirect(EventRead, s)
		}
	}

	c.readPump()
}

func (c *Client) readPump() {
	defer func() {
		c.cancel()
		c.hub.unregister <- c
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
				c.hub.log.Error().Err(err).Str("user_id", c.viewer.UserID).Str("chat_id", c.ChatID).Msg("chatws: read error")
			}
			return
		}

		switch in.Type {
		case EventMessage:
			c.handleMessage(in)
		case EventEdit:
			c.handleEdit(in)
		case EventPin:
			c.handlePin(in, true)
		case EventUnpin:
			c.handlePin(in, false)
		case EventRead:
			c.handleRead(in)
		case EventDelete:
			c.handleDelete(in)
		default:
			// unknown/future type — ignore rather than kill the connection
		}
	}
}

func (c *Client) handleMessage(in IncomingMessage) {
	if in.Body == "" && len(in.Attachments) == 0 {
		return // no bare-empty messages
	}

	var replyToID *string
	if in.ReplyToID != "" {
		replyToID = &in.ReplyToID
	}

	payload, err := c.uc.SendMessage(c.ctx, usecase.SendMessageInput{
		ChatID:           c.ChatID,
		Body:             in.Body,
		ReplyToID:        replyToID,
		Attachments:      in.Attachments,
		MentionedUserIDs: in.MentionedUserIDs,
	}, c.viewer)
	if err != nil {
		// Rejected (bad reply target, attachment not yours, etc). Consider
		// echoing a private "type": "error" event back to just this
		// client via c.sendDirect instead of silently dropping, once you
		// have an error-envelope shape you're happy with.
		c.hub.log.Warn().Err(err).Str("user_id", c.viewer.UserID).Str("chat_id", c.ChatID).Msg("chatws: reject message")
		return
	}
	payload.ClientNonce = in.ClientNonce

	c.broadcastEvent(EventMessage, payload)

	// Extension point for notifications (badges + push, eventually): once
	// you have a notifications table, create rows here for chat members
	// who aren't this sender — MarkRead below is the natural place to
	// clear them again when a recipient catches up on this chat.
}

func (c *Client) handleEdit(in IncomingMessage) {
	if in.MessageID == "" || in.Body == "" {
		return // edits must still leave content — no editing into an empty message
	}

	payload, err := c.uc.EditMessage(c.ctx, usecase.EditMessageInput{
		ChatID:           c.ChatID,
		MessageID:        in.MessageID,
		Body:             in.Body,
		MentionedUserIDs: in.MentionedUserIDs,
	}, c.viewer)
	if err != nil {
		c.hub.log.Warn().Err(err).Str("user_id", c.viewer.UserID).Str("chat_id", c.ChatID).Str("message_id", in.MessageID).Msg("chatws: reject edit")
		return
	}

	c.broadcastEvent(EventEdit, payload)
}

func (c *Client) handlePin(in IncomingMessage, pin bool) {
	if in.MessageID == "" {
		return
	}

	payload, err := c.uc.SetPinned(c.ctx, c.ChatID, in.MessageID, pin, c.viewer)
	if err != nil {
		// Covers both "not found" and "not allowed" — SetPinned's usecase
		// enforces who may pin, same as SaveMessage/EditMessage already
		// enforce their own rules. No separate check here, to match.
		c.hub.log.Warn().Err(err).Str("user_id", c.viewer.UserID).Str("chat_id", c.ChatID).Str("message_id", in.MessageID).Msg("chatws: reject pin")
		return
	}

	eventType := EventUnpin
	if pin {
		eventType = EventPin
	}
	c.broadcastEvent(eventType, payload)
}

func (c *Client) handleRead(in IncomingMessage) {
	if in.MessageID == "" {
		return
	}

	payload, err := c.uc.MarkRead(c.ctx, c.ChatID, in.MessageID, c.viewer)
	if err != nil {
		c.hub.log.Error().Err(err).Str("user_id", c.viewer.UserID).Str("chat_id", c.ChatID).Msg("chatws: mark-read failed")
		return
	}

	// Broadcasting read state to the whole room is what makes live
	// seen-by lists possible on the client side (see MarkRead doc) —
	// worth keeping at this member scale, unlike in a large chat.
	c.broadcastEvent(EventRead, payload)
}

func (c *Client) handleDelete(in IncomingMessage) {
	fmt.Println(in)
	if in.MessageID == "" {
		return
	}

	err := c.uc.DeleteMessage(c.ctx, c.ChatID, in.MessageID, c.viewer)
	if err != nil {
		c.hub.log.Error().Err(err).Str("user_id", c.viewer.UserID).Str("chat_id", c.ChatID).Msg("chatws: delete failed")
		return
	}

	c.broadcastEvent(EventDelete, &dtos.DeletePayload{MessageID: in.MessageID})
}

func (c *Client) broadcastEvent(eventType Event, payload any) {
	if err := c.hub.PublishEvent(c.ChatID, eventType, payload); err != nil {
		c.hub.log.Error().Err(err).Msg("chatws: marshal event error")
	}
}

// sendDirect writes only to this client's own queue — used for state that
// doesn't need to (or shouldn't) go to the rest of the room, like the
// initial read-state seed on connect.
func (c *Client) sendDirect(eventType Event, payload any) {
	full, err := MarshalEvent(c.ChatID, eventType, payload)
	if err != nil {
		c.hub.log.Error().Err(err).Msg("chatws: marshal event error")
		return
	}
	select {
	case c.send <- full:
	default:
		c.hub.log.Warn().Str("user_id", c.viewer.UserID).Str("chat_id", c.ChatID).Msg("chatws: dropped event, send buffer full")
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		if r := recover(); r != nil {
			c.hub.log.Error().Interface("panic", r).Str("user_id", c.viewer.UserID).Str("chat_id", c.ChatID).Msg("chatws: writePump panic recovered")
		}
		ticker.Stop()
		c.cancel()
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
