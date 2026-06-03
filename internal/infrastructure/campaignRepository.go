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
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type campaignRepository struct {
	db   *pgxpool.Pool
	psql sq.StatementBuilderType
}

type ListCampaignsParams struct {
	Search     string
	MasterID   string
	SystemID   string
	Status     string
	PublicOnly bool
	Cursor     string
	Limit      int
	Sort       dtos.CampaignListSort
	SortOrder  dtos.SortOrder
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
	CampaignID        string
	SessionID         string
	OrderIndex        *int
	BriefDescription  *string
	CachedTitle       *string
	CachedScheduledAt *time.Time
}

type EditTieParams struct {
	CampaignID        string
	SessionID         string
	BriefDescription  *string
}

func NewCampaignRepository(db *pgxpool.Pool, psql sq.StatementBuilderType) *campaignRepository {
	return &campaignRepository{db: db, psql: psql}
}

// campaignStatusRankExpr ranks campaign statuses by lifecycle priority so the
// "status" sort reads Active → Paused → Completed → Cancelled (ascending). The
// integer ranks MUST stay in sync with campaignStatusRank below, which feeds the
// keyset cursor.
const campaignStatusRankExpr = "CASE c.status " +
	"WHEN 'active' THEN 0 " +
	"WHEN 'paused' THEN 1 " +
	"WHEN 'completed' THEN 2 " +
	"WHEN 'cancelled' THEN 3 " +
	"ELSE 4 END"

// campaignStatusRank mirrors campaignStatusRankExpr in Go so a cursor built from
// the last row carries the same ordinal the SQL CASE produced.
func campaignStatusRank(s dtos.CampaignStatus) int {
	switch s {
	case dtos.CampaignActive:
		return 0
	case dtos.CampaignPaused:
		return 1
	case dtos.CampaignCompleted:
		return 2
	case dtos.CampaignCancelled:
		return 3
	default:
		return 4
	}
}

var campaignSortColumns = map[dtos.CampaignListSort]string{
	dtos.SortCampaignCreatedAt: "c.created_at",
	dtos.SortCampaignTitle:     "c.title",
	dtos.SortCampaignStatus:    campaignStatusRankExpr,
}

func (r *campaignRepository) List(ctx context.Context, p ListCampaignsParams) ([]dtos.Campaign, string, error) {
	q := r.psql.
		Select(campaignColumns...).
		From("campaigns c").
		LeftJoin("game_systems gs ON gs.id = c.system_id")

	// --- visibility ---
	// Private campaigns never surface in a catalog browse. The usecase decides
	// when PublicOnly may be relaxed (admins, or a master listing their own).
	if p.PublicOnly {
		q = q.Where(sq.NotEq{"c.availability": dtos.Private})
	}

	// --- filters ---
	if p.Search != "" {
		q = q.Where("(c.title ILIKE ? OR ? <% c.title)", "%"+p.Search+"%", p.Search)
	}
	if p.MasterID != "" {
		q = q.Where(sq.Eq{"c.master_id": p.MasterID})
	}
	if p.SystemID != "" {
		q = q.Where(sq.Eq{"c.system_id": p.SystemID})
	}
	if p.Status != "" {
		q = q.Where(sq.Eq{"c.status": p.Status})
	}

	// --- sort resolution ---
	sortKey := p.Sort
	sortCol, ok := campaignSortColumns[sortKey]
	if !ok {
		sortKey = dtos.SortCampaignCreatedAt
		sortCol = campaignSortColumns[sortKey]
	}

	// --- cursor ---
	cur, err := cursor.DecodeCursor[campaignCursor](p.Cursor)
	if err != nil {
		return nil, "", err
	}
	q, err = applyCampaignCursor(q, cur, sortKey, p.SortOrder)
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
		fmt.Sprintf("c.id %s", p.SortOrder),
	)

	q = q.Limit(uint64(p.Limit + 1))

	query, args, err := q.ToSql()
	if err != nil {
		return nil, "", fmt.Errorf("build list campaigns query: %w", err)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, "", fmt.Errorf("query list campaigns: %w", err)
	}
	defer rows.Close()

	items := make([]dtos.Campaign, 0, p.Limit)
	for rows.Next() {
		var c dtos.Campaign
		if err := scanCampaign(rows, &c); err != nil {
			return nil, "", fmt.Errorf("scan list campaign row: %w", err)
		}
		items = append(items, c)
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("iterate list campaigns: %w", err)
	}

	// --- pagination ---
	var nextCursor string
	if len(items) > p.Limit {
		items = items[:p.Limit] // drop the sentinel +1 row
		nc, err := buildNextCampaignCursor(items[p.Limit-1], sortKey, p.SortOrder)
		if err != nil {
			return nil, "", fmt.Errorf("encode next campaign cursor: %w", err)
		}
		nextCursor = nc
	}
	return items, nextCursor, nil
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

	var camp dtos.Campaign
	if err := scanCampaign(r.db.QueryRow(ctx, selectSQL, selectArgs...), &camp); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("read campaign: %w", err)
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

// ListSessionTies returns the campaign's session ties (the denormalized cache:
// cached_title/cached_scheduled_at/order_index), with per-session visibility
// applied in SQL: private and draft sessions are hidden from viewers who are
// neither the master nor an active player of that specific session. Admins see
// everything. This is the cheap shape used by the campaign detail view
// (GetByID) — it avoids reading full session rows. For full session cards
// (address, seats), use ListSessions.
func (r *campaignRepository) ListSessionTies(ctx context.Context, campaignID string, v *entities.Viewer) ([]dtos.CampaignSessionTie, error) {
	q := r.psql.
		Select(campaignSessionTieColumns...).
		From("campaign_sessions cs").
		Join("sessions s ON s.id = cs.session_id").
		Where(sq.Eq{"cs.campaign_id": campaignID})

	if !v.IsAdmin() {
		visibility := sq.Or{
			sq.And{
				sq.Expr("s.status NOT IN (?, ?)", dtos.Draft, dtos.Cancelled),
				sq.NotEq{"s.availability": dtos.Private},
			},
		}
		if v.IsAuthenticated() {
			visibility = append(visibility, sq.Eq{"s.master_id": v.UserID})
			visibility = append(visibility, sq.Expr(
				"EXISTS (SELECT 1 FROM session_players sp WHERE sp.session_id = s.id AND sp.player_id = ? AND sp.status = ?)",
				v.UserID, dtos.PlayerActive,
			))
		}
		q = q.Where(visibility)
	}

	selectSQL, selectArgs, err := q.OrderBy("cs.order_index ASC").ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list campaign session ties: %w", err)
	}

	rows, err := r.db.Query(ctx, selectSQL, selectArgs...)
	if err != nil {
		return nil, fmt.Errorf("query campaign session ties: %w", err)
	}
	defer rows.Close()

	ties := make([]dtos.CampaignSessionTie, 0)
	for rows.Next() {
		var t dtos.CampaignSessionTie
		if err := scanCampaignSessionTie(rows, &t); err != nil {
			return nil, fmt.Errorf("scan campaign session tie: %w", err)
		}
		ties = append(ties, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate campaign session ties: %w", err)
	}
	return ties, nil
}

// ListSessions returns the campaign's sessions as full session cards (address,
// seats, embedded game system), ordered by the campaign's order_index. It joins
// the real session rows rather than the cache, so it carries the per-session
// fields the profile's campaign accordion needs on expand. Visibility matches
// ListSessionTies: private/draft sessions are hidden from viewers who are
// neither the master nor an active player of that specific session; admins see
// everything.
func (r *campaignRepository) ListSessions(ctx context.Context, campaignID string, v *entities.Viewer) ([]dtos.Session, error) {
	q := r.psql.
		Select(sessionColumns...).
		From("sessions s").
		Join("game_systems gs ON gs.id = s.system_id").
		Join("campaign_sessions cs ON cs.session_id = s.id").
		Where(sq.Eq{"cs.campaign_id": campaignID})

	if !v.IsAdmin() {
		visibility := sq.Or{
			sq.And{
				sq.Expr("s.status NOT IN (?, ?)", dtos.Draft, dtos.Cancelled),
				sq.NotEq{"s.availability": dtos.Private},
			},
		}
		if v.IsAuthenticated() {
			visibility = append(visibility, sq.Eq{"s.master_id": v.UserID})
			visibility = append(visibility, sq.Expr(
				"EXISTS (SELECT 1 FROM session_players sp WHERE sp.session_id = s.id AND sp.player_id = ? AND sp.status = ?)",
				v.UserID, dtos.PlayerActive,
			))
		}
		q = q.Where(visibility)
	}

	selectSQL, selectArgs, err := q.OrderBy("cs.order_index ASC").ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list campaign sessions: %w", err)
	}

	rows, err := r.db.Query(ctx, selectSQL, selectArgs...)
	if err != nil {
		return nil, fmt.Errorf("query campaign sessions: %w", err)
	}
	defer rows.Close()

	sessions := make([]dtos.Session, 0)
	for rows.Next() {
		var s dtos.Session
		if err := scanSession(rows, &s); err != nil {
			return nil, fmt.Errorf("scan campaign session: %w", err)
		}
		sessions = append(sessions, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate campaign sessions: %w", err)
	}
	return sessions, nil
}

// IsCampaignMember reports whether userID is an active player of any session
// belonging to the campaign. Used to gate access to private campaigns.
func (r *campaignRepository) IsCampaignMember(ctx context.Context, campaignID, userID string) (bool, error) {
	if userID == "" {
		return false, nil
	}
	selectSQL, selectArgs, err := r.psql.
		Select("1").
		From("campaign_sessions cs").
		Join("session_players sp ON sp.session_id = cs.session_id").
		Where(sq.Eq{
			"cs.campaign_id": campaignID,
			"sp.player_id":   userID,
			"sp.status":      dtos.PlayerActive,
		}).
		Limit(1).
		ToSql()
	if err != nil {
		return false, fmt.Errorf("build is campaign member: %w", err)
	}

	var one int
	if err := r.db.QueryRow(ctx, selectSQL, selectArgs...).Scan(&one); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("is campaign member: %w", err)
	}
	return true, nil
}

func (r *campaignRepository) TieSession(ctx context.Context, p *TieSessionParams) error {
	var orderVal any
	if p.OrderIndex != nil {
		orderVal = *p.OrderIndex
	} else {
		orderVal = sq.Expr(
			"COALESCE((SELECT MAX(order_index) + 1 FROM campaign_sessions WHERE campaign_id = ?), 1)",
			p.CampaignID,
		)
	}

	var briefArg, titleArg, schedArg any
	if p.BriefDescription != nil {
		briefArg = nullString(*p.BriefDescription)
	}
	if p.CachedTitle != nil {
		titleArg = nullString(*p.CachedTitle)
	}
	if p.CachedScheduledAt != nil {
		schedArg = *p.CachedScheduledAt
	}

	insertSQL, args, err := r.psql.
		Insert("campaign_sessions").
		Columns("campaign_id", "session_id", "order_index", "brief_description", "cached_title", "cached_scheduled_at").
		Values(p.CampaignID, p.SessionID, orderVal, briefArg, titleArg, schedArg).
		ToSql()
	if err != nil {
		return fmt.Errorf("build tie session: %w", err)
	}

	if _, err := r.db.Exec(ctx, insertSQL, args...); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrAlreadyExists
		}
		return fmt.Errorf("tie session: %w", err)
	}
	return nil
}

// EditTie updates a tie's brief_description and returns the refreshed tie. Every
// tie column lives on campaign_sessions, so an UPDATE ... RETURNING is enough —
// no JOIN to sessions is needed. A missing tie surfaces as ErrNotFound.
func (r *campaignRepository) EditTie(ctx context.Context, p *EditTieParams) (*dtos.CampaignSessionTie, error) {
	var briefArg any
	if p.BriefDescription != nil {
		briefArg = nullString(*p.BriefDescription)
	}

	updateSQL, args, err := r.psql.
		Update("campaign_sessions").
		Set("brief_description", briefArg).
		Where(sq.Eq{"campaign_id": p.CampaignID, "session_id": p.SessionID}).
		Suffix("RETURNING campaign_id, session_id, order_index, COALESCE(cached_title, ''), cached_scheduled_at, brief_description").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build edit tie: %w", err)
	}

	var tie dtos.CampaignSessionTie
	if err := scanCampaignSessionTie(r.db.QueryRow(ctx, updateSQL, args...), &tie); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("edit tie: %w", err)
	}
	return &tie, nil
}

// UntieSession removes a session from a campaign. The trg_session_type trigger
// flips sessions.type back to oneshot on DELETE, so we never touch type here.
// Deleting a tie that isn't there surfaces as ErrNotFound.
func (r *campaignRepository) UntieSession(ctx context.Context, campaignID, sessionID string) error {
	deleteSQL, args, err := r.psql.
		Delete("campaign_sessions").
		Where(sq.Eq{"campaign_id": campaignID, "session_id": sessionID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build untie session: %w", err)
	}

	cmdTag, err := r.db.Exec(ctx, deleteSQL, args...)
	if err != nil {
		return fmt.Errorf("untie session: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ReorderSessions rewrites the campaign's tie ordering to the given sequence,
// assigning order_index 1..N. The input must be a strict permutation of the
// campaign's current tie set (same members, no duplicates, no strangers) —
// anything else is ErrInvalidData. Because (campaign_id, order_index) is UNIQUE,
// the rewrite would trip the constraint mid-flight on any non-trivial reshuffle;
// we defer the check to COMMIT (constraint is DEFERRABLE INITIALLY IMMEDIATE) so
// uniqueness is validated once, against the final state.
func (r *campaignRepository) ReorderSessions(ctx context.Context, campaignID string, orderedSessionIDs []string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin reorder sessions: %w", err)
	}
	defer tx.Rollback(ctx)

	// Lock + read the campaign's current tie set so a concurrent tie/untie can't
	// race the permutation check.
	selectSQL, selectArgs, err := r.psql.
		Select("session_id").
		From("campaign_sessions").
		Where(sq.Eq{"campaign_id": campaignID}).
		Suffix("FOR UPDATE").
		ToSql()
	if err != nil {
		return fmt.Errorf("build lock campaign ties: %w", err)
	}

	rows, err := tx.Query(ctx, selectSQL, selectArgs...)
	if err != nil {
		return fmt.Errorf("lock campaign ties: %w", err)
	}
	current := make(map[string]struct{})
	for rows.Next() {
		var sid string
		if err := rows.Scan(&sid); err != nil {
			rows.Close()
			return fmt.Errorf("scan campaign tie id: %w", err)
		}
		current[sid] = struct{}{}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate campaign tie ids: %w", err)
	}

	// Strict full-permutation validation: same cardinality, every input ID belongs
	// to the campaign, and no duplicates.
	if len(orderedSessionIDs) != len(current) {
		return ErrInvalidData
	}
	seen := make(map[string]struct{}, len(orderedSessionIDs))
	for _, sid := range orderedSessionIDs {
		if _, ok := current[sid]; !ok {
			return ErrInvalidData
		}
		if _, dup := seen[sid]; dup {
			return ErrInvalidData
		}
		seen[sid] = struct{}{}
	}

	// Opt this transaction into deferred uniqueness so transient order_index
	// collisions during the rewrite don't fail row-by-row.
	if _, err := tx.Exec(ctx, "SET CONSTRAINTS ALL DEFERRED"); err != nil {
		return fmt.Errorf("defer constraints: %w", err)
	}

	for i, sid := range orderedSessionIDs {
		if _, err := tx.Exec(ctx,
			"UPDATE campaign_sessions SET order_index = $1 WHERE campaign_id = $2 AND session_id = $3",
			i+1, campaignID, sid,
		); err != nil {
			return fmt.Errorf("reorder session %s: %w", sid, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrAlreadyExists
		}
		return fmt.Errorf("commit reorder sessions: %w", err)
	}
	return nil
}

func (r *campaignRepository) ListPlayers(ctx context.Context, campaignID string) ([]dtos.SessionPlayer, error) {
	return nil, nil
}
