package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/itzLilix/questboard-session-service/internal/entities"
	"github.com/itzLilix/questboard-shared/cursor"
	"github.com/itzLilix/questboard-shared/dtos"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type sessionRepository struct {
	db   *pgxpool.Pool
	psql sq.StatementBuilderType
}

type ListSessionsParams struct {
	Viewer *entities.Viewer

	Scope          dtos.SessionScope
	MasterID       string
	PlayerID       string
	Status         []dtos.SessionStatus
	TargetIsViewer bool

	Search       string
	Format       dtos.SessionFormat
	Type         dtos.SessionType
	City         string
	SystemID     string
	HasFreeSeats bool
	PriceMin     *float64
	PriceMax     *float64
	DateFrom     *time.Time
	DateTo       *time.Time
	Sort         dtos.SessionListSort
	SortOrder    dtos.SortOrder
	Cursor       string
	Limit        int
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
	ScheduledAt   *time.Time
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

var sortColumns = map[dtos.SessionListSort]string{
	dtos.SortSessionScheduledAt: "s.scheduled_at",
	dtos.SortSessionCreatedAt:   "s.created_at",
	dtos.SortSessionPrice:       "s.price",
	dtos.SortSessionTitle:       "s.title",
	dtos.SortSessionSystem:      "gs.is_curated, gs.canonical_name",
}

func (r *sessionRepository) List(ctx context.Context, p ListSessionsParams) ([]dtos.Session, string, error) {
	q := r.psql.
		Select(sessionColumns...).
		From("sessions s").
		Join("game_systems gs ON gs.id = s.system_id")

	// --- scope-specific WHERE ---
	switch p.Scope {
		case dtos.ScopeCatalog:
			q = q.Where(sq.NotEq{"s.availability": dtos.Private}).
				Where("(s.scheduled_at IS NULL OR s.scheduled_at >= NOW())")
		case dtos.ScopeMastering:
			q = q.Where(sq.Eq{"s.master_id": p.MasterID})
		case dtos.ScopePlaying:
			q = q.Where(sq.Expr(
				"EXISTS (SELECT 1 FROM session_players sp WHERE sp.session_id = s.id AND sp.player_id = ? AND sp.status = ?)",
				p.PlayerID, dtos.PlayerActive,
			))
		default:
			return nil, "", fmt.Errorf("unknown scope %q", p.Scope)
	}

	// --- visibility predicate (skipped when target == viewer or in catalog) ---
	if !p.TargetIsViewer && p.Scope != dtos.ScopeCatalog {
		visibility := sq.Or{
			sq.And{
				sq.Expr("s.status NOT IN (?, ?)", dtos.Draft, dtos.Cancelled),
				sq.NotEq{"s.availability": dtos.Private},
			},
		}
		if p.Viewer.IsAuthenticated() {
			visibility = append(visibility, sq.Eq{"s.master_id": p.Viewer.UserID})
			visibility = append(visibility, sq.Expr(
				"EXISTS (SELECT 1 FROM session_players sp WHERE sp.session_id = s.id AND sp.player_id = ? AND sp.status = ?)",
				p.Viewer.UserID, dtos.PlayerActive,
			))
		}
		q = q.Where(visibility)
	}

	// --- universal filters ---
	q = q.Where(sq.Eq{"s.status": p.Status})
	if p.Search != "" {
		q = q.Where("s.title ILIKE ?", "%"+p.Search+"%")
	}
	if p.Format != "" {
		q = q.Where(sq.Eq{"s.format": p.Format})
	}
	if p.Type != "" {
		q = q.Where(sq.Eq{"s.type": p.Type})
	}
	if p.City != "" {
		q = q.Where("s.address ILIKE ?", "%"+p.City+"%")
	}
	if p.SystemID != "" {
		q = q.Where(sq.Eq{"s.system_id": p.SystemID})
	}
	if p.HasFreeSeats {
		q = q.Where(sq.Gt{"s.free_seats": 0})
	}
	if p.PriceMin != nil {
		q = q.Where(sq.GtOrEq{"s.price": *p.PriceMin})
	}
	if p.PriceMax != nil {
		q = q.Where(sq.LtOrEq{"s.price": *p.PriceMax})
	}
	if p.DateFrom != nil {
		q = q.Where(sq.GtOrEq{"s.scheduled_at": *p.DateFrom})
	}
	if p.DateTo != nil {
		q = q.Where(sq.Lt{"s.scheduled_at": *p.DateTo})
	}

	// --- sort resolution ---
	sortKey := p.Sort
	if _, ok := sortColumns[sortKey]; !ok {
		// scope-specific defaults
		if p.Scope == dtos.ScopeCatalog {
			sortKey = "scheduled_at"
		} else {
			sortKey = "created_at"
		}
	}
	sortCol := sortColumns[sortKey]

	// --- cursor ---
	cursor, err := cursor.DecodeCursor[sessionCursor](p.Cursor)
	if err != nil {
		return nil, "", err
	}
	q, err = applyCursor(q, cursor, sortKey, p.SortOrder)
	if err != nil {
		return nil, "", err
	}

	// --- ORDER BY + tiebreaker ---
	
	nulls := "NULLS LAST"
	if p.SortOrder == dtos.SortDesc {
		nulls = "NULLS FIRST"
	}
	q = q.OrderBy(
		fmt.Sprintf("%s %s %s", sortCol, p.SortOrder, nulls),
		fmt.Sprintf("s.id %s", p.SortOrder),
	)

	q = q.Limit(uint64(p.Limit + 1))

	query, args, err := q.ToSql()
	if err != nil {
		return nil, "", fmt.Errorf("build list sessions query: %w", err)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, "", fmt.Errorf("query list sessions: %w", err)
	}
	defer rows.Close()

	items := make([]dtos.Session, 0, p.Limit)
	for rows.Next() {
		var s dtos.Session
		if err := scanSession(rows, &s); err != nil {
			return nil, "", fmt.Errorf("scan list session row: %w", err)
		}
		items = append(items, s)
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("iterate list sessions: %w", err)
	}

	// --- pagination ---
	var nextCursor string
	if len(items) > p.Limit {
		items = items[:p.Limit] // drop the sentinel +1 row
		nc, err := buildNextCursor(items[p.Limit-1], sortKey, p.SortOrder)
		if err != nil {
			return nil, "", fmt.Errorf("encode next cursor: %w", err)
		}
		nextCursor = nc
	}
	return items, nextCursor, nil
}

func (r *sessionRepository) GetByID(ctx context.Context, id string) (*dtos.Session, error) {
	query, args, err := r.psql.
		Select(sessionColumns...).
		From("sessions s").
		Join("game_systems gs ON gs.id = s.system_id").
		Where(sq.Eq{"s.id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get session query: %w", err)
	}

	row := r.db.QueryRow(ctx, query, args...)
	s := &dtos.Session{}
	if err := scanSession(row, s); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("scan session: %w", err)
	}
	return s, nil
}

func (r *sessionRepository) Create(ctx context.Context, p *CreateSessionParams) (*dtos.Session, error) {
	insert, args, err := r.psql.
		Insert("sessions").
		Columns(
			"title", "format", "scheduled_at", "system_id", "max_seats", "master_id",
			"price", "availability", "free_seats",
			"address", "lat", "lng",
			"description", "preview_url", "master_notes",
			"duration_hours",
		).
		Values(
			p.Title, p.Format, p.ScheduledAt, p.SystemID, p.MaxSeats, p.MasterID,
			p.Price, p.Availability, p.MaxSeats,
			nullString(p.Address), p.Lat, p.Lng,
			nullString(p.Description), nullString(p.PreviewURL), nullString(p.MasterNotes),
			p.DurationHours,
		).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build insert session: %w", err)
	}

	var newID string
	if err := r.db.QueryRow(ctx, insert, args...).Scan(&newID); err != nil {
		return nil, fmt.Errorf("insert session: %w", err)
	}

	return r.GetByID(ctx, newID)
}

func (r *sessionRepository) Update(ctx context.Context, id string, p *UpdateSessionParams) (*dtos.Session, error) {
	upd := r.psql.Update("sessions").Where(sq.Eq{"id": id})

	if p.Title != nil {
		upd = upd.Set("title", *p.Title)
	}
	if p.Description != nil {
		upd = upd.Set("description", nullString(*p.Description))
	}
	if p.Address != nil {
		upd = upd.Set("address", nullString(*p.Address))
	}
	if p.MasterNotes != nil {
		upd = upd.Set("master_notes", nullString(*p.MasterNotes))
	}
	if p.PreviewURL != nil {
		upd = upd.Set("preview_url", nullString(*p.PreviewURL))
	}
	if p.Format != nil {
		upd = upd.Set("format", *p.Format)
		if *p.Format == dtos.Online {
			upd = upd.Set("address", nil).Set("lat", nil).Set("lng", nil)
		}
	}
	if p.Availability != nil {
		upd = upd.Set("availability", *p.Availability)
	}
	if p.SystemID != nil {
		upd = upd.Set("system_id", *p.SystemID)
	}
	if p.ScheduledAt != nil {
		upd = upd.Set("scheduled_at", *p.ScheduledAt)
	}
	if p.DurationHours != nil {
		upd = upd.Set("duration_hours", *p.DurationHours)
	}
	if p.Lat != nil {
		upd = upd.Set("lat", *p.Lat)
	}
	if p.Lng != nil {
		upd = upd.Set("lng", *p.Lng)
	}
	if p.MaxSeats != nil {
		upd = upd.Set("max_seats", *p.MaxSeats)
		upd = upd.Set("free_seats", sq.Expr("? - (max_seats - free_seats)", *p.MaxSeats))
	}
	if p.Price != nil {
		upd = upd.Set("price", *p.Price)
	}
	upd = upd.Set("updated_at", sq.Expr("NOW()"))

	query, args, err := upd.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build update session: %w", err)
	}

	tag, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("exec update session: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}

	return r.GetByID(ctx, id)
}

func (r *sessionRepository) Delete(ctx context.Context, id string) error {
	delete, args, err := r.psql.
		Delete("sessions").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete session: %w", err)
	}

	tag, err := r.db.Exec(ctx, delete, args...)
	if err != nil {
		return fmt.Errorf("exec delete session: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *sessionRepository) UpdateStatus(ctx context.Context, id string, status dtos.SessionStatus) error {
	query, args, err := r.psql.
		Update("sessions").
		Set("status", status).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update session status: %w", err)
	}

	tag, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("exec update session status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// --- players ----------------------------------------------------------------

func (r *sessionRepository) IsPlayer(ctx context.Context, sessionID, userID string) (bool, error) {
	query, args, err := r.psql.
		Select("sp.player_id").
		From("session_players sp").
		Join("sessions s ON s.id = sp.session_id").
		Where(sq.Eq{"s.id": sessionID, "sp.player_id": userID}).
		ToSql()
	if err != nil {
		return false, fmt.Errorf("build check participant query: %w", err)
	}

	var id string
	if err := r.db.QueryRow(ctx, query, args...).Scan(&id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("exec check participant query: %w", err)
	}
	return true, nil
}

func (r *sessionRepository) ListPlayers(ctx context.Context, sessionID string) ([]dtos.SessionPlayer, error) {
	query, args, err := r.psql.
		Select(sessionPlayerColumns...).
		From("session_players sp").
		LeftJoin("characters ch ON ch.id = sp.character_id").
		Where(sq.Eq{"sp.session_id": sessionID, "sp.status": dtos.PlayerActive}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list players query: %w", err)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("exec list players query: %w", err)
	}
	defer rows.Close()

	var players []dtos.SessionPlayer
	for rows.Next() {
		var p dtos.SessionPlayer
		if err := scanSessionPlayer(rows, &p); err != nil {
			return nil, fmt.Errorf("scan player row: %w", err)
		}
		players = append(players, p)
	}

	return players, nil
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

// --- card-data aggregates ---------------------------------------------------

// GetSystemStats returns, per master, the list of game systems they've run
// (status published/ongoing/completed, non-private), with a count per system,
// ordered by count desc. Output map is keyed by master_id.
func (r *sessionRepository) GetSystemStats(ctx context.Context, masterIDs []string) (map[string][]dtos.SystemStat, error) {
	out := make(map[string][]dtos.SystemStat, len(masterIDs))
	if len(masterIDs) == 0 {
		return out, nil
	}

	query, args, err := r.psql.
		Select(
			"s.master_id",
			"gs.id", "gs.slug", "gs.canonical_name", "COALESCE(gs.badge_color, '')", "gs.is_curated",
			"COUNT(*)",
		).
		From("sessions s").
		Join("game_systems gs ON gs.id = s.system_id").
		Where(sq.Eq{"s.master_id": masterIDs}).
		Where(sq.Eq{"s.status": []dtos.SessionStatus{dtos.Published, dtos.Ongoing, dtos.Completed}}).
		Where(sq.NotEq{"s.availability": dtos.Private}).
		GroupBy("s.master_id", "gs.id", "gs.slug", "gs.canonical_name", "gs.badge_color", "gs.is_curated").
		OrderBy("s.master_id", "COUNT(*) DESC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get system stats: %w", err)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query system stats: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			masterID string
			stat     dtos.SystemStat
		)
		if err := rows.Scan(
			&masterID,
			&stat.Id, &stat.Slug, &stat.Name, &stat.BadgeColor, &stat.IsCurated,
			&stat.SessionsCount,
		); err != nil {
			return nil, fmt.Errorf("scan system stats: %w", err)
		}
		out[masterID] = append(out[masterID], stat)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate system stats: %w", err)
	}
	return out, nil
}

func (r *sessionRepository) GetNextSessions(ctx context.Context, masterIDs []string) (map[string]*dtos.NextSession, error) {
	out := make(map[string]*dtos.NextSession, len(masterIDs))
	if len(masterIDs) == 0 {
		return out, nil
	}

	query, args, err := r.psql.
		Select(
			"DISTINCT ON (s.master_id) s.master_id",
			"s.id", "s.scheduled_at", "s.format", "s.type",
			"gs.id", "gs.slug", "gs.canonical_name", "COALESCE(gs.badge_color, '')", "gs.is_curated",
		).
		From("sessions s").
		Join("game_systems gs ON gs.id = s.system_id").
		Where(sq.Eq{"s.master_id": masterIDs}).
		Where(sq.Eq{"s.status": []dtos.SessionStatus{dtos.Published, dtos.Ongoing}}).
		Where(sq.NotEq{"s.availability": dtos.Private}).
		Where("s.scheduled_at IS NOT NULL AND s.scheduled_at >= NOW()").
		OrderBy("s.master_id", "s.scheduled_at ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get next sessions: %w", err)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query next sessions: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			masterID string
			next     dtos.NextSession
		)
		if err := rows.Scan(
			&masterID,
			&next.Id, &next.ScheduledAt, &next.Format, &next.Type,
			&next.System.Id, &next.System.Slug, &next.System.Name, &next.System.BadgeColor, &next.System.IsCurated,
		); err != nil {
			return nil, fmt.Errorf("scan next session: %w", err)
		}
		out[masterID] = &next
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate next sessions: %w", err)
	}
	return out, nil
}

func (r *sessionRepository) CountMasterStat(ctx context.Context, masterId string) (int, error){
	query, args, err := r.psql.
		Select("COUNT(*)").
		From("sessions s").
		Where(sq.Eq{"s.master_id": masterId}).
		Where(sq.NotEq{"s.status": []dtos.SessionStatus{dtos.Cancelled, dtos.Draft}}).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("build count master stat: %w", err)
	}

	var stat int
	if err := r.db.QueryRow(ctx, query, args...).Scan(&stat); err != nil {
		return 0, fmt.Errorf("query count master stat: %w", err)
	}

	return stat, nil;
}

func (r *sessionRepository) CountPlayersStats(ctx context.Context, sessionId string) (map[string]int, error){
	out := make(map[string]int, 1)

	query, args, err := r.psql.
		Select("sp2.player_id", "COUNT(*)").
		From("session_players sp1").
		Join("session_players sp2 ON sp2.player_id = sp1.player_id").
		Join("sessions s ON s.id = sp2.session_id").
		Where(sq.Eq{"sp1.session_id": sessionId}).
		Where(sq.Eq{"sp2.status": dtos.PlayerActive}).
		Where(sq.Eq{"s.status": dtos.Completed}).
		GroupBy("sp2.player_id").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count players stats: %w", err)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query count players stats: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id string
		var stat int
		if err := rows.Scan(&id, &stat); err != nil {
			return nil, fmt.Errorf("scan count players stats: %w", err)
		}
		out[id] = stat
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate count players stats: %w", err)
	}
	return out, nil
}