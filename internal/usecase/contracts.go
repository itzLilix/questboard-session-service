package usecase

import (
	"context"

	"github.com/itzLilix/questboard-session-service/internal/entities"
	"github.com/itzLilix/questboard-session-service/internal/infrastructure"
	"github.com/itzLilix/questboard-shared/dtos"
)

type ProfileClient interface {
	GetBriefs(ctx context.Context, ids []string) (map[string]dtos.UserBrief, error)
}

type ProfileBroker interface {
	UpdateStats(ctx context.Context, stat map[string]int, statName dtos.UserStatName) error
}

type GameSystemsRepository interface {
	GetAll(ctx context.Context) ([]dtos.GameSystem, error)
	GetCurated(ctx context.Context) ([]dtos.GameSystem, error)
	Search(ctx context.Context, q string) ([]dtos.GameSystem, error)
	AddGameSystem(ctx context.Context, params *infrastructure.CreateGameSystemParams) (*dtos.GameSystem, error)
}

type SessionRepository interface {
	// sessions
	List(ctx context.Context, p infrastructure.ListSessionsParams, v *entities.Viewer) ([]dtos.Session, string, error)
	GetByID(ctx context.Context, id string) (*dtos.Session, error)
	Create(ctx context.Context, p *infrastructure.CreateSessionParams) (*dtos.Session, error)
	Update(ctx context.Context, id string, p *infrastructure.UpdateSessionParams) (*dtos.Session, error)
	Delete(ctx context.Context, id string) error
	UpdateStatus(ctx context.Context, id string, status dtos.SessionStatus) error

	// players
	ListPlayers(ctx context.Context, sessionID string) ([]dtos.SessionPlayer, error)
	Join(ctx context.Context, sessionID, playerID string, characterID *string) error
	Leave(ctx context.Context, sessionID, playerID string) error
	Kick(ctx context.Context, sessionID, playerID string) error
	SetCharacter(ctx context.Context, sessionID, playerID string, characterID *string) error
	IsPlayer(ctx context.Context, sessionID, userID string) (bool, error)

	// applications
	CreateApplication(ctx context.Context, p *infrastructure.CreateApplicationParams) (*dtos.SessionApplication, error)
	ListApplications(ctx context.Context, sessionID string) ([]dtos.SessionApplication, error)
	GetApplication(ctx context.Context, applicationID string) (*dtos.SessionApplication, error)
	ResolveApplication(ctx context.Context, applicationID string, status dtos.SessionApplicationStatus) error

	// files
	ListFiles(ctx context.Context, sessionID string) ([]dtos.SessionFile, error)
	AddFile(ctx context.Context, p *infrastructure.AddSessionFileParams) (*dtos.SessionFile, error)
	GetFile(ctx context.Context, fileID string) (*dtos.SessionFile, error)
	DeleteFile(ctx context.Context, fileID string) error

	// comments
	ListComments(ctx context.Context, sessionID string) ([]dtos.SessionCommentary, error)
	AddComment(ctx context.Context, p *infrastructure.AddCommentParams) (*dtos.SessionCommentary, error)
	GetComment(ctx context.Context, commentID string) (*dtos.SessionCommentary, error)
	UpdateComment(ctx context.Context, commentID, text string) (*dtos.SessionCommentary, error)
	DeleteComment(ctx context.Context, commentID string) error

	// internal
	GetSystemStats(ctx context.Context, masterIDs []string) (map[string][]dtos.SystemStat, error)
	GetNextSessions(ctx context.Context, masterIDs []string) (map[string]*dtos.NextSession, error)
	CountMasterStat(ctx context.Context, masterId string) (int, error)
	CountPlayersStats(ctx context.Context, sessionId string) (map[string]int, error)
}

type CampaignRepository interface {
	List(ctx context.Context, p infrastructure.ListCampaignsParams) ([]dtos.Campaign, string, error)
	GetByID(ctx context.Context, id string) (*dtos.Campaign, error)
	Create(ctx context.Context, p *infrastructure.CreateCampaignParams) (*dtos.Campaign, error)
	Update(ctx context.Context, id string, p *infrastructure.UpdateCampaignParams) (*dtos.Campaign, error)
	Delete(ctx context.Context, id string) error
	UpdateStatus(ctx context.Context, id string, status dtos.CampaignStatus) error

	ListSessions(ctx context.Context, campaignID string) ([]dtos.CampaignSessionTie, error)
	TieSession(ctx context.Context, p *infrastructure.TieSessionParams) error
	EditTie(ctx context.Context, p *infrastructure.EditTieParams) error
	UntieSession(ctx context.Context, campaignID, sessionID string) error

	ListPlayers(ctx context.Context, campaignID string) ([]dtos.SessionPlayer, error)
}
