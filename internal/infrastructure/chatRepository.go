package infrastructure

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/itzLilix/questboard-shared/dtos"
	"github.com/jackc/pgx/v5/pgxpool"
)

type chatRepository struct {
	db   *pgxpool.Pool
	psql sq.StatementBuilderType
}

func NewChatRepository(db *pgxpool.Pool, psql sq.StatementBuilderType) *chatRepository {
	return &chatRepository{
		db:   db,
		psql: psql,
	}
}

func (r *chatRepository) InitGeneralChat(ctx context.Context, roomID string, scope dtos.SessionType) error {
	var (
		scopeCol string
		scopeTable string
	)

	switch scope {
		case dtos.CampaignType:
			scopeCol = "campaign_id"
			scopeTable = "campaigns"
		case dtos.OneshotType:
			scopeCol = "session_id"
			scopeTable = "sessions"
		default:
			return fmt.Errorf("initialize general chat: unknown scope %v", scope)
	}

		insert := r.psql.
		Insert("chats").
		Columns(scopeCol, "kind").
		Values(roomID, dtos.ChatGeneralKind)

	if scope == dtos.OneshotType {
		insert = insert.Suffix(`
			ON CONFLICT (session_id)
			WHERE kind = 'general' AND session_id IS NOT NULL
			DO UPDATE
				SET archived_at = NULL
			RETURNING id
		`)
	} else {
		insert = insert.Suffix("RETURNING id")
	}

	query, args, err := insert.ToSql()
	if err != nil {
		return fmt.Errorf("build init general chat query: %w", err)
	}

	db := execFromCtx(ctx, r.db)

	var chatID string
	if err := db.QueryRow(ctx, query, args...).Scan(&chatID); err != nil {
		return fmt.Errorf("initialize general chat: %w", err)
	}

	_, err = db.Exec(ctx, fmt.Sprintf(`
		INSERT INTO chat_members (chat_id, user_id, role)
		SELECT $1, master_id, 'owner'
		FROM %s
		WHERE id = $2
		ON CONFLICT (chat_id, user_id) DO NOTHING
		`, scopeTable), chatID, roomID)

	if err != nil {
		return fmt.Errorf("ensure owner membership: %w", err)
	}

	err = r.InitChatPermissions(ctx, chatID)
	if err != nil {
		return fmt.Errorf("initialize chat permissions: %w", err)
	}

	return nil
}

func (r *chatRepository) RetireSessionChat(ctx context.Context, sessionID string) error {
	query, args, err := r.psql.
		Update("chats").
		Set("archived_at", sq.Expr("NOW()")).
		Where(sq.Eq{"session_id": sessionID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build archive general chat query: %w", err)
	}
	if _, err := execFromCtx(ctx, r.db).Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("archive general chat: %w", err)
	}
	return nil
}

func (r *chatRepository) ListSessionChats(ctx context.Context, sessionID, userID string) ([]dtos.ChatSummary, error) {
	query, args, err := r.psql.
		Select(
			"c.id", "c.session_id", "c.campaign_id", "c.kind",
			"c.title", "c.picture_url", 
			"c.last_message_at", "c.created_at",
			"c.dm_user_low", "c.dm_user_high",
			"lm.sender_id", "lm.body", "lm.has_attachment",
		).
		Prefix(`WITH scope AS (SELECT campaign_id FROM campaign_sessions WHERE session_id = ?)`, sessionID).
		From("chats c").
		LeftJoin("scope ON true").
		LeftJoin(`LATERAL (
			SELECT
				m.sender_id,
				m.body,
				EXISTS (SELECT 1 FROM message_attachments ma WHERE ma.message_id = m.id) AS has_attachment
			FROM messages m
			WHERE m.chat_id = c.id AND m.deleted_at IS NULL
			ORDER BY m.id DESC
			LIMIT 1
		) lm ON true`).
		Where(sq.Expr(`
			(
				c.session_id = ? AND c.archived_at IS NULL
				AND EXISTS (SELECT 1 FROM chat_members cm WHERE cm.chat_id = c.id AND cm.user_id = ?)
			)
			OR (
				scope.campaign_id IS NOT NULL
				AND (
					c.campaign_id = scope.campaign_id
					OR EXISTS (
						SELECT 1 FROM campaign_chat_attachments cca
						WHERE cca.campaign_id = scope.campaign_id AND cca.chat_id = c.id
					)
				)
				AND (
					EXISTS (SELECT 1 FROM chat_members cm WHERE cm.chat_id = c.id AND cm.user_id = ?)
					OR (
						c.kind = 'general'
						AND EXISTS (
							SELECT 1
							FROM campaign_sessions cs
							JOIN session_players sp ON sp.session_id = cs.session_id
							WHERE cs.campaign_id = scope.campaign_id
							  AND sp.player_id = ?
							  AND sp.status = 'active'
						)
					)
				)
			)`,
			sessionID, userID, userID, userID,
		)).
		OrderBy("COALESCE(c.last_message_at, c.created_at) DESC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list session chats query: %w", err)
	}

	rows, err := execFromCtx(ctx, r.db).Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list session chats: %w", err)
	}
	defer rows.Close()

	var out []dtos.ChatSummary
	for rows.Next() {
		var c dtos.ChatSummary
		var dmLow, dmHigh *string
		var lmSender, lmBody *string
		var lmHasAttachment *bool

		if err := rows.Scan(
			&c.ID, &c.SessionID, &c.CampaignID, &c.Kind,
			&c.Title, &c.PictureURL, 
			&c.LastMessageAt, &c.CreatedAt,
			&dmLow, &dmHigh,
			&lmSender, &lmBody, &lmHasAttachment,
		); err != nil {
			return nil, fmt.Errorf("scan chat summary: %w", err)
		}

		if c.Kind == dtos.ChatDirectKind {
			if dmLow != nil && *dmLow == userID {
				c.OtherUserID = dmHigh
			} else {
				c.OtherUserID = dmLow
			}
		}

		if lmSender != nil {
			c.LastMessage = &dtos.ChatLastMessage{
				SenderID:      *lmSender,
				Body:          lmBody,
				HasAttachment: lmHasAttachment != nil && *lmHasAttachment,
			}
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate chat summaries: %w", err)
	}
	return out, nil
}

func (r *chatRepository) InitChatPermissions(ctx context.Context, chatID string) error {
	query, args, err := r.psql.
		Insert("chat_role_permissions").
		Columns(
			"chat_id",
			"role",
		).
		Values(
			chatID,
			dtos.ChatMemberRole,
		).
		Values(
			chatID,
			dtos.ChatAdminRole,
		).
		Suffix("ON CONFLICT (chat_id, role) DO NOTHING").
		ToSql()

	if err != nil {
		return fmt.Errorf("build init chat permissions query: %w", err)
	}

	if _, err := execFromCtx(ctx, r.db).Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("init chat permissions: %w", err)
	}

	return nil
}

func (r *chatRepository) ListChatMembers(ctx context.Context, chatID string) ([]dtos.ChatMember, error) {
	query, args, err := r.psql.
		Select(
			"cm.user_id", "cm.role", "cm.joined_at", "cm.last_read_id",
		).
		From("chat_members cm").
		Where(sq.Eq{"cm.chat_id": chatID}).
		OrderBy("cm.joined_at ASC").
		ToSql()

	if err != nil {
		return nil, fmt.Errorf("build list chat members query: %w", err)
	}

	rows, err := execFromCtx(ctx, r.db).Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list chat members: %w", err)
	}

	defer rows.Close()

	var members []dtos.ChatMember
	for rows.Next() {
		var m dtos.ChatMember
		if err := rows.Scan(&m.UserID, &m.Role, &m.JoinedAt, &m.LastReadID); err != nil {
			return nil, fmt.Errorf("scan chat member: %w", err)
		}
		members = append(members, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate chat members: %w", err)
	}
	return members, nil
}

