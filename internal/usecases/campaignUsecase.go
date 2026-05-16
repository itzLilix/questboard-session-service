package usecase

import (
	"context"
	"time"

	"github.com/itzLilix/questboard-session-service/internal/entities"
	"github.com/itzLilix/questboard-shared/dtos"
)

type CampaignUsecase interface {
	List(ctx context.Context, in ListCampaignsInput) (dtos.Page[dtos.Campaign], error)
	GetByID(ctx context.Context, id string, v *entities.Viewer) (*dtos.Campaign, error)
	Create(ctx context.Context, in CreateCampaignInput) (*dtos.Campaign, error)
	Edit(ctx context.Context, id string, v *entities.Viewer, in EditCampaignInput) (*dtos.Campaign, error)
	Delete(ctx context.Context, id string, v *entities.Viewer) error
	ChangeStatus(ctx context.Context, id string, v *entities.Viewer, status dtos.CampaignStatus) error

	ListSessions(ctx context.Context, campaignID string, v *entities.Viewer) ([]dtos.CampaignSessionTie, error)
	TieSession(ctx context.Context, campaignID string, v *entities.Viewer, in TieSessionInput) error
	EditTie(ctx context.Context, campaignID, sessionID string, v *entities.Viewer, in EditTieInput) error
	UntieSession(ctx context.Context, campaignID, sessionID string, v *entities.Viewer) error

	ListPlayers(ctx context.Context, campaignID string, v *entities.Viewer) ([]dtos.SessionPlayer, error)
}

type campaignUsecase struct {
	repo CampaignRepository
}

func NewCampaignUsecase(repo CampaignRepository) CampaignUsecase {
	return &campaignUsecase{repo: repo}
}

// --- input shapes -----------------------------------------------------------

type ListCampaignsInput struct {
	Viewer    *entities.Viewer
	Search    string
	MasterID  string
	SystemID  string
	Status    string
	Cursor    string
	Limit     int
	Sort      string
	SortOrder string
}

type CreateCampaignInput struct {
	Viewer      *entities.Viewer
	Title       string
	Description *string
	SystemID    string
}

type EditCampaignInput struct {
	Title       *string
	Description *string
	SystemID    *string
}

type TieSessionInput struct {
	SessionID        string
	OrderIndex       *int
	BriefDescription *string
}

type EditTieInput struct {
	OrderIndex        *int
	BriefDescription  *string
	CachedTitle       *string
	CachedScheduledAt *time.Time
}

// --- method stubs -----------------------------------------------------------

func (uc *campaignUsecase) List(ctx context.Context, in ListCampaignsInput) (dtos.Page[dtos.Campaign], error) {
	return dtos.Page[dtos.Campaign]{}, ErrNotFound
}

func (uc *campaignUsecase) GetByID(ctx context.Context, id string, v *entities.Viewer) (*dtos.Campaign, error) {
	return nil, ErrNotFound
}

func (uc *campaignUsecase) Create(ctx context.Context, in CreateCampaignInput) (*dtos.Campaign, error) {
	return nil, ErrNotFound
}

func (uc *campaignUsecase) Edit(ctx context.Context, id string, v *entities.Viewer, in EditCampaignInput) (*dtos.Campaign, error) {
	return nil, ErrNotFound
}

func (uc *campaignUsecase) Delete(ctx context.Context, id string, v *entities.Viewer) error {
	return ErrNotFound
}

func (uc *campaignUsecase) ChangeStatus(ctx context.Context, id string, v *entities.Viewer, status dtos.CampaignStatus) error {
	return ErrNotFound
}

func (uc *campaignUsecase) ListSessions(ctx context.Context, campaignID string, v *entities.Viewer) ([]dtos.CampaignSessionTie, error) {
	return nil, ErrNotFound
}

func (uc *campaignUsecase) TieSession(ctx context.Context, campaignID string, v *entities.Viewer, in TieSessionInput) error {
	return ErrNotFound
}

func (uc *campaignUsecase) EditTie(ctx context.Context, campaignID, sessionID string, v *entities.Viewer, in EditTieInput) error {
	return ErrNotFound
}

func (uc *campaignUsecase) UntieSession(ctx context.Context, campaignID, sessionID string, v *entities.Viewer) error {
	return ErrNotFound
}

func (uc *campaignUsecase) ListPlayers(ctx context.Context, campaignID string, v *entities.Viewer) ([]dtos.SessionPlayer, error) {
	return nil, ErrNotFound
}
