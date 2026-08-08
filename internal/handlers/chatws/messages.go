package chatws

import (
	"context"
	"encoding/json"
	"time"
)

// IncomingMessage is the envelope clients send us over the socket.
type IncomingMessage struct {
	Type string `json:"type"` // "message" | "pin" | "unpin"

	// type == "message"
	Content          string   `json:"content,omitempty"`
	ReplyToID        string   `json:"reply_to_id,omitempty"`
	AttachmentIDs    []string `json:"attachment_ids,omitempty"`
	MentionedUserIDs []string `json:"mentioned_user_ids,omitempty"`
	ClientNonce      string   `json:"client_nonce,omitempty"` // echoed back so the client can reconcile its optimistic render

	// type == "pin" | "unpin"
	MessageID string `json:"message_id,omitempty"`
}

// OutgoingEvent is the generic envelope broadcast to a room. Payload shape
// depends on Type — clients switch on Type before unmarshaling Payload
// into the matching struct below.
type OutgoingEvent struct {
	Type    string          `json:"type"` // "message" | "pin" | "unpin"
	ChatID  string          `json:"chat_id"`
	Payload json.RawMessage `json:"payload"`
}

// MessagePayload is the Payload for a "message" event.
type MessagePayload struct {
	ID               string        `json:"id"`
	SenderID         string        `json:"sender_id"`
	Content          string        `json:"content"`
	ReplyTo          *ReplySnippet `json:"reply_to,omitempty"`
	Attachments      []Attachment  `json:"attachments,omitempty"`
	MentionedUserIDs []string      `json:"mentioned_user_ids,omitempty"`
	CreatedAt        time.Time     `json:"created_at"`
	ClientNonce      string        `json:"client_nonce,omitempty"`
}

// ReplySnippet is a small denormalized preview of the message being
// replied to, so clients can render "replying to: ..." without a second
// round trip. Resolved by Persister.SaveMessage, which already has to
// look up ReplyToID to validate it.
type ReplySnippet struct {
	MessageID      string `json:"message_id"`
	SenderID       string `json:"sender_id"`
	ContentPreview string `json:"content_preview"`
}

// Attachment describes a file already uploaded through your existing
// upload endpoint before the message referencing it was sent.
type Attachment struct {
	ID        string `json:"id"`
	FileName  string `json:"file_name"`
	URL       string `json:"url"`
	MIMEType  string `json:"mime_type"`
	SizeBytes int64  `json:"size_bytes"`
}

// PinPayload is the Payload for a "pin" or "unpin" event.
type PinPayload struct {
	MessageID string    `json:"message_id"`
	PinnedBy  string    `json:"pinned_by,omitempty"`
	PinnedAt  time.Time `json:"pinned_at,omitempty"`
}

// ReadPayload is the Payload for a "read" event — a watermark, not a
// per-message receipt. "User X has read up through message Y."
type ReadPayload struct {
	UserID            string    `json:"user_id"`
	LastReadMessageID string    `json:"last_read_message_id"`
	ReadAt            time.Time `json:"read_at"`
}

// NewMessage is what readPump hands to Persister after a "message" event
// comes in, before it's been validated or resolved.
type NewMessage struct {
	ChatID           string
	SenderID         string
	Content          string
	ReplyToID        string // empty if not a reply
	AttachmentIDs    []string
	MentionedUserIDs []string
}

// Persister saves chat messages and pin state. Implement against your
// existing messages/attachments repo.
type Persister interface {
	// SaveMessage persists a new message and returns the fully resolved
	// payload to broadcast (attachment metadata, reply snippet, etc).
	// Return an error — not a partial message — if ReplyToID or any
	// AttachmentID doesn't belong to this chat/sender; readPump treats
	// that as a rejected send.
	SaveMessage(ctx context.Context, msg NewMessage) (MessagePayload, error)

	// SetPinned pins or unpins an existing message and returns the
	// resulting pin state to broadcast.
	SetPinned(ctx context.Context, chatID, messageID, actorID string, pinned bool) (PinPayload, error)

	// MarkRead upserts the caller's read watermark for a chat — one row
	// per (chatID, userID), not one per message. Should be a no-op (or
	// simply not lower the watermark) if lastReadMessageID is older than
	// what's already stored, in case events arrive out of order.
	// Consider also clearing any (chatID, userID) notification rows up
	// to lastReadMessageID here, once notifications exist — keeps badge
	// state and in-chat read state from drifting apart.
	MarkRead(ctx context.Context, chatID, userID, lastReadMessageID string) (ReadPayload, error)

	// ListReadState returns the current read watermark for every member
	// of the chat. Called once when a client connects, to seed its local
	// "who's read what" roster before any live "read" events arrive —
	// "seen by" for a given message is then just: every member whose
	// watermark is at or past that message in the chat's ordering.
	ListReadState(ctx context.Context, chatID string) ([]ReadPayload, error)
}

// PermissionChecker gates who may pin/unpin in a chat — e.g. the session or
// campaign master. Kept separate from MembershipChecker since "can read
// this chat" and "can pin in this chat" are different questions. Pass nil
// to ServeClient/RegisterRoutes if you haven't decided on pin permissions
// yet — pinning is then allowed for anyone who can reach the chat.
type PermissionChecker interface {
	CanPin(ctx context.Context, chatID, userID string) (bool, error)
}
