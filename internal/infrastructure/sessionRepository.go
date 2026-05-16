package infrastructure

import (
	"context"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/itzLilix/questboard-session-service/internal/entities"
	"github.com/itzLilix/questboard-shared/dtos"
	"github.com/jackc/pgx/v5/pgxpool"
)

type sessionRepository struct {
	db   *pgxpool.Pool
	psql sq.StatementBuilderType
}

type ListSessionsParams struct {
	Viewer        *entities.Viewer
	Search        string
	Format        string
	Type          string
	City          string
	SystemID      string
	HasFreeSeats  bool
	PriceMin      *float64
	PriceMax      *float64
	DateFrom      *time.Time
	DateTo        *time.Time
	Sort          string
	SortOrder     string
	Cursor        string
	Limit         int
	IncludeDrafts bool
}

type CreateSessionParams struct {
	Title         string
	Description   string
	Address       string
	MasterNotes   string
	PreviewURL    string
	Format        dtos.SessionFormat
	Availability  dtos.SessionAvailability
	SystemID      string
	MasterID      string
	ScheduledAt   time.Time
	DurationHours *float64
	Lat           *float64
	Lng           *float64
	MaxSeats      int16
	Price         float64
}

type UpdateSessionParams struct {
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

type CreateApplicationParams struct {
	SessionID   string
	ApplicantID string
	Message     *string
}

type AddSessionFileParams struct {
	SessionID  string
	UploaderID string
	Filename   string
	URL        string
	MimeType   string
	SizeBytes  int64
}

type AddCommentParams struct {
	SessionID string
	AuthorID  string
	Text      string
}

func NewSessionRepository(db *pgxpool.Pool, psql sq.StatementBuilderType) *sessionRepository {
	return &sessionRepository{db: db, psql: psql}
}

// --- sessions ---------------------------------------------------------------

func (r *sessionRepository) List(ctx context.Context, p ListSessionsParams) ([]dtos.Session, string, error) {
	return nil, "", nil
}

func (r *sessionRepository) GetByID(ctx context.Context, id string) (*dtos.Session, error) {
	return nil, ErrNotFound
}

func (r *sessionRepository) Create(ctx context.Context, p *CreateSessionParams) (*dtos.Session, error) {
	return nil, nil
}

func (r *sessionRepository) Update(ctx context.Context, id string, p *UpdateSessionParams) (*dtos.Session, error) {
	return nil, nil
}

func (r *sessionRepository) Delete(ctx context.Context, id string) error {
	return nil
}

func (r *sessionRepository) UpdateStatus(ctx context.Context, id string, status dtos.SessionStatus) error {
	return nil
}

// --- players ----------------------------------------------------------------

func (r *sessionRepository) ListPlayers(ctx context.Context, sessionID string) ([]dtos.SessionPlayer, error) {
	return nil, nil
}

func (r *sessionRepository) Join(ctx context.Context, sessionID, playerID string, characterID *string) error {
	return nil
}

func (r *sessionRepository) Leave(ctx context.Context, sessionID, playerID string) error {
	return nil
}

func (r *sessionRepository) Kick(ctx context.Context, sessionID, playerID string) error {
	return nil
}

func (r *sessionRepository) SetCharacter(ctx context.Context, sessionID, playerID string, characterID *string) error {
	return nil
}

// --- applications -----------------------------------------------------------

func (r *sessionRepository) CreateApplication(ctx context.Context, p *CreateApplicationParams) (*dtos.SessionApplication, error) {
	return nil, nil
}

func (r *sessionRepository) ListApplications(ctx context.Context, sessionID string) ([]dtos.SessionApplication, error) {
	return nil, nil
}

func (r *sessionRepository) GetApplication(ctx context.Context, applicationID string) (*dtos.SessionApplication, error) {
	return nil, ErrNotFound
}

func (r *sessionRepository) ResolveApplication(ctx context.Context, applicationID string, status dtos.SessionApplicationStatus) error {
	return nil
}

// --- files ------------------------------------------------------------------

func (r *sessionRepository) ListFiles(ctx context.Context, sessionID string) ([]dtos.SessionFile, error) {
	return nil, nil
}

func (r *sessionRepository) AddFile(ctx context.Context, p *AddSessionFileParams) (*dtos.SessionFile, error) {
	return nil, nil
}

func (r *sessionRepository) GetFile(ctx context.Context, fileID string) (*dtos.SessionFile, error) {
	return nil, ErrNotFound
}

func (r *sessionRepository) DeleteFile(ctx context.Context, fileID string) error {
	return nil
}

// --- comments ---------------------------------------------------------------

func (r *sessionRepository) ListComments(ctx context.Context, sessionID string) ([]dtos.SessionCommentary, error) {
	return nil, nil
}

func (r *sessionRepository) AddComment(ctx context.Context, p *AddCommentParams) (*dtos.SessionCommentary, error) {
	return nil, nil
}

func (r *sessionRepository) GetComment(ctx context.Context, commentID string) (*dtos.SessionCommentary, error) {
	return nil, ErrNotFound
}

func (r *sessionRepository) UpdateComment(ctx context.Context, commentID, text string) (*dtos.SessionCommentary, error) {
	return nil, nil
}

func (r *sessionRepository) DeleteComment(ctx context.Context, commentID string) error {
	return nil
}
