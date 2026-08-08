package usecase

import (
	"context"
	"fmt"

	"github.com/itzLilix/questboard-session-service/internal/entities"
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
	
	//fmt.Println("хуй")
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