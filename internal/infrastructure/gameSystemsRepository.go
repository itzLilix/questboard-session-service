package infrastructure

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/itzLilix/questboard-shared/dtos"
	"github.com/jackc/pgx/v5/pgxpool"
)

type gameSystemsRepository struct {
	db   *pgxpool.Pool
	psql sq.StatementBuilderType
}

func NewGameSystemsRepository(db *pgxpool.Pool, psql sq.StatementBuilderType) *gameSystemsRepository {
	return &gameSystemsRepository{db: db, psql: psql}
}

func (r *gameSystemsRepository) GetCurated() ([]dtos.GameSystem, error) {
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

	rows, err := r.db.Query(context.Background(), query, args...)
	if err != nil {
		return nil, fmt.Errorf("query curated systems: %w", err)
	}
	defer rows.Close()

	systems := make([]dtos.GameSystem, 0)
	for rows.Next() {
		gs, err := scanGameSystem(rows)
		if err != nil {
			return nil, fmt.Errorf("scan curated system: %w", err)
		}
		systems = append(systems, gs)
	}

	return systems, nil
}

func (r *gameSystemsRepository) Search(q string) ([]dtos.GameSystem, error) {
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

	rows, err := r.db.Query(context.Background(), query, args...)
	if err != nil {
		return nil, fmt.Errorf("query search systems: %w", err)
	}
	defer rows.Close()

	systems := make([]dtos.GameSystem, 0)
	for rows.Next() {
		gs, err := scanGameSystem(rows)
		if err != nil {
			return nil, fmt.Errorf("scan search system: %w", err)
		}
		systems = append(systems, gs)
	}

	return systems, nil
}
