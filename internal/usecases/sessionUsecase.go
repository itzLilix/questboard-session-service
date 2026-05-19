package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/itzLilix/questboard-session-service/internal/entities"
	"github.com/itzLilix/questboard-session-service/internal/infrastructure"
	"github.com/itzLilix/questboard-shared/dtos"
	"github.com/rs/zerolog/log"
)

type SessionUsecase interface {
	List(ctx context.Context, in ListSessionsInput) (dtos.SessionListResponse, error)
	GetByID(ctx context.Context, id string, v *entities.Viewer) (*dtos.SessionResponse, error)
	Create(ctx context.Context, in CreateSessionInput) (*dtos.Session, error)
	Edit(ctx context.Context, id string, v *entities.Viewer, in EditSessionInput) (*dtos.Session, error)
	Delete(ctx context.Context, id string, v *entities.Viewer) error
	ChangeStatus(ctx context.Context, id string, v *entities.Viewer, status dtos.SessionStatus) (*dtos.Session, error)

	ListPlayers(ctx context.Context, sessionID string, v *entities.Viewer) (*dtos.SessionPlayersResponse, error)
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

	GetCardData(ctx context.Context, masterIDs []string) ([]dtos.SessionCardData, error)
}

type sessionUsecase struct {
	repo    SessionRepository
	profile ProfileClient
}

func NewSessionUsecase(repo SessionRepository, profile ProfileClient) SessionUsecase {
	return &sessionUsecase{repo: repo, profile: profile}
}

// enrich fetches UserBrief for the given ids. On profile-service failure it
// logs and returns an empty map — callers degrade gracefully rather than 5xx.
func (uc *sessionUsecase) enrich(ctx context.Context, ids []string) map[string]dtos.UserBrief {
	briefs, err := uc.profile.GetBriefs(ctx, ids)
	if err != nil {
		log.Error().Err(err).Strs("ids", ids).Msg("profile enrich failed; returning empty users map")
		return map[string]dtos.UserBrief{}
	}
	if briefs == nil {
		return map[string]dtos.UserBrief{}
	}
	return briefs
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

func (uc *sessionUsecase) List(ctx context.Context, in ListSessionsInput) (dtos.SessionListResponse, error) {
	scope := in.Scope
	if scope == "" {
		scope = dtos.ScopeCatalog
	}
	fmt.Println(in)
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
					return dtos.SessionListResponse{}, ErrForbidden
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
					return dtos.SessionListResponse{}, ErrForbidden
				}
				playerID = in.Viewer.UserID
				targetIsViewer = true
			}
		default:
			return dtos.SessionListResponse{}, fmt.Errorf("%w: unknown scope %q", ErrInvalidData, scope)
	}

	statuses := resolveStatusFilter(in.Status, scope, targetIsViewer)
	if len(statuses) == 0 {
		return dtos.SessionListResponse{Items: []dtos.Session{}, Users: map[string]dtos.UserBrief{}}, nil
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
		return dtos.SessionListResponse{}, mapRepoErr("list sessions", err)
	}
	fmt.Println("hi")

	masterIDs := make([]string, 0, len(items))
	for _, s := range items {
		masterIDs = append(masterIDs, s.MasterID)
	}
	users := uc.enrich(ctx, masterIDs)

	return dtos.SessionListResponse{Items: items, NextCursor: nextCursor, Users: users}, nil
}

func (uc *sessionUsecase) GetByID(ctx context.Context, id string, v *entities.Viewer) (*dtos.SessionResponse, error) {
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

	// Drafts have no players in practice; skip the fetch.
	var players []dtos.SessionPlayer
	if s.Status != dtos.Draft {
		players, err = uc.repo.ListPlayers(ctx, id)
		if err != nil {
			return nil, mapRepoErr("list players for get session by id", err)
		}
	}
	if players == nil {
		players = []dtos.SessionPlayer{}
	}

	ids := make([]string, 0, 1+len(players))
	ids = append(ids, s.MasterID)
	for _, p := range players {
		ids = append(ids, p.PlayerID)
	}
	users := uc.enrich(ctx, ids)

	return &dtos.SessionResponse{Session: *s, Players: players, Users: users}, nil
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

func (uc *sessionUsecase) ListPlayers(ctx context.Context, sessionID string, v *entities.Viewer) (*dtos.SessionPlayersResponse, error) {
	session, err := uc.repo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, mapRepoErr("get session for list players", err)
	}

	players, err := uc.repo.ListPlayers(ctx, sessionID)
	if err != nil {
		return nil, mapRepoErr("list session players", err)
	}

	if session.Status == dtos.Draft {
		return nil, ErrInvalidData
	}

	if session.Availability == dtos.Private {
		allowed := v.CanActAs(session.MasterID)
		if !allowed {
			for _, p := range players {
				if v.Is(p.PlayerID) {
					allowed = true
					break
				}
			}
		}
		if !allowed {
			return nil, ErrForbidden
		}
	}

	if players == nil {
		players = []dtos.SessionPlayer{}
	}
	ids := make([]string, 0, len(players))
	for _, p := range players {
		ids = append(ids, p.PlayerID)
	}
	users := uc.enrich(ctx, ids)

	return &dtos.SessionPlayersResponse{Players: players, Users: users}, nil
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

// --- card-data --------------------------------------------------------------

const maxCardDataBatch = 50

// GetCardData returns SessionCardData per requested master id. Every requested
// id appears in the result (in the same order) — masters with no public
// sessions get an entry with empty SystemStats and nil NextSession.
func (uc *sessionUsecase) GetCardData(ctx context.Context, masterIDs []string) ([]dtos.SessionCardData, error) {
	if len(masterIDs) == 0 {
		return []dtos.SessionCardData{}, nil
	}
	if len(masterIDs) > maxCardDataBatch {
		return nil, fmt.Errorf("%w: at most %d masterId values per request", ErrInvalidData, maxCardDataBatch)
	}

	// dedupe while preserving order
	seen := make(map[string]struct{}, len(masterIDs))
	deduped := make([]string, 0, len(masterIDs))
	for _, id := range masterIDs {
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		deduped = append(deduped, id)
	}

	stats, err := uc.repo.GetSystemStats(ctx, deduped)
	if err != nil {
		return nil, mapRepoErr("get system stats", err)
	}
	next, err := uc.repo.GetNextSessions(ctx, deduped)
	if err != nil {
		return nil, mapRepoErr("get next sessions", err)
	}

	out := make([]dtos.SessionCardData, 0, len(deduped))
	for _, id := range deduped {
		out = append(out, dtos.SessionCardData{
			UserID:      id,
			SystemStats: stats[id],
			NextSession: next[id],
		})
	}
	return out, nil
}
