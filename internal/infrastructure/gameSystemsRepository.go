package infrastructure

import (
	"context"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/itzLilix/questboard-shared/dtos"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type gameSystemsRepository struct {
	db   *pgxpool.Pool
	psql sq.StatementBuilderType
}

type CreateGameSystemParams struct {
    Slug string
    Name string
	IsCurated bool
	BadgeColor *string
}

func NewGameSystemsRepository(db *pgxpool.Pool, psql sq.StatementBuilderType) *gameSystemsRepository {
	return &gameSystemsRepository{db: db, psql: psql}
}

func (r *gameSystemsRepository) GetAll(ctx context.Context) ([]dtos.GameSystem, error) {
	query, args, err := r.psql.
		Select(gameSystemColumns...).
		From("game_systems gs").
		LeftJoin("sessions s ON s.system_id = gs.id").
		GroupBy("gs.id").
		OrderBy("COUNT(s.id) DESC", "gs.canonical_name ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get all query: %w", err)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query all systems: %w", err)
	}
	defer rows.Close()

	systems := make([]dtos.GameSystem, 0)
	for rows.Next() {
		var gs dtos.GameSystem
		if err := scanGameSystem(rows, &gs); err != nil {
			return nil, fmt.Errorf("scan system: %w", err)
		}
		systems = append(systems, gs)
	}

	return systems, nil
}

func (r *gameSystemsRepository) GetCurated(ctx context.Context) ([]dtos.GameSystem, error) {
	query, args, err := r.psql.
		Select(gameSystemColumns...).
		From("game_systems gs").
		LeftJoin("sessions s ON s.system_id = gs.id").
		Where(sq.Eq{"gs.is_curated": true}).
		GroupBy("gs.id").
		OrderBy("COUNT(s.id) DESC", "gs.canonical_name ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build curated query: %w", err)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query curated systems: %w", err)
	}
	defer rows.Close()

	systems := make([]dtos.GameSystem, 0)
	for rows.Next() {
		var gs dtos.GameSystem
		err := scanGameSystem(rows, &gs)
		if err != nil {
			return nil, fmt.Errorf("scan curated system: %w", err)
		}
		systems = append(systems, gs)
	}

	return systems, nil
}

func (r *gameSystemsRepository) Search(ctx context.Context, q string) ([]dtos.GameSystem, error) {
	query, args, err := r.psql.
		Select(gameSystemColumns...).
		From("game_systems gs").
		LeftJoin("sessions s ON s.system_id = gs.id").
		Where("gs.canonical_name ILIKE ? OR gs.slug ILIKE ?", "%"+q+"%", "%"+q+"%").
		GroupBy("gs.id").
		OrderBy("COUNT(s.id) DESC", "gs.canonical_name ASC").
		Limit(20).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build search query: %w", err)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query search systems: %w", err)
	}
	defer rows.Close()

	systems := make([]dtos.GameSystem, 0)
	for rows.Next() {
		var gs dtos.GameSystem
		err := scanGameSystem(rows, &gs)
		if err != nil {
			return nil, fmt.Errorf("scan search system: %w", err)
		}
		systems = append(systems, gs)
	}

	return systems, nil
}

func (r *gameSystemsRepository) AddGameSystem(ctx context.Context, params *CreateGameSystemParams) (*dtos.GameSystem, error) {
	sql, args, err := r.psql.Insert("game_systems").
		Columns("slug", "canonical_name", "badge_color", "is_curated").
		Values(params.Slug, params.Name, params.BadgeColor, params.IsCurated).
		Suffix("RETURNING id, slug, canonical_name, COALESCE(badge_color, ''), is_curated").
		ToSql()

	if err != nil {
		return nil, fmt.Errorf("add game system: %w", err)
	}
	
	row := r.db.QueryRow(ctx, sql, args...)
	gs := &dtos.GameSystem{}
	if err := scanGameSystem(row, gs); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrAlreadyExists
		}
		return nil, fmt.Errorf("add game system: %w", err)
	}

	return gs, nil
}