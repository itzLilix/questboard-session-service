package chatws

import (
	"encoding/json"

	"github.com/itzLilix/questboard-shared/dtos"
)

// IncomingMessage is the envelope clients send us over the socket.
type IncomingMessage struct {
	Type Event `json:"type"` // "message" | "edit" | "pin" | "unpin" | "read"

	// type == "message" | "edit" (edit reuses Body/MentionedUserIDs;
	// ReplyToID/Attachments/ClientNonce are set only when Type == "message")
	Body             string                 `json:"body,omitempty"`
	ReplyToID        string                 `json:"replyToId,omitempty"`
	Attachments      []dtos.AttachmentInput `json:"attachments,omitempty"`
	MentionedUserIDs []string               `json:"mentionedUserIds,omitempty"`
	ClientNonce      string                 `json:"clientNonce,omitempty"` // echoed back so the client can reconcile its optimistic render

	// type == "edit" | "pin" | "unpin" | "read" — for "read", this is the
	// last message being marked read, not a target to mutate
	MessageID string `json:"messageId,omitempty"`
}

// OutgoingEvent is the generic envelope broadcast to a room. Payload shape
// depends on Type — clients switch on Type before unmarshaling Payload
// into the matching DTO from questboard-shared/dtos:
//
//	EventMessage | EventEdit -> dtos.MessagePayload
//	EventPin | EventUnpin    -> dtos.PinPayload
//	EventRead                -> dtos.ReadPayload
type OutgoingEvent struct {
	Type    Event           `json:"type"`
	ChatID  string          `json:"chatId"`
	Payload json.RawMessage `json:"payload"`
}

// Event is the closed set of wire event types.
type Event string

const (
	EventMessage Event = "message"
	EventEdit    Event = "edit"
	EventPin     Event = "pin"
	EventUnpin   Event = "unpin"
	EventRead    Event = "read"
	EventDelete  Event = "delete"
)

// MarshalEvent builds the OutgoingEvent envelope bytes for a payload.
// Exported so callers outside this package — REST handlers publishing a
// change made outside the socket — build the exact same wire format
// instead of a second, possibly-drifting one. Client and Hub both use
// this internally too, so there's one place that owns the envelope shape.
func MarshalEvent(chatID string, eventType Event, payload any) ([]byte, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return json.Marshal(OutgoingEvent{Type: eventType, ChatID: chatID, Payload: raw})
}
