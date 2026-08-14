package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/itzLilix/questboard-shared/dtos"
	"github.com/jackc/pgx/v5"
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

type SaveMessageParams struct {
	ChatID string
	SenderID string
	Body string
	Attachments []dtos.AttachmentInput
	MentionedUserIDs []string
	ReplyTo *dtos.ReplySnippet
}

type ListMessagesParams struct {
	ChatID string
	Before *string // exclusive cursor: return messages with id < Before
	Limit  int
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

func (r *chatRepository) GetSendPermission(ctx context.Context, chatID, userID string) (bool, error) {
	query, args, err := r.psql.
		Select("COALESCE(crp.can_send_messages, cm.role = 'owner')").
		From("chat_members cm").
		LeftJoin("chat_role_permissions crp ON crp.chat_id = cm.chat_id AND crp.role = cm.role").
		Where(sq.Eq{
			"cm.chat_id": chatID,
			"cm.user_id": userID,
		}).
		ToSql()
	if err != nil {
		return false, fmt.Errorf("build send-permission query: %w", err)
	}

	var canSend bool
	if err := r.db.QueryRow(ctx, query, args...).Scan(&canSend); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, ErrNotFound
		}
		return false, fmt.Errorf("query send permission: %w", err)
	}
	return canSend, nil
}

func (r *chatRepository) GetReplySnippet(ctx context.Context, chatID, messageID string) (*dtos.ReplySnippet, error) {
	query, args, err := r.psql.
		Select(
			"sender_id",
			"deleted_at",
			"CASE WHEN deleted_at IS NULL THEN LEFT(body, 200) END",
		).
		From("messages").
		Where(sq.Eq{"id": messageID, "chat_id": chatID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build reply-snippet query: %w", err)
	}

	var (
		senderID  string
		deletedAt *time.Time
		preview   *string
	)
	if err := r.db.QueryRow(ctx, query, args...).Scan(&senderID, &deletedAt, &preview); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("query reply snippet: %w", err)
	}

	snippet := &dtos.ReplySnippet{MessageID: messageID, SenderID: senderID, Deleted: deletedAt != nil}
	if preview != nil {
		snippet.ContentPreview = *preview
	}
	return snippet, nil
}

func (r *chatRepository) SaveMessage(ctx context.Context, in *SaveMessageParams) (*dtos.MessagePayload, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("generate message id: %w", err)
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) // no-op once committed

	var replyToID *string
	if in.ReplyTo != nil {
		replyToID = &in.ReplyTo.MessageID
	}

	insertMsg, args, err := r.psql.
		Insert("messages").
		Columns("id", "chat_id", "sender_id", "body", "reply_to_id").
		Values(id.String(), in.ChatID, in.SenderID, in.Body, replyToID).
		Suffix("RETURNING created_at").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build insert message query: %w", err)
	}

	var createdAt time.Time
	if err := tx.QueryRow(ctx, insertMsg, args...).Scan(&createdAt); err != nil {
		return nil, fmt.Errorf("insert message: %w", err)
	}

	// One INSERT per attachment rather than a multi-row VALUES + RETURNING
	// — Postgres doesn't guarantee RETURNING preserves input row order for
	// multi-row inserts, and we need each returned id matched to its
	// source attachment.
	attachments := make([]dtos.Attachment, 0, len(in.Attachments))
	for _, a := range in.Attachments {
		q, args, err := r.psql.
			Insert("message_attachments").
			Columns("message_id", "name", "url", "mime_type", "size_bytes").
			Values(id.String(), a.FileName, a.URL, a.MIMEType, a.SizeBytes).
			Suffix("RETURNING id").
			ToSql()
		if err != nil {
			return nil, fmt.Errorf("build insert attachment query: %w", err)
		}
		var attID string
		if err := tx.QueryRow(ctx, q, args...).Scan(&attID); err != nil {
			return nil, fmt.Errorf("insert attachment: %w", err)
		}
		attachments = append(attachments, dtos.Attachment{
			ID: attID, FileName: a.FileName, URL: a.URL, MIMEType: a.MIMEType, SizeBytes: a.SizeBytes,
		})
	}

	if len(in.MentionedUserIDs) > 0 {
		mentionBuilder := r.psql.Insert("message_mentions").Columns("message_id", "mentioned_user_id")
		for _, uid := range in.MentionedUserIDs {
			mentionBuilder = mentionBuilder.Values(id.String(), uid)
		}
		q, args, err := mentionBuilder.ToSql()
		if err != nil {
			return nil, fmt.Errorf("build insert mentions query: %w", err)
		}
		if _, err := tx.Exec(ctx, q, args...); err != nil {
			return nil, fmt.Errorf("insert mentions: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return &dtos.MessagePayload{
		ID:               id.String(),
		SenderID:         in.SenderID,
		Body:          in.Body,
		ReplyTo:          in.ReplyTo,
		Attachments:      attachments,
		MentionedUserIDs: in.MentionedUserIDs,
		CreatedAt:        createdAt,
	}, nil
}

func (r *chatRepository) ListMessages(ctx context.Context, p *ListMessagesParams) ([]dtos.MessagePayload, error) {
	qb := r.psql.
		Select(
			"m.id", "m.sender_id", "m.body", "m.created_at", "m.edited_at",
			"r.id", "r.sender_id", "r.deleted_at",
			"CASE WHEN r.deleted_at IS NULL THEN LEFT(r.body, 200) END",
		).
		From("messages m").
		LeftJoin("messages r ON r.id = m.reply_to_id").
		Where(sq.Eq{"m.chat_id": p.ChatID}).
		Where("m.deleted_at IS NULL").
		OrderBy("m.id DESC").
		Limit(uint64(p.Limit))

	if p.Before != nil {
		qb = qb.Where(sq.Lt{"m.id": *p.Before})
	}

	query, args, err := qb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list messages query: %w", err)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query messages: %w", err)
	}
	defer rows.Close()

	type scanned struct {
		id, senderID, body   string
		createdAt            time.Time
		editedAt             *time.Time
		replyID, replySender *string
		replyDeletedAt       *time.Time
		replyPreview         *string
	}

	var scannedRows []scanned
	for rows.Next() {
		var s scanned
		if err := rows.Scan(&s.id, &s.senderID, &s.body, &s.createdAt, &s.editedAt,
			&s.replyID, &s.replySender, &s.replyDeletedAt, &s.replyPreview); err != nil {
			return nil, fmt.Errorf("scan message row: %w", err)
		}
		scannedRows = append(scannedRows, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate message rows: %w", err)
	}
	if len(scannedRows) == 0 {
		return []dtos.MessagePayload{}, nil
	}

	ids := make([]string, len(scannedRows))
	for i, s := range scannedRows {
		ids[i] = s.id
	}

	attByMsg, err := r.batchAttachments(ctx, ids)
	if err != nil {
		return nil, err
	}
	mentionsByMsg, err := r.batchMentions(ctx, ids)
	if err != nil {
		return nil, err
	}

	out := make([]dtos.MessagePayload, len(scannedRows))
	for i, s := range scannedRows {
		payload := dtos.MessagePayload{
			ID:               s.id,
			SenderID:         s.senderID,
			Body:          s.body,
			Attachments:      attByMsg[s.id],
			MentionedUserIDs: mentionsByMsg[s.id],
			CreatedAt:        s.createdAt,
			EditedAt:         s.editedAt,
		}
		if s.replyID != nil {
			payload.ReplyTo = &dtos.ReplySnippet{
				MessageID: *s.replyID,
				SenderID:  *s.replySender,
				Deleted:   s.replyDeletedAt != nil,
			}
			if s.replyPreview != nil {
				payload.ReplyTo.ContentPreview = *s.replyPreview
			}
		}
		out[i] = payload
	}
	return out, nil
}

func (r *chatRepository) batchAttachments(ctx context.Context, messageIDs []string) (map[string][]dtos.Attachment, error) {
	query, args, err := r.psql.
		Select("message_id", "id", "name", "url", "mime_type", "size_bytes").
		From("message_attachments").
		Where(sq.Eq{"message_id": messageIDs}).
		OrderBy("id").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build attachments query: %w", err)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query attachments: %w", err)
	}
	defer rows.Close()

	out := make(map[string][]dtos.Attachment)
	for rows.Next() {
		var msgID string
		var a dtos.Attachment
		if err := rows.Scan(&msgID, &a.ID, &a.FileName, &a.URL, &a.MIMEType, &a.SizeBytes); err != nil {
			return nil, fmt.Errorf("scan attachment row: %w", err)
		}
		out[msgID] = append(out[msgID], a)
	}
	return out, rows.Err()
}

func (r *chatRepository) batchMentions(ctx context.Context, messageIDs []string) (map[string][]string, error) {
	query, args, err := r.psql.
		Select("message_id", "mentioned_user_id").
		From("message_mentions").
		Where(sq.Eq{"message_id": messageIDs}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build mentions query: %w", err)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query mentions: %w", err)
	}
	defer rows.Close()

	out := make(map[string][]string)
	for rows.Next() {
		var msgID, userID string
		if err := rows.Scan(&msgID, &userID); err != nil {
			return nil, fmt.Errorf("scan mention row: %w", err)
		}
		out[msgID] = append(out[msgID], userID)
	}
	return out, rows.Err()
}

func (r *chatRepository) AddMembers(ctx context.Context, chatID string, userIDs []string) error {
	ins := r.psql.
		Insert("chat_members").
		Columns("chat_id", "user_id", "role")
	for _, uid := range userIDs {
		ins = ins.Values(chatID, uid, dtos.ChatMemberRole)
	}
	insSQL, insArgs, err := ins.
		Suffix("ON CONFLICT (chat_id, user_id) DO NOTHING").
		ToSql()
	if err != nil {
		return fmt.Errorf("build members upsert: %w", err)
	}

	if _, err := execFromCtx(ctx, r.db).Exec(ctx, insSQL, insArgs...); err != nil {
		return fmt.Errorf("exec get general id: %w", err)
	}
	return nil
}

func (r *chatRepository) AddMembersToGeneral(ctx context.Context, sessionID string, userIDs []string) error { 
	var chatId string

	query, args, err := r.psql.
		Select("c.id").
		From("chats c").
		Where(sq.And{
			sq.Eq{"c.kind": dtos.ChatGeneralKind},
			sq.Or{
				sq.And{
					sq.Eq{"c.session_id": sessionID},
					sq.Expr("c.archived_at IS NULL"),
				},
				sq.Expr(`
					c.campaign_id = (
						SELECT campaign_id
						FROM campaign_sessions
						WHERE session_id = ?
					)
				`, sessionID),
			},
		}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build get general id query: %w", err)
	}
	
	if err := execFromCtx(ctx, r.db).QueryRow(ctx, query, args...).Scan(&chatId); err != nil {
		return fmt.Errorf("exec get general id: %w", err)
	}

	return r.AddMembers(ctx, chatId, userIDs)
}

func (r *chatRepository) GetPermissions(ctx context.Context, chatID, userID string) (*dtos.ChatPermissions, error) {
	// Owner rows never exist in chat_role_permissions by design (the
	// schema seeds admin+member only) — cm.role = 'owner' short-circuits
	// every column to true before COALESCE is even consulted. For
	// admin/member, COALESCE falls back to the same defaults the
	// columns themselves declare, so "no row yet" behaves identically
	// to "a row with default values" — consistent with the fix already
	// applied to GetSendPermission.
	query, args, err := r.psql.
		Select(
			"cm.role",
			"cm.role = 'owner' OR COALESCE(crp.can_send_messages, TRUE)",
			"cm.role = 'owner' OR COALESCE(crp.can_send_files, TRUE)",
			"cm.role = 'owner' OR COALESCE(crp.can_pin_messages, TRUE)",
			"cm.role = 'owner' OR COALESCE(crp.can_change_info, FALSE)",
			"cm.role = 'owner' OR COALESCE(crp.can_add_members, FALSE)",
			"cm.role = 'owner' OR COALESCE(crp.can_remove_members, FALSE)",
			"cm.role = 'owner' OR COALESCE(crp.can_delete_messages, FALSE)",
			"cm.role = 'owner' OR COALESCE(crp.can_manage_roles, FALSE)",
			"cm.role = 'owner' OR COALESCE(crp.can_manage_permissions, FALSE)",
		).
		From("chat_members cm").
		LeftJoin("chat_role_permissions crp ON crp.chat_id = cm.chat_id AND crp.role = cm.role").
		Where(sq.Eq{"cm.chat_id": chatID, "cm.user_id": userID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build permissions query: %w", err)
	}

	var p dtos.ChatPermissions
	if err := r.db.QueryRow(ctx, query, args...).Scan(
		&p.Role, &p.CanSendMessages, &p.CanSendFiles, &p.CanPinMessages,
		&p.CanChangeInfo, &p.CanAddMembers, &p.CanRemoveMembers,
		&p.CanDeleteMessages, &p.CanManageRoles, &p.CanManagePermissions,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("query permissions: %w", err)
	}
	return &p, nil
}