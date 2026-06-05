package handlers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/itzLilix/questboard-session-service/internal/middleware"
	"github.com/itzLilix/questboard-shared/dtos"
	"github.com/rs/zerolog"
)

type ChatHandler interface {
	RegisterRoutes(app fiber.Router)
}

type chatHandler struct {
	uc      ChatUsecase
	session SessionUsecase
	rbac    middleware.RBACMiddleware
	log     zerolog.Logger
	hub    *Hub
}

func NewChatHandler(uc ChatUsecase, session SessionUsecase, rbac middleware.RBACMiddleware, log zerolog.Logger) ChatHandler {
	return &chatHandler{uc: uc, session: session, rbac: rbac, log: log, hub: NewHub(log)}
}

func (h *chatHandler) RegisterRoutes(app fiber.Router) {
	_ = app.Group("/chats", h.rbac.Protected())
}

// --- request types ----------------------------------------------------------

type CreateChatRequest struct {
	Kind      dtos.ChatKind   `json:"kind"`
	Title     *string  `json:"title,omitempty"`// groups only
	MemberIDs []string `json:"memberIds,omitempty"`
}

type UpdateChatRequest struct {
	Title      *string `json:"title,omitempty"`
	PictureURL *string `json:"pictureUrl,omitempty"`
}

type SendMessageRequest struct {
	Body          string   `json:"body"`
	ReplyToID     *string  `json:"replyToId,omitempty"`
	AttachmentIDs []string `json:"attachmentIds,omitempty"`
}

type EditMessageRequest struct {
	Body string `json:"body"`
}

type AddMembersRequest struct {
	UserIDs []string `json:"userIds"`
}

type ChangeRoleRequest struct {
	Role dtos.ChatRole `json:"role"`
}

type MarkReadRequest struct {
	LastReadID string `json:"lastReadId"`
}

type UpdatePermissionsRequest struct {
	CanSendMessages      *bool `json:"canSendMessages,omitempty"`
	CanSendFiles         *bool `json:"canSendFiles,omitempty"`
	CanPinMessages       *bool `json:"canPinMessages,omitempty"`
	CanChangeInfo        *bool `json:"canChangeInfo,omitempty"`
	CanAddMembers        *bool `json:"canAddMembers,omitempty"`
	CanRemoveMembers     *bool `json:"canRemoveMembers,omitempty"`
	CanDeleteMessages    *bool `json:"canDeleteMessages,omitempty"`
	CanManageRoles       *bool `json:"canManageRoles,omitempty"`
	CanManagePermissions *bool `json:"canManagePermissions,omitempty"`
}