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

// --- sessions ---------------------------------------------------------------

var sessionColumns = []string{
	"s.id", "s.title", "s.format", "s.scheduled_at", "s.durationHours",
	"s.address", "s.lat", "s.lng",
	"s.type", "s.availability",
	"COALESCE(s.description, '')", "COALESCE(s.preview_url, '')",
	"s.max_seats", "s.master_id", "s.price", "COALESCE(s.master_notes, '')",
	"s.status", "s.free_seats", "s.created_at", "s.updated_at",
	// embedded game system (join on game_systems gs)
	"gs.slug", "gs.canonical_name", "COALESCE(gs.badge_color, '')", "gs.is_curated",
}

func scanSession(row pgx.Row, s *dtos.Session) error {
	return nil
}

// --- campaigns --------------------------------------------------------------

var campaignColumns = []string{
	"c.id", "c.title", "c.description", "c.master_id", "c.status",
	"c.created_at", "c.updated_at",
	"gs.slug", "gs.canonical_name", "COALESCE(gs.badge_color, '')", "gs.is_curated",
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
	// optional character (LEFT JOIN characters ch ON ch.id = sp.character_id)
	"ch.id", "ch.player_id", "ch.name", "ch.class", "ch.level",
	"ch.avatar_url", "ch.description", "ch.sheet_url",
	"ch.created_at", "ch.updated_at",
}

func scanSessionPlayer(row pgx.Row, p *dtos.SessionPlayer) error {
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
