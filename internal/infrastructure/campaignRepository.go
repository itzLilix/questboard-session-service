package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/itzLilix/questboard-session-service/internal/entities"
	"github.com/itzLilix/questboard-shared/dtos"
	"github.com/jackc/pgx/v5"
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
	Title        string
	Description  *string
	MasterID     string
	SystemID     *string
	Status       dtos.CampaignStatus
	Availability dtos.SessionAvailability
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
	selectSQL, selectArgs, err := r.psql.
		Select(campaignColumns...).
		From("campaigns c").
		LeftJoin("game_systems gs ON gs.id = c.system_id").
		Where(sq.Eq{"c.id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select campaign: %w", err)
	}

	var (
		camp        dtos.Campaign
		description sql.NullString
		sysID       sql.NullString
		sysSlug     sql.NullString
		sysName     sql.NullString
		sysBadge    sql.NullString
		sysCurated  sql.NullBool
	)
	if err := r.db.QueryRow(ctx, selectSQL, selectArgs...).Scan(
		&camp.ID, &camp.Title, &description, &camp.MasterID, &camp.Status, &camp.Availability,
		&camp.CreatedAt, &camp.UpdatedAt,
		&sysID, &sysSlug, &sysName, &sysBadge, &sysCurated,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("read campaign: %w", err)
	}

	if description.Valid {
		s := description.String
		camp.Description = &s
	}
	if sysID.Valid {
		camp.System = dtos.GameSystem{
			Id:         sysID.String,
			Slug:       sysSlug.String,
			Name:       sysName.String,
			BadgeColor: sysBadge.String,
			IsCurated:  sysCurated.Bool,
		}
	}

	return &camp, nil
}

func (r *campaignRepository) Create(ctx context.Context, p *CreateCampaignParams) (*dtos.Campaign, error) {
	if p == nil {
		return nil, fmt.Errorf("create campaign: nil params")
	}

	status := p.Status
	if status == "" {
		status = dtos.CampaignActive
	}
	availability := p.Availability
	if availability == "" {
		availability = dtos.Open
	}

	var (
		descArg any = nil
		sysArg  any = nil
	)
	if p.Description != nil {
		descArg = nullString(*p.Description)
	}
	if p.SystemID != nil && *p.SystemID != "" {
		sysArg = *p.SystemID
	}

	insertSQL, insertArgs, err := r.psql.
		Insert("campaigns").
		Columns("title", "description", "master_id", "system_id", "status", "availability").
		Values(p.Title, descArg, p.MasterID, sysArg, status, availability).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build insert campaign: %w", err)
	}

	var newID string
	if err := r.db.QueryRow(ctx, insertSQL, insertArgs...).Scan(&newID); err != nil {
		return nil, fmt.Errorf("insert campaign: %w", err)
	}

	return r.GetByID(ctx, newID)
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
