package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/itzLilix/questboard-session-service/internal/entities"
	"github.com/itzLilix/questboard-session-service/internal/infrastructure"
	"github.com/itzLilix/questboard-shared/dtos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

type sessionDeps struct {
	repo    *MockSessionRepository
	profile *MockProfileClient
	broker  *MockProfileBroker
}

func newSessionUC(t *testing.T) (*sessionUsecase, sessionDeps) {
	repo := NewMockSessionRepository(t)
	profile := NewMockProfileClient(t)
	broker := NewMockProfileBroker(t)
	uc := NewSessionUsecase(repo, profile, broker)
	return uc, sessionDeps{repo: repo, profile: profile, broker: broker}
}

func viewer(id string) *entities.Viewer {
	return &entities.Viewer{UserID: id}
}

func adminViewer(id string) *entities.Viewer {
	return &entities.Viewer{UserID: id, Role: dtos.AdminRole}
}

func publishedSession(masterID string) *dtos.Session {
	return &dtos.Session{
		Id:           "s-1",
		Title:        "Test Session",
		MasterID:     masterID,
		Status:       dtos.Published,
		Format:       dtos.Online,
		Availability: dtos.Open,
		MaxSeats:     6,
		FreeSeats:    6,
	}
}

func draftSession(masterID string) *dtos.Session {
	return &dtos.Session{
		Id:           "s-1",
		Title:        "Draft Session",
		MasterID:     masterID,
		Status:       dtos.Draft,
		Format:       dtos.Online,
		Availability: dtos.Open,
		MaxSeats:     6,
		FreeSeats:    6,
	}
}

// stub enrich — profile.GetBriefs returns empty map
func stubEnrich(d sessionDeps) {
	d.profile.EXPECT().GetBriefs(mock.Anything, mock.Anything).
		Return(map[string]dtos.UserBrief{}, nil).Maybe()
}

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

func TestSession_List_Success(t *testing.T) {
	uc, d := newSessionUC(t)
	stubEnrich(d)

	items := []dtos.Session{*publishedSession("master-1")}
	d.repo.EXPECT().List(mock.Anything, mock.Anything, mock.Anything).
		Return(items, "cursor-next", nil)

	resp, err := uc.List(context.Background(), ListSessionsInput{}, viewer("u-1"))
	require.NoError(t, err)
	assert.Len(t, resp.Items, 1)
	assert.Equal(t, "cursor-next", resp.NextCursor)
}

func TestSession_List_ValidationError(t *testing.T) {
	uc, _ := newSessionUC(t)

	_, err := uc.List(context.Background(), ListSessionsInput{Scope: "bad"}, viewer("u-1"))
	assert.ErrorIs(t, err, ErrInvalidData)
}

func TestSession_List_RepoError(t *testing.T) {
	uc, d := newSessionUC(t)

	d.repo.EXPECT().List(mock.Anything, mock.Anything, mock.Anything).
		Return(nil, "", errors.New("db"))

	_, err := uc.List(context.Background(), ListSessionsInput{}, viewer("u-1"))
	assert.ErrorIs(t, err, ErrInternal)
}

func TestSession_List_EmptyStatuses_ReturnsEmpty(t *testing.T) {
	uc, _ := newSessionUC(t)

	// Catalog scope, requesting only draft — filtered out for non-viewer → empty statuses
	in := ListSessionsInput{
		Scope:  dtos.ScopeCatalog,
		Status: []string{string(dtos.Draft)},
	}
	resp, err := uc.List(context.Background(), in, viewer("u-1"))
	require.NoError(t, err)
	assert.Empty(t, resp.Items)
}

// ---------------------------------------------------------------------------
// GetByID
// ---------------------------------------------------------------------------

func TestSession_GetByID_PublishedSession(t *testing.T) {
	uc, d := newSessionUC(t)
	stubEnrich(d)

	s := publishedSession("master-1")
	d.repo.EXPECT().GetByID(mock.Anything, "s-1").Return(s, nil)
	d.repo.EXPECT().ListPlayers(mock.Anything, "s-1").Return(nil, nil)

	resp, err := uc.GetByID(context.Background(), "s-1", viewer("u-1"))
	require.NoError(t, err)
	assert.Equal(t, "s-1", resp.Session.Id)
	assert.Empty(t, resp.Players)
}

func TestSession_GetByID_NotFound(t *testing.T) {
	uc, d := newSessionUC(t)

	d.repo.EXPECT().GetByID(mock.Anything, "nope").Return(nil, infrastructure.ErrNotFound)

	_, err := uc.GetByID(context.Background(), "nope", viewer("u-1"))
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestSession_GetByID_DraftSession_OwnerCanSee(t *testing.T) {
	uc, d := newSessionUC(t)
	stubEnrich(d)

	s := draftSession("master-1")
	d.repo.EXPECT().GetByID(mock.Anything, "s-1").Return(s, nil)
	// Draft sessions: isParticipant check
	d.repo.EXPECT().IsPlayer(mock.Anything, "s-1", "master-1").Return(false, nil)
	// Draft sessions skip ListPlayers

	resp, err := uc.GetByID(context.Background(), "s-1", viewer("master-1"))
	require.NoError(t, err)
	assert.Equal(t, dtos.Draft, resp.Session.Status)
}

func TestSession_GetByID_DraftSession_StrangerForbidden(t *testing.T) {
	uc, d := newSessionUC(t)

	s := draftSession("master-1")
	d.repo.EXPECT().GetByID(mock.Anything, "s-1").Return(s, nil)
	d.repo.EXPECT().IsPlayer(mock.Anything, "s-1", "stranger").Return(false, nil)

	_, err := uc.GetByID(context.Background(), "s-1", viewer("stranger"))
	assert.ErrorIs(t, err, ErrForbidden)
}

func TestSession_GetByID_PrivateSession_PlayerCanSee(t *testing.T) {
	uc, d := newSessionUC(t)
	stubEnrich(d)

	s := publishedSession("master-1")
	s.Availability = dtos.Private
	d.repo.EXPECT().GetByID(mock.Anything, "s-1").Return(s, nil)
	d.repo.EXPECT().IsPlayer(mock.Anything, "s-1", "player-1").Return(true, nil)
	d.repo.EXPECT().ListPlayers(mock.Anything, "s-1").Return(
		[]dtos.SessionPlayer{{SessionID: "s-1", PlayerID: "player-1"}}, nil,
	)

	resp, err := uc.GetByID(context.Background(), "s-1", viewer("player-1"))
	require.NoError(t, err)
	assert.Len(t, resp.Players, 1)
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func validCreateInput() SessionInput {
	return SessionInput{
		Title:    ptr("My Session"),
		Format:   ptr(dtos.Online),
		SystemID: ptr("sys-1"),
		MaxSeats: ptr(int16(6)),
	}
}

func TestSession_Create_Success(t *testing.T) {
	uc, d := newSessionUC(t)

	created := &dtos.Session{Id: "new-1", Title: "My Session", MasterID: "master-1"}
	d.repo.EXPECT().Create(mock.Anything, mock.MatchedBy(func(p *infrastructure.CreateSessionParams) bool {
		return p.Title == "My Session" && p.MasterID == "master-1" && p.MaxSeats == 6
	})).Return(created, nil)

	result, err := uc.Create(context.Background(), validCreateInput(), viewer("master-1"))
	require.NoError(t, err)
	assert.Equal(t, "new-1", result.Id)
}

func TestSession_Create_Unauthenticated(t *testing.T) {
	uc, _ := newSessionUC(t)

	_, err := uc.Create(context.Background(), validCreateInput(), &entities.Viewer{})
	assert.ErrorIs(t, err, ErrForbidden)
}

func TestSession_Create_MissingTitle(t *testing.T) {
	uc, _ := newSessionUC(t)
	in := validCreateInput()
	in.Title = nil

	_, err := uc.Create(context.Background(), in, viewer("u-1"))
	assert.ErrorIs(t, err, ErrInvalidData)
}

func TestSession_Create_MissingFormat(t *testing.T) {
	uc, _ := newSessionUC(t)
	in := validCreateInput()
	in.Format = nil

	_, err := uc.Create(context.Background(), in, viewer("u-1"))
	assert.ErrorIs(t, err, ErrInvalidData)
}

func TestSession_Create_MissingSystemID(t *testing.T) {
	uc, _ := newSessionUC(t)
	in := validCreateInput()
	in.SystemID = nil

	_, err := uc.Create(context.Background(), in, viewer("u-1"))
	assert.ErrorIs(t, err, ErrInvalidData)
}

func TestSession_Create_MissingMaxSeats(t *testing.T) {
	uc, _ := newSessionUC(t)
	in := validCreateInput()
	in.MaxSeats = nil

	_, err := uc.Create(context.Background(), in, viewer("u-1"))
	assert.ErrorIs(t, err, ErrInvalidData)
}

func TestSession_Create_EmptyTitle(t *testing.T) {
	uc, _ := newSessionUC(t)
	in := validCreateInput()
	in.Title = ptr("")

	_, err := uc.Create(context.Background(), in, viewer("u-1"))
	assert.ErrorIs(t, err, ErrInvalidData)
}

func TestSession_Create_DefaultsAvailabilityToOpen(t *testing.T) {
	uc, d := newSessionUC(t)

	d.repo.EXPECT().Create(mock.Anything, mock.MatchedBy(func(p *infrastructure.CreateSessionParams) bool {
		return p.Availability == dtos.Open
	})).Return(&dtos.Session{Id: "x"}, nil)

	in := validCreateInput()
	in.Availability = nil // should default to Open

	_, err := uc.Create(context.Background(), in, viewer("u-1"))
	require.NoError(t, err)
}

func TestSession_Create_ValidationError_MaxSeatsTooHigh(t *testing.T) {
	uc, _ := newSessionUC(t)
	in := validCreateInput()
	in.MaxSeats = ptr(int16(51))

	_, err := uc.Create(context.Background(), in, viewer("u-1"))
	assert.ErrorIs(t, err, ErrInvalidData)
}

func TestSession_Create_RepoError(t *testing.T) {
	uc, d := newSessionUC(t)
	d.repo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil, errors.New("db"))

	_, err := uc.Create(context.Background(), validCreateInput(), viewer("u-1"))
	assert.ErrorIs(t, err, ErrInternal)
}

// ---------------------------------------------------------------------------
// Edit
// ---------------------------------------------------------------------------

func TestSession_Edit_Success(t *testing.T) {
	uc, d := newSessionUC(t)

	existing := publishedSession("master-1")
	updated := publishedSession("master-1")
	updated.Title = "Updated"

	d.repo.EXPECT().GetByID(mock.Anything, "s-1").Return(existing, nil)
	d.repo.EXPECT().Update(mock.Anything, "s-1", mock.Anything).Return(updated, nil)

	result, err := uc.Edit(context.Background(), "s-1", viewer("master-1"), SessionInput{Title: ptr("Updated")})
	require.NoError(t, err)
	assert.Equal(t, "Updated", result.Title)
}

func TestSession_Edit_Unauthenticated(t *testing.T) {
	uc, _ := newSessionUC(t)

	_, err := uc.Edit(context.Background(), "s-1", &entities.Viewer{}, SessionInput{})
	assert.ErrorIs(t, err, ErrForbidden)
}

func TestSession_Edit_NotOwner(t *testing.T) {
	uc, d := newSessionUC(t)

	existing := publishedSession("master-1")
	d.repo.EXPECT().GetByID(mock.Anything, "s-1").Return(existing, nil)

	_, err := uc.Edit(context.Background(), "s-1", viewer("other"), SessionInput{})
	assert.ErrorIs(t, err, ErrForbidden)
}

func TestSession_Edit_AdminCanEdit(t *testing.T) {
	uc, d := newSessionUC(t)

	existing := publishedSession("master-1")
	updated := publishedSession("master-1")
	d.repo.EXPECT().GetByID(mock.Anything, "s-1").Return(existing, nil)
	d.repo.EXPECT().Update(mock.Anything, "s-1", mock.Anything).Return(updated, nil)

	_, err := uc.Edit(context.Background(), "s-1", adminViewer("admin-1"), SessionInput{})
	assert.NoError(t, err)
}

func TestSession_Edit_CompletedSession_Rejected(t *testing.T) {
	uc, d := newSessionUC(t)

	existing := publishedSession("master-1")
	existing.Status = dtos.Completed
	d.repo.EXPECT().GetByID(mock.Anything, "s-1").Return(existing, nil)

	_, err := uc.Edit(context.Background(), "s-1", viewer("master-1"), SessionInput{})
	assert.ErrorIs(t, err, ErrInvalidStatus)
}

func TestSession_Edit_CancelledSession_Rejected(t *testing.T) {
	uc, d := newSessionUC(t)

	existing := publishedSession("master-1")
	existing.Status = dtos.Cancelled
	d.repo.EXPECT().GetByID(mock.Anything, "s-1").Return(existing, nil)

	_, err := uc.Edit(context.Background(), "s-1", viewer("master-1"), SessionInput{})
	assert.ErrorIs(t, err, ErrInvalidStatus)
}

func TestSession_Edit_PriceLockedWithPlayers(t *testing.T) {
	uc, d := newSessionUC(t)

	existing := publishedSession("master-1")
	existing.FreeSeats = 4 // 2 players joined (6 max - 4 free)
	d.repo.EXPECT().GetByID(mock.Anything, "s-1").Return(existing, nil)

	_, err := uc.Edit(context.Background(), "s-1", viewer("master-1"), SessionInput{Price: ptr(999.0)})
	assert.ErrorIs(t, err, ErrInvalidData)
	assert.Contains(t, err.Error(), "price is locked")
}

func TestSession_Edit_MaxSeatsBelowPlayerCount(t *testing.T) {
	uc, d := newSessionUC(t)

	existing := publishedSession("master-1")
	existing.MaxSeats = 6
	existing.FreeSeats = 3 // 3 players
	d.repo.EXPECT().GetByID(mock.Anything, "s-1").Return(existing, nil)

	_, err := uc.Edit(context.Background(), "s-1", viewer("master-1"), SessionInput{MaxSeats: ptr(int16(2))})
	assert.ErrorIs(t, err, ErrInvalidData)
	assert.Contains(t, err.Error(), "maxSeats cannot drop below")
}

func TestSession_Edit_MaxSeatsAbovePlayerCount_OK(t *testing.T) {
	uc, d := newSessionUC(t)

	existing := publishedSession("master-1")
	existing.MaxSeats = 6
	existing.FreeSeats = 3 // 3 players
	updated := publishedSession("master-1")
	d.repo.EXPECT().GetByID(mock.Anything, "s-1").Return(existing, nil)
	d.repo.EXPECT().Update(mock.Anything, "s-1", mock.Anything).Return(updated, nil)

	_, err := uc.Edit(context.Background(), "s-1", viewer("master-1"), SessionInput{MaxSeats: ptr(int16(4))})
	assert.NoError(t, err)
}

func TestSession_Edit_NotFound(t *testing.T) {
	uc, d := newSessionUC(t)
	d.repo.EXPECT().GetByID(mock.Anything, "nope").Return(nil, infrastructure.ErrNotFound)

	_, err := uc.Edit(context.Background(), "nope", viewer("u-1"), SessionInput{})
	assert.ErrorIs(t, err, ErrNotFound)
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestSession_Delete_Success(t *testing.T) {
	uc, d := newSessionUC(t)

	existing := publishedSession("master-1")
	d.repo.EXPECT().GetByID(mock.Anything, "s-1").Return(existing, nil)
	d.repo.EXPECT().Delete(mock.Anything, "s-1").Return(nil)

	err := uc.Delete(context.Background(), "s-1", viewer("master-1"))
	assert.NoError(t, err)
}

func TestSession_Delete_Unauthenticated(t *testing.T) {
	uc, _ := newSessionUC(t)

	err := uc.Delete(context.Background(), "s-1", &entities.Viewer{})
	assert.ErrorIs(t, err, ErrForbidden)
}

func TestSession_Delete_NotOwner(t *testing.T) {
	uc, d := newSessionUC(t)

	existing := publishedSession("master-1")
	d.repo.EXPECT().GetByID(mock.Anything, "s-1").Return(existing, nil)

	err := uc.Delete(context.Background(), "s-1", viewer("other"))
	assert.ErrorIs(t, err, ErrForbidden)
}

func TestSession_Delete_NotFound(t *testing.T) {
	uc, d := newSessionUC(t)
	d.repo.EXPECT().GetByID(mock.Anything, "nope").Return(nil, infrastructure.ErrNotFound)

	err := uc.Delete(context.Background(), "nope", viewer("u-1"))
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestSession_Delete_AdminCanDelete(t *testing.T) {
	uc, d := newSessionUC(t)

	existing := publishedSession("master-1")
	d.repo.EXPECT().GetByID(mock.Anything, "s-1").Return(existing, nil)
	d.repo.EXPECT().Delete(mock.Anything, "s-1").Return(nil)

	err := uc.Delete(context.Background(), "s-1", adminViewer("admin-1"))
	assert.NoError(t, err)
}

// ---------------------------------------------------------------------------
// ChangeStatus — state machine
// ---------------------------------------------------------------------------

func TestSession_ChangeStatus_Unauthenticated(t *testing.T) {
	uc, _ := newSessionUC(t)

	_, err := uc.ChangeStatus(context.Background(), "s-1", &entities.Viewer{}, dtos.Published)
	assert.ErrorIs(t, err, ErrForbidden)
}

func TestSession_ChangeStatus_InvalidStatusString(t *testing.T) {
	uc, _ := newSessionUC(t)

	_, err := uc.ChangeStatus(context.Background(), "s-1", viewer("u-1"), dtos.SessionStatus("bogus"))
	assert.ErrorIs(t, err, ErrInvalidStatus)
}

func TestSession_ChangeStatus_NotOwner(t *testing.T) {
	uc, d := newSessionUC(t)

	existing := draftSession("master-1")
	d.repo.EXPECT().GetByID(mock.Anything, "s-1").Return(existing, nil)

	_, err := uc.ChangeStatus(context.Background(), "s-1", viewer("other"), dtos.Published)
	assert.ErrorIs(t, err, ErrForbidden)
}

func TestSession_ChangeStatus_DraftToPublished(t *testing.T) {
	uc, d := newSessionUC(t)

	existing := draftSession("master-1")
	d.repo.EXPECT().GetByID(mock.Anything, "s-1").Return(existing, nil).Once()
	d.repo.EXPECT().UpdateStatus(mock.Anything, "s-1", dtos.Published).Return(nil)
	d.repo.EXPECT().CountMasterStat(mock.Anything, "master-1").Return(5, nil)
	d.broker.EXPECT().UpdateStats(mock.Anything, map[string]int{"master-1": 5}, dtos.HostedStatName).Return(nil)
	// final GetByID to return updated session
	updated := publishedSession("master-1")
	d.repo.EXPECT().GetByID(mock.Anything, "s-1").Return(updated, nil).Once()

	result, err := uc.ChangeStatus(context.Background(), "s-1", viewer("master-1"), dtos.Published)
	require.NoError(t, err)
	assert.Equal(t, dtos.Published, result.Status)
}

func TestSession_ChangeStatus_DraftToCancelled(t *testing.T) {
	uc, d := newSessionUC(t)

	existing := draftSession("master-1")
	d.repo.EXPECT().GetByID(mock.Anything, "s-1").Return(existing, nil).Once()
	d.repo.EXPECT().UpdateStatus(mock.Anything, "s-1", dtos.Cancelled).Return(nil)
	d.repo.EXPECT().CountMasterStat(mock.Anything, "master-1").Return(3, nil)
	d.broker.EXPECT().UpdateStats(mock.Anything, mock.Anything, dtos.HostedStatName).Return(nil)
	cancelled := draftSession("master-1")
	cancelled.Status = dtos.Cancelled
	d.repo.EXPECT().GetByID(mock.Anything, "s-1").Return(cancelled, nil).Once()

	result, err := uc.ChangeStatus(context.Background(), "s-1", viewer("master-1"), dtos.Cancelled)
	require.NoError(t, err)
	assert.Equal(t, dtos.Cancelled, result.Status)
}

func TestSession_ChangeStatus_DraftToOngoing_Invalid(t *testing.T) {
	uc, d := newSessionUC(t)

	existing := draftSession("master-1")
	d.repo.EXPECT().GetByID(mock.Anything, "s-1").Return(existing, nil)

	_, err := uc.ChangeStatus(context.Background(), "s-1", viewer("master-1"), dtos.Ongoing)
	assert.ErrorIs(t, err, ErrInvalidStatus)
}

func TestSession_ChangeStatus_PublishedToOngoing(t *testing.T) {
	uc, d := newSessionUC(t)

	existing := publishedSession("master-1")
	d.repo.EXPECT().GetByID(mock.Anything, "s-1").Return(existing, nil).Once()
	d.repo.EXPECT().UpdateStatus(mock.Anything, "s-1", dtos.Ongoing).Return(nil)
	ongoing := publishedSession("master-1")
	ongoing.Status = dtos.Ongoing
	d.repo.EXPECT().GetByID(mock.Anything, "s-1").Return(ongoing, nil).Once()

	result, err := uc.ChangeStatus(context.Background(), "s-1", viewer("master-1"), dtos.Ongoing)
	require.NoError(t, err)
	assert.Equal(t, dtos.Ongoing, result.Status)
}

func TestSession_ChangeStatus_PublishedToDraft_Invalid(t *testing.T) {
	uc, d := newSessionUC(t)

	existing := publishedSession("master-1")
	d.repo.EXPECT().GetByID(mock.Anything, "s-1").Return(existing, nil)

	_, err := uc.ChangeStatus(context.Background(), "s-1", viewer("master-1"), dtos.Draft)
	assert.ErrorIs(t, err, ErrInvalidStatus)
}

func TestSession_ChangeStatus_OngoingToCompleted(t *testing.T) {
	uc, d := newSessionUC(t)

	existing := publishedSession("master-1")
	existing.Status = dtos.Ongoing
	d.repo.EXPECT().GetByID(mock.Anything, "s-1").Return(existing, nil).Once()
	d.repo.EXPECT().UpdateStatus(mock.Anything, "s-1", dtos.Completed).Return(nil)
	d.repo.EXPECT().CountPlayersStats(mock.Anything, "s-1").Return(map[string]int{"p-1": 3}, nil)
	d.broker.EXPECT().UpdateStats(mock.Anything, map[string]int{"p-1": 3}, dtos.PlayedStatName).Return(nil)
	completed := publishedSession("master-1")
	completed.Status = dtos.Completed
	d.repo.EXPECT().GetByID(mock.Anything, "s-1").Return(completed, nil).Once()

	result, err := uc.ChangeStatus(context.Background(), "s-1", viewer("master-1"), dtos.Completed)
	require.NoError(t, err)
	assert.Equal(t, dtos.Completed, result.Status)
}

func TestSession_ChangeStatus_CompletedToAnything_Invalid(t *testing.T) {
	uc, d := newSessionUC(t)

	existing := publishedSession("master-1")
	existing.Status = dtos.Completed
	d.repo.EXPECT().GetByID(mock.Anything, "s-1").Return(existing, nil)

	_, err := uc.ChangeStatus(context.Background(), "s-1", viewer("master-1"), dtos.Published)
	assert.ErrorIs(t, err, ErrInvalidStatus)
}

func TestSession_ChangeStatus_CancelledToPublished(t *testing.T) {
	uc, d := newSessionUC(t)

	existing := publishedSession("master-1")
	existing.Status = dtos.Cancelled
	d.repo.EXPECT().GetByID(mock.Anything, "s-1").Return(existing, nil).Once()
	d.repo.EXPECT().UpdateStatus(mock.Anything, "s-1", dtos.Published).Return(nil)
	d.repo.EXPECT().CountMasterStat(mock.Anything, "master-1").Return(4, nil)
	d.broker.EXPECT().UpdateStats(mock.Anything, mock.Anything, dtos.HostedStatName).Return(nil)
	restored := publishedSession("master-1")
	d.repo.EXPECT().GetByID(mock.Anything, "s-1").Return(restored, nil).Once()

	result, err := uc.ChangeStatus(context.Background(), "s-1", viewer("master-1"), dtos.Published)
	require.NoError(t, err)
	assert.Equal(t, dtos.Published, result.Status)
}

func TestSession_ChangeStatus_CancelledToOngoing_Invalid(t *testing.T) {
	uc, d := newSessionUC(t)

	existing := publishedSession("master-1")
	existing.Status = dtos.Cancelled
	d.repo.EXPECT().GetByID(mock.Anything, "s-1").Return(existing, nil)

	_, err := uc.ChangeStatus(context.Background(), "s-1", viewer("master-1"), dtos.Ongoing)
	assert.ErrorIs(t, err, ErrInvalidStatus)
}

func TestSession_ChangeStatus_OfflinePublish_RequiresAddress(t *testing.T) {
	uc, d := newSessionUC(t)

	existing := draftSession("master-1")
	existing.Format = dtos.Offline
	existing.Location = nil
	d.repo.EXPECT().GetByID(mock.Anything, "s-1").Return(existing, nil)

	_, err := uc.ChangeStatus(context.Background(), "s-1", viewer("master-1"), dtos.Published)
	assert.ErrorIs(t, err, ErrInvalidData)
	assert.Contains(t, err.Error(), "address")
}

func TestSession_ChangeStatus_OfflinePublish_EmptyAddress(t *testing.T) {
	uc, d := newSessionUC(t)

	existing := draftSession("master-1")
	existing.Format = dtos.Offline
	existing.Location = &dtos.Location{Address: ""}
	d.repo.EXPECT().GetByID(mock.Anything, "s-1").Return(existing, nil)

	_, err := uc.ChangeStatus(context.Background(), "s-1", viewer("master-1"), dtos.Published)
	assert.ErrorIs(t, err, ErrInvalidData)
}

func TestSession_ChangeStatus_OfflinePublish_WithAddress_OK(t *testing.T) {
	uc, d := newSessionUC(t)

	existing := draftSession("master-1")
	existing.Format = dtos.Offline
	existing.Location = &dtos.Location{Address: "123 Main St"}
	d.repo.EXPECT().GetByID(mock.Anything, "s-1").Return(existing, nil).Once()
	d.repo.EXPECT().UpdateStatus(mock.Anything, "s-1", dtos.Published).Return(nil)
	d.repo.EXPECT().CountMasterStat(mock.Anything, "master-1").Return(1, nil)
	d.broker.EXPECT().UpdateStats(mock.Anything, mock.Anything, dtos.HostedStatName).Return(nil)
	published := publishedSession("master-1")
	published.Format = dtos.Offline
	d.repo.EXPECT().GetByID(mock.Anything, "s-1").Return(published, nil).Once()

	_, err := uc.ChangeStatus(context.Background(), "s-1", viewer("master-1"), dtos.Published)
	assert.NoError(t, err)
}

// ---------------------------------------------------------------------------
// ListPlayers
// ---------------------------------------------------------------------------

func TestSession_ListPlayers_Success(t *testing.T) {
	uc, d := newSessionUC(t)
	stubEnrich(d)

	s := publishedSession("master-1")
	players := []dtos.SessionPlayer{
		{SessionID: "s-1", PlayerID: "p-1"},
		{SessionID: "s-1", PlayerID: "p-2"},
	}
	d.repo.EXPECT().GetByID(mock.Anything, "s-1").Return(s, nil)
	d.repo.EXPECT().ListPlayers(mock.Anything, "s-1").Return(players, nil)

	resp, err := uc.ListPlayers(context.Background(), "s-1", viewer("u-1"))
	require.NoError(t, err)
	assert.Len(t, resp.Players, 2)
}

func TestSession_ListPlayers_DraftSession_ReturnsError(t *testing.T) {
	uc, d := newSessionUC(t)

	s := draftSession("master-1")
	d.repo.EXPECT().GetByID(mock.Anything, "s-1").Return(s, nil)
	d.repo.EXPECT().ListPlayers(mock.Anything, "s-1").Return(nil, nil)

	_, err := uc.ListPlayers(context.Background(), "s-1", viewer("u-1"))
	assert.ErrorIs(t, err, ErrInvalidData)
}

func TestSession_ListPlayers_PrivateSession_OwnerCanSee(t *testing.T) {
	uc, d := newSessionUC(t)
	stubEnrich(d)

	s := publishedSession("master-1")
	s.Availability = dtos.Private
	d.repo.EXPECT().GetByID(mock.Anything, "s-1").Return(s, nil)
	d.repo.EXPECT().ListPlayers(mock.Anything, "s-1").Return(nil, nil)

	resp, err := uc.ListPlayers(context.Background(), "s-1", viewer("master-1"))
	require.NoError(t, err)
	assert.Empty(t, resp.Players)
}

func TestSession_ListPlayers_PrivateSession_StrangerForbidden(t *testing.T) {
	uc, d := newSessionUC(t)

	s := publishedSession("master-1")
	s.Availability = dtos.Private
	d.repo.EXPECT().GetByID(mock.Anything, "s-1").Return(s, nil)
	d.repo.EXPECT().ListPlayers(mock.Anything, "s-1").Return(nil, nil)

	_, err := uc.ListPlayers(context.Background(), "s-1", viewer("stranger"))
	assert.ErrorIs(t, err, ErrForbidden)
}

func TestSession_ListPlayers_PrivateSession_PlayerCanSee(t *testing.T) {
	uc, d := newSessionUC(t)
	stubEnrich(d)

	s := publishedSession("master-1")
	s.Availability = dtos.Private
	players := []dtos.SessionPlayer{{SessionID: "s-1", PlayerID: "player-1"}}
	d.repo.EXPECT().GetByID(mock.Anything, "s-1").Return(s, nil)
	d.repo.EXPECT().ListPlayers(mock.Anything, "s-1").Return(players, nil)

	resp, err := uc.ListPlayers(context.Background(), "s-1", viewer("player-1"))
	require.NoError(t, err)
	assert.Len(t, resp.Players, 1)
}

func TestSession_ListPlayers_NotFound(t *testing.T) {
	uc, d := newSessionUC(t)
	d.repo.EXPECT().GetByID(mock.Anything, "nope").Return(nil, infrastructure.ErrNotFound)

	_, err := uc.ListPlayers(context.Background(), "nope", viewer("u-1"))
	assert.ErrorIs(t, err, ErrNotFound)
}

// ---------------------------------------------------------------------------
// GetCardData
// ---------------------------------------------------------------------------

func TestSession_GetCardData_Success(t *testing.T) {
	uc, d := newSessionUC(t)

	stats := map[string][]dtos.SystemStat{
		"m-1": {{GameSystem: dtos.GameSystem{Name: "D&D"}, SessionsCount: 10}},
	}
	next := map[string]*dtos.NextSession{
		"m-1": {Id: "s-next", ScheduledAt: time.Now()},
	}
	d.repo.EXPECT().GetSystemStats(mock.Anything, []string{"m-1"}).Return(stats, nil)
	d.repo.EXPECT().GetNextSessions(mock.Anything, []string{"m-1"}).Return(next, nil)

	result, err := uc.GetCardData(context.Background(), []string{"m-1"})
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "m-1", result[0].UserID)
	assert.Len(t, result[0].SystemStats, 1)
	assert.NotNil(t, result[0].NextSession)
}

func TestSession_GetCardData_EmptyInput(t *testing.T) {
	uc, _ := newSessionUC(t)

	result, err := uc.GetCardData(context.Background(), []string{})
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestSession_GetCardData_ExceedsBatchLimit(t *testing.T) {
	uc, _ := newSessionUC(t)

	ids := make([]string, 51)
	for i := range ids {
		ids[i] = "m-" + string(rune('a'+i%26))
	}

	_, err := uc.GetCardData(context.Background(), ids)
	assert.ErrorIs(t, err, ErrInvalidData)
}

func TestSession_GetCardData_Deduplicates(t *testing.T) {
	uc, d := newSessionUC(t)

	d.repo.EXPECT().GetSystemStats(mock.Anything, mock.MatchedBy(func(ids []string) bool {
		return len(ids) == 2
	})).Return(map[string][]dtos.SystemStat{}, nil)
	d.repo.EXPECT().GetNextSessions(mock.Anything, mock.MatchedBy(func(ids []string) bool {
		return len(ids) == 2
	})).Return(map[string]*dtos.NextSession{}, nil)

	result, err := uc.GetCardData(context.Background(), []string{"m-1", "m-2", "m-1", "m-2", "m-1"})
	require.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestSession_GetCardData_SkipsEmptyIDs(t *testing.T) {
	uc, d := newSessionUC(t)

	d.repo.EXPECT().GetSystemStats(mock.Anything, mock.MatchedBy(func(ids []string) bool {
		return len(ids) == 1 && ids[0] == "m-1"
	})).Return(map[string][]dtos.SystemStat{}, nil)
	d.repo.EXPECT().GetNextSessions(mock.Anything, mock.Anything).
		Return(map[string]*dtos.NextSession{}, nil)

	result, err := uc.GetCardData(context.Background(), []string{"", "m-1", ""})
	require.NoError(t, err)
	assert.Len(t, result, 1)
}

func TestSession_GetCardData_StatsRepoError(t *testing.T) {
	uc, d := newSessionUC(t)

	d.repo.EXPECT().GetSystemStats(mock.Anything, mock.Anything).
		Return(nil, errors.New("db"))

	_, err := uc.GetCardData(context.Background(), []string{"m-1"})
	assert.ErrorIs(t, err, ErrInternal)
}

func TestSession_GetCardData_NextSessionsRepoError(t *testing.T) {
	uc, d := newSessionUC(t)

	d.repo.EXPECT().GetSystemStats(mock.Anything, mock.Anything).
		Return(map[string][]dtos.SystemStat{}, nil)
	d.repo.EXPECT().GetNextSessions(mock.Anything, mock.Anything).
		Return(nil, errors.New("db"))

	_, err := uc.GetCardData(context.Background(), []string{"m-1"})
	assert.ErrorIs(t, err, ErrInternal)
}

// ---------------------------------------------------------------------------
// enrich (private helper — tested via integration with GetByID/List)
// ---------------------------------------------------------------------------

func TestSession_Enrich_ProfileError_ReturnsEmptyMap(t *testing.T) {
	uc, d := newSessionUC(t)
	d.profile.EXPECT().GetBriefs(mock.Anything, mock.Anything).Return(nil, errors.New("grpc down"))

	result := uc.enrich(context.Background(), []string{"u-1"})
	assert.Empty(t, result)
}

func TestSession_Enrich_NilBriefs_ReturnsEmptyMap(t *testing.T) {
	uc, d := newSessionUC(t)
	d.profile.EXPECT().GetBriefs(mock.Anything, mock.Anything).Return(nil, nil)

	result := uc.enrich(context.Background(), []string{"u-1"})
	assert.Empty(t, result)
}
