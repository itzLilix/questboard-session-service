package usecase

import (
	"context"
	"time"

	"github.com/itzLilix/questboard-session-service/internal/entities"
	"github.com/itzLilix/questboard-shared/dtos"
)

type SessionUsecase interface {
	List(ctx context.Context, in ListSessionsInput) (dtos.Page[dtos.Session], error)
	GetByID(ctx context.Context, id string, v *entities.Viewer) (*dtos.Session, error)
	Create(ctx context.Context, in CreateSessionInput) (*dtos.Session, error)
	Edit(ctx context.Context, id string, v *entities.Viewer, in EditSessionInput) (*dtos.Session, error)
	Delete(ctx context.Context, id string, v *entities.Viewer) error
	ChangeStatus(ctx context.Context, id string, v *entities.Viewer, status dtos.SessionStatus) error

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

// --- input shapes -----------------------------------------------------------

type ListSessionsInput struct {
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
	Viewer       *entities.Viewer
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
	ScheduledAt   time.Time
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

// --- method stubs -----------------------------------------------------------

func (uc *sessionUsecase) List(ctx context.Context, in ListSessionsInput) (dtos.Page[dtos.Session], error) {
	return dtos.Page[dtos.Session]{}, ErrNotFound
}

func (uc *sessionUsecase) GetByID(ctx context.Context, id string, v *entities.Viewer) (*dtos.Session, error) {
	return nil, ErrNotFound
}

func (uc *sessionUsecase) Create(ctx context.Context, in CreateSessionInput) (*dtos.Session, error) {
	return nil, ErrNotFound
}

func (uc *sessionUsecase) Edit(ctx context.Context, id string, v *entities.Viewer, in EditSessionInput) (*dtos.Session, error) {
	return nil, ErrNotFound
}

func (uc *sessionUsecase) Delete(ctx context.Context, id string, v *entities.Viewer) error {
	return ErrNotFound
}

func (uc *sessionUsecase) ChangeStatus(ctx context.Context, id string, v *entities.Viewer, status dtos.SessionStatus) error {
	return ErrNotFound
}

func (uc *sessionUsecase) ListPlayers(ctx context.Context, sessionID string, v *entities.Viewer) ([]dtos.SessionPlayer, error) {
	return nil, ErrNotFound
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

func (uc *sessionUsecase) Apply(ctx context.Context, sessionID string, v *entities.Viewer, message *string) (*dtos.SessionApplication, error) {
	return nil, ErrNotFound
}

func (uc *sessionUsecase) ListApplications(ctx context.Context, sessionID string, v *entities.Viewer) ([]dtos.SessionApplication, error) {
	return nil, ErrNotFound
}

func (uc *sessionUsecase) ResolveApplication(ctx context.Context, applicationID string, v *entities.Viewer, status dtos.SessionApplicationStatus) error {
	return ErrNotFound
}

func (uc *sessionUsecase) ListFiles(ctx context.Context, sessionID string, v *entities.Viewer) ([]dtos.SessionFile, error) {
	return nil, ErrNotFound
}

func (uc *sessionUsecase) UploadFile(ctx context.Context, in UploadFileInput) (*dtos.SessionFile, error) {
	return nil, ErrNotFound
}

func (uc *sessionUsecase) DeleteFile(ctx context.Context, fileID string, v *entities.Viewer) error {
	return ErrNotFound
}

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
