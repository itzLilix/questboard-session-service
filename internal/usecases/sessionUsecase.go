package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/itzLilix/questboard-session-service/internal/entities"
	"github.com/itzLilix/questboard-session-service/internal/infrastructure"
	"github.com/itzLilix/questboard-shared/dtos"
)

type SessionUsecase interface {
	List(ctx context.Context, in ListSessionsInput) (dtos.Page[dtos.Session], error)
	GetByID(ctx context.Context, id string, v *entities.Viewer) (*dtos.Session, error)
	Create(ctx context.Context, in CreateSessionInput) (*dtos.Session, error)
	Edit(ctx context.Context, id string, v *entities.Viewer, in EditSessionInput) (*dtos.Session, error)
	Delete(ctx context.Context, id string, v *entities.Viewer) error
	ChangeStatus(ctx context.Context, id string, v *entities.Viewer, status dtos.SessionStatus) (*dtos.Session, error)

	ListPlayers(ctx context.Context, sessionID string, v *entities.Viewer) ([]dtos.SessionPlayer, error)
	Join(ctx context.Context, sessionID string, v *entities.Viewer, characterID *string) error
	Leave(ctx context.Context, sessionID string, v *entities.Viewer) error
	Kick(ctx context.Context, sessionID string, v *entities.Viewer, playerID string) error
	SetMyCharacter(ctx context.Context, sessionID string, v *entities.Viewer, characterID *string) error

	Apply(ctx context.Context, sessionID string, v *entities.Viewer, message *string) (*dtos.SessionApplication, error)
	ListApplications(ctx context.Context, sessionID string, v *entities.Viewer) ([]dtos.SessionApplication, error)
	ResolveApplication(ctx context.Context, applicationID string, v *entities.Viewer, status dtos.SessionApplicationStatus) error

	ListFiles(ctx context.Context, sessionID string, v *entities.Viewer) ([]dtos.SessionFile, error)
	UploadFile(ctx context.Context, in UploadFileInput) (*dtos.SessionFile, error)
	DeleteFile(ctx context.Context, fileID string, v *entities.Viewer) error

	ListComments(ctx context.Context, sessionID string, v *entities.Viewer) ([]dtos.SessionCommentary, error)
	AddComment(ctx context.Context, sessionID string, v *entities.Viewer, text string) (*dtos.SessionCommentary, error)
	EditComment(ctx context.Context, commentID string, v *entities.Viewer, text string) (*dtos.SessionCommentary, error)
	DeleteComment(ctx context.Context, commentID string, v *entities.Viewer) error
}

type sessionUsecase struct {
	repo SessionRepository
}

func NewSessionUsecase(repo SessionRepository) SessionUsecase {
	return &sessionUsecase{repo: repo}
}

// --- input -----------------------------------------------------------

type ListSessionsInput struct {
	Viewer *entities.Viewer

	Scope    dtos.SessionScope
	MasterID string
	PlayerID string
	Status   []string // raw values from handler; usecase expands "public" preset

	Search       string
	Format       string
	Type         string
	City         string
	SystemID     string
	HasFreeSeats bool
	PriceMin     *float64
	PriceMax     *float64
	DateFrom     *time.Time
	DateTo       *time.Time
	Sort         string
	SortOrder    string
	Cursor       string
	Limit        int
}

type CreateSessionInput struct {
	Viewer        *entities.Viewer
	Title         string
	Description   string
	Address       string
	MasterNotes   string
	PreviewURL    string
	Format        dtos.SessionFormat
	Availability  dtos.SessionAvailability
	SystemID      string
	ScheduledAt   *time.Time
	DurationHours *float64
	Lat           *float64
	Lng           *float64
	MaxSeats      int16
	Price         float64
}

type EditSessionInput struct {
	Title         *string
	Description   *string
	Address       *string
	MasterNotes   *string
	PreviewURL    *string
	Format        *dtos.SessionFormat
	Availability  *dtos.SessionAvailability
	SystemID      *string
	ScheduledAt   *time.Time
	DurationHours *float64
	Lat           *float64
	Lng           *float64
	MaxSeats      *int16
	Price         *float64
}

type UploadFileInput struct {
	Viewer    *entities.Viewer
	SessionID string
	Filename  string
	MimeType  string
	Body      []byte
	SizeBytes int64
}

// --- sessions -----------------------------------------------------------

func (uc *sessionUsecase) List(ctx context.Context, in ListSessionsInput) (dtos.Page[dtos.Session], error) {
	scope := in.Scope
	if scope == "" {
		scope = dtos.ScopeCatalog
	}

	// resolve target user and whether it's the viewer
	var (
		masterID, playerID string
		targetIsViewer     bool
	)
	switch scope {
		case dtos.ScopeCatalog:
			targetIsViewer = false
		case dtos.ScopeMastering:
			if in.MasterID != "" {
				masterID = in.MasterID
				targetIsViewer = in.Viewer.Is(masterID)
			} else {
				if !in.Viewer.IsAuthenticated() {
					return dtos.Page[dtos.Session]{}, ErrForbidden
				}
				masterID = in.Viewer.UserID
				targetIsViewer = true
			}
		case dtos.ScopePlaying:
			if in.PlayerID != "" {
				playerID = in.PlayerID
				targetIsViewer = in.Viewer.Is(playerID)
			} else {
				if !in.Viewer.IsAuthenticated() {
					return dtos.Page[dtos.Session]{}, ErrForbidden
				}
				playerID = in.Viewer.UserID
				targetIsViewer = true
			}
		default:
			return dtos.Page[dtos.Session]{}, fmt.Errorf("%w: unknown scope %q", ErrInvalidData, scope)
	}

	// resolve status filter (preset expansion + allowlist enforcement)
	statuses := resolveStatusFilter(in.Status, scope, targetIsViewer)
	if len(statuses) == 0 {
		// every value was dropped → no rows can match
		return dtos.Page[dtos.Session]{Items: []dtos.Session{}}, nil
	}

	limit := in.Limit
	if limit <= 0 {
		limit = 20
	} else if limit > 100 {
		limit = 100
	}

	params := infrastructure.ListSessionsParams{
		Viewer:         in.Viewer,
		Scope:          scope,
		MasterID:       masterID,
		PlayerID:       playerID,
		Status:         statuses,
		TargetIsViewer: targetIsViewer,
		Search:         in.Search,
		Format:         in.Format,
		Type:           in.Type,
		City:           in.City,
		SystemID:       in.SystemID,
		HasFreeSeats:   in.HasFreeSeats,
		PriceMin:       in.PriceMin,
		PriceMax:       in.PriceMax,
		DateFrom:       in.DateFrom,
		DateTo:         in.DateTo,
		Sort:           in.Sort,
		SortOrder:      in.SortOrder,
		Cursor:         in.Cursor,
		Limit:          limit,
	}

	items, nextCursor, err := uc.repo.List(ctx, params)
	if err != nil {
		return dtos.Page[dtos.Session]{}, mapRepoErr("list sessions", err)
	}
	return dtos.Page[dtos.Session]{Items: items, NextCursor: nextCursor}, nil
}

func (uc *sessionUsecase) GetByID(ctx context.Context, id string, v *entities.Viewer) (*dtos.Session, error) {
	s, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, mapRepoErr("get session by id", err)
	}

	if s.Status == dtos.Draft || s.Availability == dtos.Private {
		isParticipant, err := uc.isParticipant(ctx, s, v)
		if err != nil {
			return nil, mapRepoErr("check participant for get session by id", err)
		}
		if !isParticipant {
			return nil, ErrForbidden
		}
	}

	return s, nil
}

func (uc *sessionUsecase) Create(ctx context.Context, in CreateSessionInput) (*dtos.Session, error) {
	if !in.Viewer.IsAuthenticated() {
		return nil, ErrForbidden
	}
	if err := validateCreateSession(&in); err != nil {
		return nil, err
	}

	params := &infrastructure.CreateSessionParams{
		Title:         in.Title,
		Description:   in.Description,
		Address:       in.Address,
		MasterNotes:   in.MasterNotes,
		PreviewURL:    in.PreviewURL,
		Format:        in.Format,
		Availability:  in.Availability,
		SystemID:      in.SystemID,
		MasterID:      in.Viewer.UserID,
		ScheduledAt:   in.ScheduledAt,
		DurationHours: in.DurationHours,
		Lat:           in.Lat,
		Lng:           in.Lng,
		MaxSeats:      in.MaxSeats,
		Price:         in.Price,
	}
	if params.Availability == "" {
		params.Availability = dtos.Open
	}

	s, err := uc.repo.Create(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("create session: %w: %v", ErrInternal, err)
	}
	return s, nil
}

func (uc *sessionUsecase) Edit(ctx context.Context, id string, v *entities.Viewer, in EditSessionInput) (*dtos.Session, error) {
	if !v.IsAuthenticated() {
		return nil, ErrForbidden
	}

	existing, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, mapRepoErr("get session for edit", err)
	}
	if !v.CanActAs(existing.MasterID) {
		return nil, ErrForbidden
	}

	if existing.Status == dtos.Completed || existing.Status == dtos.Cancelled {
		return nil, fmt.Errorf("%w: cannot edit after completion/cancellation", ErrInvalidStatus)
	}

	hasJoiners := existing.MaxSeats != existing.FreeSeats
	if hasJoiners {
		if in.Price != nil {
			return nil, fmt.Errorf("%w: price is locked once players have joined", ErrInvalidData)
		}
		if in.MaxSeats != nil {
			currentPlayers := int16(existing.MaxSeats - existing.FreeSeats)
			if *in.MaxSeats < currentPlayers {
				return nil, fmt.Errorf("%w: maxSeats cannot drop below current player count (%d)", ErrInvalidData, currentPlayers)
			}
		}
	}

	updated, err := uc.repo.Update(ctx, id, &infrastructure.UpdateSessionParams{
		Title:         in.Title,
		Description:   in.Description,
		Address:       in.Address,
		MasterNotes:   in.MasterNotes,
		PreviewURL:    in.PreviewURL,
		Format:        in.Format,
		Availability:  in.Availability,
		SystemID:      in.SystemID,
		ScheduledAt:   in.ScheduledAt,
		DurationHours: in.DurationHours,
		Lat:           in.Lat,
		Lng:           in.Lng,
		MaxSeats:      in.MaxSeats,
		Price:         in.Price,
	})
	if err != nil {
		return nil, mapRepoErr("update session", err)
	}

	// TODO(notifications): push a "session updated" event to
	// the players if the edit touched advertised fields on a published+ session.
	// if existing.Status != dtos.Draft && hasAdvertisedChanges(&in) {}

	return updated, nil
}

func (uc *sessionUsecase) Delete(ctx context.Context, id string, v *entities.Viewer) error {
	if !v.IsAuthenticated() {
		return ErrForbidden
	}

	existing, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return mapRepoErr("get session for delete", err)
	}
	if !v.CanActAs(existing.MasterID) {
		return ErrForbidden
	}

	if err := uc.repo.Delete(ctx, id); err != nil {
		return mapRepoErr("delete session", err)
	}
	return nil
}

func (uc *sessionUsecase) ChangeStatus(ctx context.Context, id string, v *entities.Viewer, status dtos.SessionStatus) (*dtos.Session, error) {
	if !v.IsAuthenticated() {
		return nil, ErrForbidden
	}
	if !isValidSessionStatus(status) {
		return nil, fmt.Errorf("%w: invalid status %q", ErrInvalidStatus, status)
	}

	existing, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, mapRepoErr("get session for status change", err)
	}
	if !v.CanActAs(existing.MasterID) {
		return nil, ErrForbidden
	}

	switch existing.Status {
		case dtos.Draft:
			if status != dtos.Published && status != dtos.Cancelled {
				return nil, ErrInvalidStatus
			}
		case dtos.Published:
			if status != dtos.Ongoing && status != dtos.Cancelled {
				return nil, ErrInvalidStatus
			}
		case dtos.Ongoing:
			if status != dtos.Completed && status != dtos.Cancelled {
				return nil, ErrInvalidStatus
			}
		case dtos.Cancelled:
			if status != dtos.Published {
				return nil, ErrInvalidStatus
			}
		case dtos.Completed:
			return nil, ErrInvalidStatus
		default:
			return nil, ErrInvalidStatus
	}

	// publish-time completeness check: offline sessions must have an address
	if status == dtos.Published && existing.Format == dtos.Offline {
		if existing.Location == nil || existing.Location.Address == "" {
			return nil, fmt.Errorf("%w: offline sessions require an address before publish", ErrInvalidData)
		}
	}

	if err := uc.repo.UpdateStatus(ctx, id, status); err != nil {
		return nil, mapRepoErr("update session status", err)
	}
	return uc.repo.GetByID(ctx, id)
}

// --- players ----------------------------------------------------------------

func (uc *sessionUsecase) isParticipant(ctx context.Context, s *dtos.Session, v *entities.Viewer) (bool, error) {
	if !v.IsAuthenticated() {
		return false, nil
	}

	isPlayer, err := uc.repo.IsPlayer(ctx, s.Id, v.UserID)
	if err != nil {
		return false, mapRepoErr("check participant", err)
	}
	if !isPlayer && !v.CanActAs(s.MasterID) {
		return false, nil
	}
	return true, nil
}

func (uc *sessionUsecase) ListPlayers(ctx context.Context, sessionID string, v *entities.Viewer) ([]dtos.SessionPlayer, error) {
	session, err := uc.repo.GetByID(ctx, sessionID)
	if err != nil {
		return []dtos.SessionPlayer{}, mapRepoErr("get session for list players", err)
	}

	players, err := uc.repo.ListPlayers(ctx, sessionID)
	if err != nil {
		return []dtos.SessionPlayer{}, mapRepoErr("list session players", err)
	}

	if session.Status == dtos.Draft {
		return []dtos.SessionPlayer{}, ErrInvalidData
	}

	if  session.Availability == dtos.Private {
		if v.CanActAs(session.MasterID) {
			return players, nil
		}
		for _, p := range players {
			if v.Is(p.PlayerID) {
				return players, nil
			}
		}
		return []dtos.SessionPlayer{}, ErrForbidden
	}

	return players, nil
}

func (uc *sessionUsecase) Join(ctx context.Context, sessionID string, v *entities.Viewer, characterID *string) error {
	return ErrNotFound
}

func (uc *sessionUsecase) Leave(ctx context.Context, sessionID string, v *entities.Viewer) error {
	return ErrNotFound
}

func (uc *sessionUsecase) Kick(ctx context.Context, sessionID string, v *entities.Viewer, playerID string) error {
	return ErrNotFound
}

func (uc *sessionUsecase) SetMyCharacter(ctx context.Context, sessionID string, v *entities.Viewer, characterID *string) error {
	return ErrNotFound
}

// --- applications -----------------------------------------------------------

func (uc *sessionUsecase) Apply(ctx context.Context, sessionID string, v *entities.Viewer, message *string) (*dtos.SessionApplication, error) {
	return nil, ErrNotFound
}

func (uc *sessionUsecase) ListApplications(ctx context.Context, sessionID string, v *entities.Viewer) ([]dtos.SessionApplication, error) {
	return nil, ErrNotFound
}

func (uc *sessionUsecase) ResolveApplication(ctx context.Context, applicationID string, v *entities.Viewer, status dtos.SessionApplicationStatus) error {
	return ErrNotFound
}

// --- files ----------------------------------------------------------------

func (uc *sessionUsecase) ListFiles(ctx context.Context, sessionID string, v *entities.Viewer) ([]dtos.SessionFile, error) {
	return nil, ErrNotFound
}

func (uc *sessionUsecase) UploadFile(ctx context.Context, in UploadFileInput) (*dtos.SessionFile, error) {
	return nil, ErrNotFound
}

func (uc *sessionUsecase) DeleteFile(ctx context.Context, fileID string, v *entities.Viewer) error {
	return ErrNotFound
}

// --- comments ---------------------------------------------------------------

func (uc *sessionUsecase) ListComments(ctx context.Context, sessionID string, v *entities.Viewer) ([]dtos.SessionCommentary, error) {
	return nil, ErrNotFound
}

func (uc *sessionUsecase) AddComment(ctx context.Context, sessionID string, v *entities.Viewer, text string) (*dtos.SessionCommentary, error) {
	return nil, ErrNotFound
}

func (uc *sessionUsecase) EditComment(ctx context.Context, commentID string, v *entities.Viewer, text string) (*dtos.SessionCommentary, error) {
	return nil, ErrNotFound
}

func (uc *sessionUsecase) DeleteComment(ctx context.Context, commentID string, v *entities.Viewer) error {
	return ErrNotFound
}
