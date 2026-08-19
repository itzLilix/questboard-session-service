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

type ListMessagesInput struct {
	ChatID string
	Before *string
	Limit  int
}

const (
	defaultMessagesLimit = 50
	maxMessagesLimit     = 100
)

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

func (uc *chatUsecase) GetMessageById(ctx context.Context, chatID, messageId string, v *entities.Viewer) (*dtos.MessagePayload, error) {
	if !v.IsAuthenticated() {
		return nil, ErrUnauthorized
	}

	allowed, err := uc.CanAccessChat(ctx, chatID, v)
	if err != nil {
		return nil, fmt.Errorf("check chat access: %w", err)
	}
	if !allowed {
		return nil, ErrForbidden
	}

	payload, err := uc.chatRepo.GetMessageByID(ctx, chatID, messageId)
	if err != nil {
		return nil, mapRepoErr("get message", err)
	}
	return payload, nil
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

func (uc *chatUsecase) EditMessage(ctx context.Context, in EditMessageInput, v *entities.Viewer) (*dtos.MessagePayload, error) {
	if !v.IsAuthenticated() {
		return nil, ErrUnauthorized
	}
	if strings.TrimSpace(in.Body) == "" {
		return nil, ErrInvalidData
	}

	senderID, err := uc.chatRepo.GetMessageOwner(ctx, in.ChatID, in.MessageID)
	if err != nil {
		return nil, mapRepoErr("get message owner", err)
	}
	if senderID != v.UserID {
		return nil, ErrForbidden
	}

	payload, err := uc.chatRepo.EditMessage(ctx, &infrastructure.EditMessageParams{
		ChatID:           in.ChatID,
		MessageID:        in.MessageID,
		Body:             in.Body,
		MentionedUserIDs: in.MentionedUserIDs,
	})
	if err != nil {
		return nil, mapRepoErr("edit message", err)
	}
	return payload, nil
}

func (uc *chatUsecase) ListPinned(ctx context.Context, chatID string, v *entities.Viewer) ([]dtos.PinPayload, error) {
	if !v.IsAuthenticated() {
		return nil, ErrUnauthorized
	}

	allowed, err := uc.CanAccessChat(ctx, chatID, v)
	if err != nil {
		return nil, fmt.Errorf("check chat access: %w", err)
	}
	if !allowed {
		return nil, ErrForbidden
	}

	items, err := uc.chatRepo.ListPinned(ctx, chatID)
	if err != nil {
		return nil, mapRepoErr("list pinned messages", err)
	}
	if items == nil {
		items = []dtos.PinPayload{}
	}
	return items, nil
}

func (uc *chatUsecase) SetPinned(ctx context.Context, chatID, messageID string, pinned bool, v *entities.Viewer) (*dtos.PinPayload, error) {
	if !v.IsAuthenticated() {
		return nil, ErrUnauthorized
	}

	perms, err := uc.chatRepo.GetPermissions(ctx, chatID, v.UserID)
	if err != nil {
		return nil, mapRepoErr("get permissions", err)
	}
	if !perms.CanPinMessages {
		return nil, ErrForbidden
	}

	payload, err := uc.chatRepo.SetPinned(ctx, chatID, messageID, pinned, v.UserID)
	if err != nil {
		return nil, mapRepoErr("set pinned", err)
	}
	return payload, nil
}

func (uc *chatUsecase) DeleteMessage(ctx context.Context, chatID, messageID string, v *entities.Viewer) error {
	if !v.IsAuthenticated() {
		return ErrUnauthorized
	}

	senderID, err := uc.chatRepo.GetMessageOwner(ctx, chatID, messageID)
	if err != nil {
		return mapRepoErr("get message owner", err)
	}

	if senderID != v.UserID {
		perms, err := uc.chatRepo.GetPermissions(ctx, chatID, v.UserID)
		if err != nil {
			return mapRepoErr("get permissions", err)
		}
		if !perms.CanDeleteMessages {
			return ErrForbidden
		}
	}

	if err := uc.chatRepo.DeleteMessage(ctx, chatID, messageID); err != nil {
		return mapRepoErr("delete message", err)
	}
	return nil
}