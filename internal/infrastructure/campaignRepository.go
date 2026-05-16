package infrastructure

import (
	"context"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/itzLilix/questboard-session-service/internal/entities"
	"github.com/itzLilix/questboard-shared/dtos"
	"github.com/jackc/pgx/v5/pgxpool"
)

type campaignRepository struct {
	db   *pgxpool.Pool
	psql sq.StatementBuilderType
}

type ListCampaignsParams struct {
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

type CreateCampaignParams struct {
	Title       string
	Description *string
	MasterID    string
	SystemID    string
}

type UpdateCampaignParams struct {
	Title       *string
	Description *string
	SystemID    *string
}

type TieSessionParams struct {
	CampaignID       string
	SessionID        string
	OrderIndex       *int
	BriefDescription *string
}

type EditTieParams struct {
	CampaignID        string
	SessionID         string
	OrderIndex        *int
	BriefDescription  *string
	CachedTitle       *string
	CachedScheduledAt *time.Time
}

func NewCampaignRepository(db *pgxpool.Pool, psql sq.StatementBuilderType) *campaignRepository {
	return &campaignRepository{db: db, psql: psql}
}

func (r *campaignRepository) List(ctx context.Context, p ListCampaignsParams) ([]dtos.Campaign, string, error) {
	return nil, "", nil
}

func (r *campaignRepository) GetByID(ctx context.Context, id string) (*dtos.Campaign, error) {
	return nil, ErrNotFound
}

func (r *campaignRepository) Create(ctx context.Context, p *CreateCampaignParams) (*dtos.Campaign, error) {
	return nil, nil
}

func (r *campaignRepository) Update(ctx context.Context, id string, p *UpdateCampaignParams) (*dtos.Campaign, error) {
	return nil, nil
}

func (r *campaignRepository) Delete(ctx context.Context, id string) error {
	return nil
}

func (r *campaignRepository) UpdateStatus(ctx context.Context, id string, status dtos.CampaignStatus) error {
	return nil
}

func (r *campaignRepository) ListSessions(ctx context.Context, campaignID string) ([]dtos.CampaignSessionTie, error) {
	return nil, nil
}

func (r *campaignRepository) TieSession(ctx context.Context, p *TieSessionParams) error {
	return nil
}

func (r *campaignRepository) EditTie(ctx context.Context, p *EditTieParams) error {
	return nil
}

func (r *campaignRepository) UntieSession(ctx context.Context, campaignID, sessionID string) error {
	return nil
}

func (r *campaignRepository) ListPlayers(ctx context.Context, campaignID string) ([]dtos.SessionPlayer, error) {
	return nil, nil
}
