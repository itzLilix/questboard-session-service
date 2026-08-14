package usecase

import (
	"context"
	"fmt"

	"github.com/itzLilix/questboard-session-service/internal/entities"
	"github.com/itzLilix/questboard-session-service/internal/infrastructure"
	"github.com/itzLilix/questboard-shared/dtos"
)

type campaignUsecase struct {
	repo     CampaignRepository
	sessionRepo SessionRepository
	chatRepo ChatRepository
	txManager infrastructure.TxManager
}

func NewCampaignUsecase(repo CampaignRepository, sessionRepo SessionRepository, chatRepo ChatRepository, txManager infrastructure.TxManager) *campaignUsecase {
	return &campaignUsecase{repo: repo, sessionRepo: sessionRepo, chatRepo: chatRepo, txManager: txManager}
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
	BriefDescription  *string
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
			return nil, ErrUnauthorized
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
	ties, err := uc.repo.ListSessionTies(ctx, id, v)
	if err != nil {
		return nil, mapRepoErr("list session ties by campaign id", err)
	}
	campaign.Sessions = ties

	return campaign, nil
}

func (uc *campaignUsecase) Create(ctx context.Context, in CampaignInput, v *entities.Viewer) (*dtos.Campaign, error) {
	if !v.IsAuthenticated() {
		return nil, ErrUnauthorized
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

	params := &infrastructure.CreateCampaignParams{
		Title:        *in.Title,
		Description:  in.Description,
		MasterID:     masterId,
		SystemID:     in.SystemID,
		Status:       status,
		Availability: availability,
	}

	var campaign *dtos.Campaign
	err := uc.txManager.WithTx(ctx, func(ctx context.Context) error {
		var err error
		campaign, err = uc.repo.Create(ctx, params)
		if err != nil {
			return err
		}
		return uc.chatRepo.InitGeneralChat(ctx, campaign.ID, dtos.CampaignType)
	})
	//campaign, err := uc.repo.Create(ctx, params)
	if err != nil {
		return nil, mapRepoErr("create campaign", err)
	}
	return campaign, nil
}

func (uc *campaignUsecase) Edit(ctx context.Context, id string, v *entities.Viewer, in CampaignInput) (*dtos.Campaign, error) {
	if !v.IsAuthenticated() {
		return nil, ErrUnauthorized
	}
	if err := validateCampaign(&in, v); err != nil {
		return nil, err
	}

	existing, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, mapRepoErr("get campaign for edit", err)
	}
	if !v.CanActAs(existing.MasterID) {
		return nil, ErrForbidden
	}
	if existing.Status == dtos.CampaignCompleted || existing.Status == dtos.CampaignCancelled {
		return nil, fmt.Errorf("%w: cannot edit after completion/cancellation", ErrInvalidStatus)
	}

	updated, err := uc.repo.Update(ctx, id, &infrastructure.UpdateCampaignParams{
		Title:        in.Title,
		Description:  in.Description,
		SystemID:     in.SystemID,
		Availability: in.Availability,
	})
	if err != nil {
		return nil, mapRepoErr("update campaign", err)
	}
	return updated, nil
}

func (uc *campaignUsecase) Delete(ctx context.Context, id string, v *entities.Viewer) error {
	if !v.IsAuthenticated() {
		return ErrUnauthorized
	}

	existing, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return mapRepoErr("get campaign for delete", err)
	}
	if !v.CanActAs(existing.MasterID) {
		return ErrForbidden
	}

	if err := uc.repo.Delete(ctx, id); err != nil {
		return mapRepoErr("delete campaign", err)
	}
	return nil
}

func (uc *campaignUsecase) ChangeStatus(ctx context.Context, id string, v *entities.Viewer, status dtos.CampaignStatus) error {
	if !v.IsAuthenticated() {
		return ErrUnauthorized
	}
	if !isValidCampaignStatus(status) {
		return fmt.Errorf("%w: invalid status %q", ErrInvalidStatus, status)
	}

	existing, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return mapRepoErr("get campaign for status change", err)
	}
	if !v.CanActAs(existing.MasterID) {
		return ErrForbidden
	}

	// switch existing.Status {
	// case dtos.CampaignActive:
	// 	if status != dtos.CampaignPaused && status != dtos.CampaignCompleted && status != dtos.CampaignCancelled {
	// 		return ErrInvalidStatus
	// 	}
	// case dtos.CampaignPaused:
	// 	if status != dtos.CampaignActive && status != dtos.CampaignCompleted && status != dtos.CampaignCancelled {
	// 		return ErrInvalidStatus
	// 	}
	// case dtos.CampaignCancelled:
	// 	if status != dtos.CampaignActive {
	// 		return ErrInvalidStatus
	// 	}
	// case dtos.CampaignCompleted:
	// 	if status != dtos.CampaignActive {
	// 		return ErrInvalidStatus
	// 	}
	// default:
	// 	return ErrInvalidStatus
	// }

	if err := uc.repo.UpdateStatus(ctx, id, status); err != nil {
		return mapRepoErr("update campaign status", err)
	}
	return nil
}

// ListSessions returns a campaign's sessions as full session cards (the profile
// accordion's lazy expand). It gates on campaign visibility exactly like GetByID
// — a private campaign's sessions are not enumerable by anyone but its
// master/admins and its members — and then leans on the repo's per-session SQL
// filter for the rest.
func (uc *campaignUsecase) ListSessions(ctx context.Context, campaignID string, v *entities.Viewer) ([]dtos.Session, error) {
	campaign, err := uc.repo.GetByID(ctx, campaignID)
	if err != nil {
		return nil, mapRepoErr("get campaign for list sessions", err)
	}

	if campaign.Availability == dtos.Private && !v.CanActAs(campaign.MasterID) {
		if !v.IsAuthenticated() {
			return nil, ErrUnauthorized
		}
		member, err := uc.repo.IsCampaignMember(ctx, campaignID, v.UserID)
		if err != nil {
			return nil, mapRepoErr("check campaign membership", err)
		}
		if !member {
			return nil, ErrNotFound
		}
	}

	sessions, err := uc.repo.ListSessions(ctx, campaignID, v)
	if err != nil {
		return nil, mapRepoErr("list campaign sessions", err)
	}
	return sessions, nil
}

func (uc *campaignUsecase) TieSession(ctx context.Context, campaignID string, in TieSessionInput, v *entities.Viewer) error {
	if !v.IsAuthenticated() {
		return ErrUnauthorized
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

	session, err := uc.sessionRepo.GetByID(ctx, in.SessionID)
	if err != nil {
		return mapRepoErr("get session for tie session", err)
	}
	if session.MasterID != campaign.MasterID {
		return ErrForbidden
	}
	if session.Type == dtos.CampaignType {
		return ErrConflict
	}

	params := &infrastructure.TieSessionParams{
		CampaignID:        campaignID,
		SessionID:         in.SessionID,
		OrderIndex:        in.OrderIndex,
		BriefDescription:  in.BriefDescription,
		CachedTitle:       &session.Title,
		CachedScheduledAt: session.ScheduledAt,
	}

	// err = uc.repo.TieSession(ctx, params)
	// if err != nil {
	// 	return mapRepoErr("tie session", err)
	// }
	err = uc.txManager.WithTx(ctx, func(ctx context.Context) error {
		err = uc.repo.TieSession(ctx, params)
		if err != nil {
			return err
		}
		return uc.chatRepo.RetireSessionChat(ctx, in.SessionID)
	})
	if err != nil {
		return mapRepoErr("tie session", err)
	}
	return nil
}

func (uc *campaignUsecase) EditTie(ctx context.Context, campaignID, sessionID string, v *entities.Viewer, in EditTieInput) (*dtos.CampaignSessionTie, error) {
	if !v.IsAuthenticated() {
		return nil, ErrUnauthorized
	}
	if in.BriefDescription == nil {
		return nil, ErrInvalidData
	}
	if err := validateEditTie(&in); err != nil {
		return nil, ErrInvalidData
	}

	campaign, err := uc.repo.GetByID(ctx, campaignID)
	if err != nil {
		return nil, mapRepoErr("get campaign for edit tie", err)
	}
	if !v.CanActAs(campaign.MasterID) {
		return nil, ErrForbidden
	}

	tie, err := uc.repo.EditTie(ctx, &infrastructure.EditTieParams{
		CampaignID:       campaignID,
		SessionID:        sessionID,
		BriefDescription: in.BriefDescription,
	})
	if err != nil {
		return nil, mapRepoErr("edit tie", err)
	}
	return tie, nil
}

func (uc *campaignUsecase) UntieSession(ctx context.Context, campaignID, sessionID string, v *entities.Viewer) error {
	if !v.IsAuthenticated() {
		return ErrUnauthorized
	}

	campaign, err := uc.repo.GetByID(ctx, campaignID)
	if err != nil {
		return mapRepoErr("get campaign for untie session", err)
	}
	if !v.CanActAs(campaign.MasterID) {
		return ErrForbidden
	}

	// if err := uc.repo.UntieSession(ctx, campaignID, sessionID); err != nil {
	// 	return mapRepoErr("untie session", err)
	// }
	err = uc.txManager.WithTx(ctx, func(ctx context.Context) error {
		err = uc.repo.UntieSession(ctx, campaignID, sessionID)
		if err != nil {
			return mapRepoErr("untie session", err)
		}
		return uc.chatRepo.InitGeneralChat(ctx, sessionID, dtos.OneshotType)
	})
	if err != nil {
		return mapRepoErr("untie session", err)
	}
	return nil
}

func (uc *campaignUsecase) ReorderSessions(ctx context.Context, campaignID string, orderedSessionIDs []string, v *entities.Viewer) ([]dtos.CampaignSessionTie, error) {
	if !v.IsAuthenticated() {
		return nil, ErrUnauthorized
	}
	if len(orderedSessionIDs) == 0 {
		return nil, ErrInvalidData
	}

	campaign, err := uc.repo.GetByID(ctx, campaignID)
	if err != nil {
		return nil, mapRepoErr("get campaign for reorder sessions", err)
	}
	if !v.CanActAs(campaign.MasterID) {
		return nil, ErrForbidden
	}

	if err := uc.repo.ReorderSessions(ctx, campaignID, orderedSessionIDs); err != nil {
		return nil, mapRepoErr("reorder sessions", err)
	}

	// Re-read so the handler hands back the freshly-ordered ties (the repo reorder
	// returns only an error). Visibility-filtered exactly like GetByID's embed.
	ties, err := uc.repo.ListSessionTies(ctx, campaignID, v)
	if err != nil {
		return nil, mapRepoErr("list session ties after reorder", err)
	}
	return ties, nil
}

func (uc *campaignUsecase) ListPlayers(ctx context.Context, campaignID string, v *entities.Viewer) ([]dtos.SessionPlayer, error) {
	return nil, ErrNotFound
}
