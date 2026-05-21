package infrastructure

import (
	"database/sql"

	"github.com/itzLilix/questboard-shared/dtos"
	"github.com/jackc/pgx/v5"
)

var gameSystemColumns = []string{
	"gs.id", "gs.slug", "gs.canonical_name", "COALESCE(gs.badge_color, '')", "gs.is_curated",
}

func scanGameSystem(rows pgx.Row, gs *dtos.GameSystem) error {
	err := rows.Scan(&gs.Id, &gs.Slug, &gs.Name, &gs.BadgeColor, &gs.IsCurated)
	return err
}

// --- sessions ---------------------------------------------------------------

var sessionColumns = []string{
	"s.id", "s.title", "s.format", "s.scheduled_at", "s.duration_hours",
	"s.address", "s.lat", "s.lng",
	"s.type", "s.availability",
	"COALESCE(s.description, '')", "COALESCE(s.preview_url, '')",
	"s.max_seats", "s.master_id", "s.price", "COALESCE(s.master_notes, '')",
	"s.status", "s.free_seats", "s.created_at", "s.updated_at",
	// embedded game system (join on game_systems gs)
	"gs.id", "gs.slug", "gs.canonical_name", "COALESCE(gs.badge_color, '')", "gs.is_curated",
}

func scanSession(row pgx.Row, s *dtos.Session) error {
	var (
		scheduledAt         sql.NullTime
		duration            sql.NullFloat64
		address             sql.NullString
		lat, lng            sql.NullFloat64
		maxSeats, freeSeats int16
		price               float64
	)
	err := row.Scan(
		&s.Id,
		&s.Title,
		&s.Format,
		&scheduledAt,
		&duration,
		&address,
		&lat,
		&lng,
		&s.Type,
		&s.Availability,
		&s.Description,
		&s.PreviewUrl,
		&maxSeats,
		&s.MasterID,
		&price,
		&s.MasterNotes,
		&s.Status,
		&freeSeats,
		&s.CreatedAt,
		&s.UpdatedAt,
		&s.System.Id,
		&s.System.Slug,
		&s.System.Name,
		&s.System.BadgeColor,
		&s.System.IsCurated,
	)
	if err != nil {
		return err
	}
	s.MaxSeats = uint8(maxSeats)
	s.FreeSeats = uint8(freeSeats)
	s.Price = price
	if scheduledAt.Valid {
		t := scheduledAt.Time
		s.ScheduledAt = &t
	}
	if duration.Valid {
		d := duration.Float64
		s.Duration = &d
	}
	if address.Valid && address.String != "" {
		s.Location = &dtos.Location{Address: address.String}
		if lat.Valid {
			s.Location.Lat = lat.Float64
		}
		if lng.Valid {
			s.Location.Lng = lng.Float64
		}
	}
	return nil
}

// --- campaigns --------------------------------------------------------------

var campaignColumns = []string{
	"c.id", "c.title", "c.description", "c.master_id", "c.status",
	"c.created_at", "c.updated_at",
	"gs.id", "gs.slug", "gs.canonical_name", "COALESCE(gs.badge_color, '')", "gs.is_curated",
}

func scanCampaign(row pgx.Row, c *dtos.Campaign) error {
	return nil
}

var campaignSessionTieColumns = []string{
	"cs.campaign_id", "cs.session_id", "cs.order_index",
	"COALESCE(cs.cached_title, '')", "cs.cached_scheduled_at",
	"cs.brief_description",
}

func scanCampaignSessionTie(row pgx.Row, t *dtos.CampaignSessionTie) error {
	return nil
}

// --- session players (+ optional character) ---------------------------------

var sessionPlayerColumns = []string{
	"sp.session_id", "sp.player_id", "sp.status", "sp.joined_at",
	// character
	"ch.id", "ch.player_id", "ch.name", "ch.class", "ch.level",
	"ch.avatar_url", "ch.description", 
	//"ch.sheet_url",
	"ch.created_at", "ch.updated_at",
}

func scanSessionPlayer(row pgx.Row, p *dtos.SessionPlayer) error {
	// character fields come from a LEFT JOIN — all null when sp.character_id is null
	var (
		chID          sql.NullString
		chPlayerID    sql.NullString
		chName        sql.NullString
		chClass       sql.NullString
		chLevel       sql.NullInt16
		chAvatarURL   sql.NullString
		chDescription sql.NullString
		chSheetURL    sql.NullString
		chCreatedAt   sql.NullTime
		chUpdatedAt   sql.NullTime
	)
	err := row.Scan(
		&p.SessionID,
		&p.PlayerID,
		&p.Status,
		&p.JoinedAt,
		&chID,
		&chPlayerID,
		&chName,
		&chClass,
		&chLevel,
		&chAvatarURL,
		&chDescription,
		&chSheetURL,
		&chCreatedAt,
		&chUpdatedAt,
	)
	if err != nil {
		return err
	}

	if !chID.Valid {
		return nil
	}

	c := &dtos.Character{
		ID:        chID.String,
		PlayerID:  chPlayerID.String,
		Name:      chName.String,
		CreatedAt: chCreatedAt.Time,
		UpdatedAt: chUpdatedAt.Time,
	}
	if chClass.Valid {
		v := chClass.String
		c.Class = &v
	}
	if chLevel.Valid {
		v := int(chLevel.Int16)
		c.Level = &v
	}
	if chAvatarURL.Valid {
		v := chAvatarURL.String
		c.AvatarURL = &v
	}
	if chDescription.Valid {
		v := chDescription.String
		c.Description = &v
	}
	if chSheetURL.Valid {
		v := chSheetURL.String
		c.SheetURL = &v
	}
	p.Character = c
	return nil
}

// --- session applications ---------------------------------------------------

var sessionApplicationColumns = []string{
	"a.id", "a.session_id", "a.applicant_id", "a.status",
	"a.message", "a.created_at", "a.resolved_at",
}

func scanSessionApplication(row pgx.Row, a *dtos.SessionApplication) error {
	return nil
}

// --- session files ----------------------------------------------------------

var sessionFileColumns = []string{
	"f.id", "f.session_id", "f.uploader_id", "f.filename",
	"f.url", "f.mime_type", "f.size_bytes", "f.uploaded_at",
}

func scanSessionFile(row pgx.Row, f *dtos.SessionFile) error {
	return nil
}

// --- session commentaries ---------------------------------------------------

var sessionCommentaryColumns = []string{
	"sc.id", "sc.session_id", "sc.author_id", "sc.text",
	"sc.created_at", "sc.updated_at",
}

func scanSessionCommentary(row pgx.Row, c *dtos.SessionCommentary) error {
	return nil
}

// --- characters -------------------------------------------------------------

var characterColumns = []string{
	"ch.id", "ch.player_id", "ch.name", "ch.class", "ch.level",
	"ch.avatar_url", "ch.description", "ch.sheet_url",
	"ch.created_at", "ch.updated_at",
}

func scanCharacter(row pgx.Row, c *dtos.Character) error {
	return nil
}
