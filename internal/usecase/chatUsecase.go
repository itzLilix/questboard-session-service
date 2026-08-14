package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/itzLilix/questboard-session-service/internal/entities"
	"github.com/itzLilix/questboard-session-service/internal/infrastructure"
	"github.com/itzLilix/questboard-shared/dtos"
)

type chatUsecase struct{
	chatRepo ChatRepository
	sessionRepo SessionRepository
}

func NewChatUsecase(chatRepo ChatRepository, sessionRepo SessionRepository) *chatUsecase { return &chatUsecase{chatRepo: chatRepo, sessionRepo: sessionRepo} }

type ListChatsInput struct {
	RoomID string
	Scope  dtos.SessionType
}

type AttachmentInput struct {
	FileName  string `json:"fileName"`
	URL       string `json:"url"`
	MIMEType  string `json:"mimeType,omitempty"`
	SizeBytes int64  `json:"sizeBytes,omitempty"`
}

type SendMessageInput struct {
	ChatID        string
	Body          string  
	ReplyToID     *string 
	Attachments   []dtos.AttachmentInput
	MentionedUserIDs []string
}

type EditMessageInput struct {
	ChatID      	 string
	MessageID     	 string
	Body          	 string
	MentionedUserIDs []string
}

func (uc *chatUsecase) CanAccessChat(ctx context.Context, chatID string, v *entities.Viewer) (bool, error) {return true, nil}

func (uc *chatUsecase) ListForSession(ctx context.Context, sessionID string, v *entities.Viewer) ([]dtos.ChatSummary, error) {
	if !v.IsAuthenticated() {
		return nil, ErrUnauthorized
	}
	if sessionID == "" {
		return nil, fmt.Errorf("%w: missing session id", ErrInvalidData)
	}
	
	session, err := uc.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, mapRepoErr("get session", err)
	}
	
	if !v.CanActAs(session.MasterID) {
		isPlayer, err := uc.sessionRepo.IsPlayer(ctx, sessionID, v.UserID)
		if err != nil {
			return nil, mapRepoErr("check session membership", err)
		}
		if !isPlayer {
			return nil, ErrUnauthorized
		}
	}

	items, err := uc.chatRepo.ListSessionChats(ctx, sessionID, v.UserID)
	if err != nil {
		return nil, mapRepoErr("list session chats", err)
	}
	if items == nil {
		items = []dtos.ChatSummary{}
	}
	return items, nil
}

const (
	defaultMessagesLimit = 50
	maxMessagesLimit     = 100
)

type ListMessagesInput struct {
	ChatID string
	Before *string
	Limit  int
}

func (uc *chatUsecase) ListMessages(ctx context.Context, in *ListMessagesInput, v *entities.Viewer) (*dtos.MessagePage, error) {
	if !v.IsAuthenticated() {
		return nil, ErrUnauthorized
	}
	if in.Before != nil {
		if _, err := uuid.Parse(*in.Before); err != nil {
			return nil, ErrInvalidData
		}
	}

	allowed, err := uc.CanAccessChat(ctx, in.ChatID, v)
	if err != nil {
		return nil, fmt.Errorf("check chat access: %w", err)
	}
	if !allowed {
		return nil, ErrForbidden
	}

	limit := in.Limit
	switch {
	case limit <= 0:
		limit = defaultMessagesLimit
	case limit > maxMessagesLimit:
		limit = maxMessagesLimit
	}

	// Fetch one extra row to know whether there's a next page without a
	// separate COUNT query.
	rows, err := uc.chatRepo.ListMessages(ctx, &infrastructure.ListMessagesParams{
		ChatID: in.ChatID,
		Before: in.Before,
		Limit:  limit + 1,
	})
	if err != nil {
		return nil, mapRepoErr("list messages", err)
	}

	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}

	page := &dtos.MessagePage{Messages: rows, HasMore: hasMore}
	if hasMore && len(rows) > 0 {
		last := rows[len(rows)-1].ID
		page.NextCursor = &last
	}
	return page, nil
}

func (uc *chatUsecase) SendMessage(ctx context.Context, in SendMessageInput, v *entities.Viewer) (*dtos.MessagePayload, error) {
	if !v.IsAuthenticated() {
		return nil, ErrUnauthorized
	}
	if strings.TrimSpace(in.Body) == "" && len(in.Attachments) == 0 {
		return nil, ErrInvalidData
	}

	canSend, err := uc.chatRepo.GetSendPermission(ctx, in.ChatID, v.UserID)
	if err != nil {
		return nil, mapRepoErr("get send permission", err)
	}
	if !canSend {
		return nil, ErrForbidden
	}

	var reply *dtos.ReplySnippet
	if in.ReplyToID != nil && *in.ReplyToID != "" {
		reply, err = uc.chatRepo.GetReplySnippet(ctx, in.ChatID, *in.ReplyToID)
		if err != nil {
			return nil, mapRepoErr("get reply snippet", err)
		}
	}

	// TODO(Phase 5 — file uploads): once uploads have their own table,
	// verify each in.Attachments[i].URL belongs to a completed upload by
	// v.UserID before persisting it below. Right now attachment metadata
	// is trusted as-is from the client.

	newMsg := infrastructure.SaveMessageParams{
		ChatID:           in.ChatID,
		SenderID:         v.UserID,
		Body:             in.Body,
		Attachments:      in.Attachments,
		MentionedUserIDs: in.MentionedUserIDs,
	}
	if in.ReplyToID != nil {
		newMsg.ReplyTo = reply
	}

	payload, err := uc.chatRepo.SaveMessage(ctx, &newMsg)
	if err != nil {
		return nil, mapRepoErr("save message", err)
	}
	return payload, nil
}

func (uc *chatUsecase) EditMessage(ctx context.Context, in EditMessageInput, v *entities.Viewer) (*dtos.MessagePayload, error) {return nil, nil}
func (uc *chatUsecase) SetPinned(ctx context.Context, chatID, messageID string, pinned bool, v *entities.Viewer) (*dtos.PinPayload, error) {return nil, nil}
func (uc *chatUsecase) MarkRead(ctx context.Context, chatID, lastReadMessageID string, v *entities.Viewer) (*dtos.ReadPayload, error) {return nil, nil}
func (uc *chatUsecase) ListReadState(ctx context.Context, chatID string, v *entities.Viewer) ([]dtos.ReadPayload, error) {return nil, nil}

func (uc *chatUsecase) GetPermissions(ctx context.Context, chatID string, v *entities.Viewer) (*dtos.ChatPermissions, error) {
	if !v.IsAuthenticated() {
		return nil, ErrUnauthorized
	}
	perms, err := uc.chatRepo.GetPermissions(ctx, chatID, v.UserID)
	if err != nil {
		return nil, mapRepoErr("get permissions", err)
	}
	return perms, nil
}