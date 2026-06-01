package usecase

import (
	"context"
	"time"

	"github.com/itzLilix/questboard-session-service/internal/entities"
	"github.com/itzLilix/questboard-session-service/internal/infrastructure"
	"github.com/itzLilix/questboard-shared/dtos"
)

type campaignUsecase struct {
	repo     CampaignRepository
	sessions SessionRepository
}

func NewCampaignUsecase(repo CampaignRepository, sessions SessionRepository) *campaignUsecase {
	return &campaignUsecase{repo: repo, sessions: sessions}
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

// --- methods -----------------------------------------------------------

func (uc *campaignUsecase) List(ctx context.Context, in ListCampaignsInput, v *entities.Viewer) (dtos.Page[dtos.Campaign], error) {
	params, err := validateListCampaigns(&in, v)
	if err != nil {
		return dtos.Page[dtos.Campaign]{}, err
	}

	items, nextCursor, err := uc.repo.List(ctx, params)
	if err != nil {
		return dtos.Page[dtos.Campaign]{}, mapRepoErr("list campaigns", err)
	}
	if items == nil {
		items = []dtos.Campaign{}
	}

	return dtos.Page[dtos.Campaign]{Items: items, NextCursor: nextCursor}, nil
}

func (uc *campaignUsecase) GetByID(ctx context.Context, id string, v *entities.Viewer) (*dtos.Campaign, error) {
	campaign, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, mapRepoErr("get campaign by id", err)
	}

	// A private campaign is hidden from everyone except its master/admins and its
	// members (active players of any session in it). An "open" session inside a
	// private campaign is a deliberate recruiting door, but it must not expose the
	// campaign it belongs to — hence the gate before we ever return campaign data.
	if campaign.Availability == dtos.Private && !v.CanActAs(campaign.MasterID) {
		if !v.IsAuthenticated() {
			return nil, ErrNotFound
		}
		member, err := uc.repo.IsCampaignMember(ctx, id, v.UserID)
		if err != nil {
			return nil, mapRepoErr("check campaign membership", err)
		}
		if !member {
			return nil, ErrNotFound
		}
	}

	// The embedded session list is filtered at the repo (SQL) layer: private and
	// draft sessions are dropped for viewers who aren't the master or a player of
	// that specific session, so a public campaign never leaks its private sessions.
	sessions, err := uc.repo.ListSessions(ctx, id, v)
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
	if in.Availability != nil && *in.Availability != "" {
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

func (uc *campaignUsecase) TieSession(ctx context.Context, campaignID string, in TieSessionInput, v *entities.Viewer) error {
	if !v.IsAuthenticated() {
		return ErrForbidden
	}
	if in.SessionID == "" {
		return ErrInvalidData
	}

	campaign, err := uc.repo.GetByID(ctx, campaignID)
	if err != nil {
		return mapRepoErr("get campaign for tie session", err)
	}
	if !v.CanActAs(campaign.MasterID) {
		return ErrForbidden
	}

	session, err := uc.sessions.GetByID(ctx, in.SessionID)
	if err != nil {
		return mapRepoErr("get session for tie session", err)
	}
	if session.MasterID != campaign.MasterID {
		return ErrForbidden
	}
	if session.Type == dtos.CampaignType {
		return ErrConflict
	}

	err = uc.repo.TieSession(ctx, &infrastructure.TieSessionParams{
		CampaignID:        campaignID,
		SessionID:         in.SessionID,
		OrderIndex:        in.OrderIndex,
		BriefDescription:  in.BriefDescription,
		CachedTitle:       &session.Title,
		CachedScheduledAt: session.ScheduledAt,
	})
	if err != nil {
		return mapRepoErr("tie session", err)
	}
	return nil
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
