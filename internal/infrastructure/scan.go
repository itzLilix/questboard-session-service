package infrastructure

import (
	"github.com/itzLilix/questboard-shared/dtos"
	"github.com/jackc/pgx/v5"
)

var gameSystemColumns = []string{
	"gs.slug", "gs.canonical_name", "COALESCE(gs.badge_color, '')", "gs.is_curated",
}

func scanGameSystem(rows pgx.Row, gs *dtos.GameSystem) error {
	err := rows.Scan(&gs.Slug, &gs.Name, &gs.BadgeColor, &gs.IsCurated)
	return err
}
