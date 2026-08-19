package chatws

import (
	"context"

	"github.com/itzLilix/questboard-session-service/internal/entities"
	"github.com/itzLilix/questboard-session-service/internal/usecase"
	"github.com/itzLilix/questboard-shared/dtos"
)

// Persister saves chat messages, edits, pin state, and read state.
// Implement against your existing messages/attachments repo.
type Usecase interface {
	// SaveMessage persists a new message (plus its message_attachments /
	// message_mentions rows) and returns the fully resolved payload to
	// broadcast. Two things the schema doesn't enforce for you, so check
	// them here:
	//   - reply_to_id has no constraint tying it to the same chat_id —
	//     messages.chat_id is NOT NULL, which rules out a composite FK
	//     with ON DELETE SET NULL the way pinned_messages uses one — so
	//     without an explicit check, a reply could point into a message
	//     in a different chat entirely.
	//   - Attachments arrive as client-supplied URLs, not IDs into a
	//     table you already trust; confirm each URL actually belongs to
	//     a completed upload by this sender before inserting it.
	// Return an error — not a partial message — on either failure;
	// readPump treats any error as a rejected send.
	SendMessage(ctx context.Context, in usecase.SendMessageInput, v *entities.Viewer) (*dtos.MessagePayload, error)

	// EditMessage updates an existing message's content (and mentions) and
	// returns the resolved payload to broadcast. Unlike pinning, this is
	// a plain ownership check rather than a role check — reject unless
	// actorID matches the message's sender_id — so it belongs here next
	// to the row rather than behind PermissionChecker. Reject if the
	// message has deleted_at set.
	EditMessage(ctx context.Context, in usecase.EditMessageInput, v *entities.Viewer) (*dtos.MessagePayload, error)

	// SetPinned pins or unpins an existing message and returns the
	// resulting pin state to broadcast.
	SetPinned(ctx context.Context, chatID, messageID string, pinned bool, v *entities.Viewer) (*dtos.PinPayload, error)

	// MarkRead upserts the caller's read watermark for a chat — one row
	// per (chatID, userID), not one per message. Should be a no-op (or
	// simply not lower the watermark) if lastReadMessageID is older than
	// what's already stored, in case events arrive out of order.
	// Consider also clearing any (chatID, userID) notification rows up
	// to lastReadMessageID here, once notifications exist — keeps badge
	// state and in-chat read state from drifting apart.
	MarkRead(ctx context.Context, chatID, lastReadMessageID string, v *entities.Viewer) (*dtos.ReadPayload, error)

	// ListReadState returns the current read watermark for every member
	// of the chat. Called once when a client connects, to seed its local
	// "who's read what" roster before any live "read" events arrive —
	// "seen by" for a given message is then just: every member whose
	// watermark is at or past that message in the chat's ordering.
	ListReadState(ctx context.Context, chatID string, v *entities.Viewer) ([]dtos.ReadPayload, error)

	// CanAccessChat gates the websocket upgrade itself — the
	// IsCampaignMember/IsPlayer + chat_members-fallback rules already
	// built for this app, reduced to one yes/no for a given chatID.
	CanAccessChat(ctx context.Context, chatID string, v *entities.Viewer) (bool, error)

	DeleteMessage(ctx context.Context, chatID, messageID string, v *entities.Viewer) (error)
}
