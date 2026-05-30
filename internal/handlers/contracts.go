package handlers

import (
	"context"

	"github.com/itzLilix/questboard-session-service/internal/entities"
	uc "github.com/itzLilix/questboard-session-service/internal/usecase"
	"github.com/itzLilix/questboard-shared/dtos"
)

type CampaignUsecase interface {
	List(ctx context.Context, in uc.ListCampaignsInput, v *entities.Viewer) (dtos.Page[dtos.Campaign], error)
	GetByID(ctx context.Context, id string, v *entities.Viewer) (*dtos.Campaign, error)
	Create(ctx context.Context, in uc.CampaignInput, v *entities.Viewer) (*dtos.Campaign, error)
	Edit(ctx context.Context, id string, v *entities.Viewer, in uc.CampaignInput) (*dtos.Campaign, error)
	Delete(ctx context.Context, id string, v *entities.Viewer) error
	ChangeStatus(ctx context.Context, id string, v *entities.Viewer, status dtos.CampaignStatus) error

	ListSessions(ctx context.Context, campaignID string, v *entities.Viewer) ([]dtos.CampaignSessionTie, error)
	TieSession(ctx context.Context, campaignID string, v *entities.Viewer, in uc.TieSessionInput) error
	EditTie(ctx context.Context, campaignID, sessionID string, v *entities.Viewer, in uc.EditTieInput) error
	UntieSession(ctx context.Context, campaignID, sessionID string, v *entities.Viewer) error

	ListPlayers(ctx context.Context, campaignID string, v *entities.Viewer) ([]dtos.SessionPlayer, error)
}

type CharacterUsecase interface {
	ListMine(ctx context.Context, v *entities.Viewer, campaignID *string) ([]dtos.Character, error)
	GetByID(ctx context.Context, id string, v *entities.Viewer) (*dtos.Character, error)
	Create(ctx context.Context, v *entities.Viewer, in uc.CreateCharacterInput) (*dtos.Character, error)
	Edit(ctx context.Context, id string, v *entities.Viewer, in uc.EditCharacterInput) (*dtos.Character, error)
	Delete(ctx context.Context, id string, v *entities.Viewer) error
}

type GameSystemsUsecase interface {
	GetAll(ctx context.Context) ([]dtos.GameSystem, error)
	GetCurated(ctx context.Context) ([]dtos.GameSystem, error)
	Search(ctx context.Context, query string) ([]dtos.GameSystem, error)
	AddUserSystem(ctx context.Context, input *uc.CreateGameSystemInput) (*dtos.GameSystem, error)
}

type SessionUsecase interface {
	List(ctx context.Context, in uc.ListSessionsInput, v *entities.Viewer) (dtos.SessionListResponse, error)
	GetByID(ctx context.Context, id string, v *entities.Viewer) (*dtos.SessionResponse, error)
	Create(ctx context.Context, in uc.SessionInput, v *entities.Viewer) (*dtos.Session, error)
	Edit(ctx context.Context, id string, v *entities.Viewer, in uc.SessionInput) (*dtos.Session, error)
	Delete(ctx context.Context, id string, v *entities.Viewer) error
	ChangeStatus(ctx context.Context, id string, v *entities.Viewer, status dtos.SessionStatus) (*dtos.Session, error)

	ListPlayers(ctx context.Context, sessionID string, v *entities.Viewer) (*dtos.SessionPlayersResponse, error)
	Join(ctx context.Context, sessionID string, v *entities.Viewer) error
	Leave(ctx context.Context, sessionID string, v *entities.Viewer) error
	Kick(ctx context.Context, sessionID string, v *entities.Viewer, playerID string) error
	SetMyCharacter(ctx context.Context, sessionID string, v *entities.Viewer, characterID *string) error

	Apply(ctx context.Context, sessionID string, v *entities.Viewer, message *string) (*dtos.SessionApplication, error)
	ListApplications(ctx context.Context, sessionID string, v *entities.Viewer) ([]dtos.SessionApplication, error)
	ResolveApplication(ctx context.Context, applicationID string, v *entities.Viewer, status dtos.SessionApplicationStatus) error

	ListFiles(ctx context.Context, sessionID string, v *entities.Viewer) ([]dtos.SessionFile, error)
	UploadFile(ctx context.Context, in uc.UploadFileInput, v *entities.Viewer) (*dtos.SessionFile, error)
	DeleteFile(ctx context.Context, fileID string, v *entities.Viewer) error

	ListComments(ctx context.Context, sessionID string, v *entities.Viewer) ([]dtos.SessionCommentary, error)
	AddComment(ctx context.Context, sessionID string, v *entities.Viewer, text string) (*dtos.SessionCommentary, error)
	EditComment(ctx context.Context, commentID string, v *entities.Viewer, text string) (*dtos.SessionCommentary, error)
	DeleteComment(ctx context.Context, commentID string, v *entities.Viewer) error

	GetCardData(ctx context.Context, masterIDs []string) ([]dtos.SessionCardData, error)
}