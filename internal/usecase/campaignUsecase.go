package usecase

import (
	"context"
	"time"

	"github.com/itzLilix/questboard-session-service/internal/entities"
	"github.com/itzLilix/questboard-session-service/internal/infrastructure"
	"github.com/itzLilix/questboard-shared/dtos"
)

type campaignUsecase struct {
	repo CampaignRepository
}

func NewCampaignUsecase(repo CampaignRepository) *campaignUsecase {
	return &campaignUsecase{repo: repo}
}

// --- input types -----------------------------------------------------------

type ListCampaignsInput struct {
	Search    string
	MasterID  string
	SystemID  string
	Status    string
	Cursor    string
	Limit     int
	Sort      string
	SortOrder string
}

type CampaignInput struct {
	Title        *string
	Description  *string
	SystemID     *string
	Availability *dtos.SessionAvailability
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

func (uc *campaignUsecase) List(ctx context.Context, in ListCampaignsInput, v *entities.Viewer) (dtos.Page[dtos.Campaign], error) {
	return dtos.Page[dtos.Campaign]{}, ErrNotFound
}

func (uc *campaignUsecase) GetByID(ctx context.Context, id string, v *entities.Viewer) (*dtos.Campaign, error) {
	campaign, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, mapRepoErr("get campaign by id", err)
	}
	sessions, err := uc.repo.ListSessions(ctx, id)
	if err != nil {
		return nil, mapRepoErr("list sessions by campaign id", err)
	}
	campaign.Sessions = sessions

	return campaign, nil
}

func (uc *campaignUsecase) Create(ctx context.Context, in CampaignInput, v *entities.Viewer) (*dtos.Campaign, error) {
	if !v.IsAuthenticated() {
		return nil, ErrForbidden
	}

	if in.Title == nil {
		return nil, ErrInvalidData
	}

	if err := validateCampaign(&in, v); err != nil {
		return nil, err
	}

	status := dtos.CampaignActive
	masterId := v.UserID
	availability := dtos.Open
	if in.Availability != nil {
		availability = *in.Availability
	}

	campaign, err := uc.repo.Create(ctx, &infrastructure.CreateCampaignParams{
		Title:        *in.Title,
		Description:  in.Description,
		MasterID:     masterId,
		SystemID:     in.SystemID,
		Status:       status,
		Availability: availability,
	})
	if err != nil {
		return nil, mapRepoErr("create campaign", err)
	}
	return campaign, nil
}

func (uc *campaignUsecase) Edit(ctx context.Context, id string, v *entities.Viewer, in CampaignInput) (*dtos.Campaign, error) {
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
