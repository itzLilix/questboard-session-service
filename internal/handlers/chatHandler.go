package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/itzLilix/questboard-session-service/internal/entities"
	"github.com/itzLilix/questboard-session-service/internal/handlers/chatws"
	"github.com/itzLilix/questboard-session-service/internal/middleware"
	uc "github.com/itzLilix/questboard-session-service/internal/usecase"
	"github.com/itzLilix/questboard-shared/dtos"
	"github.com/rs/zerolog"
)

type ChatHandler interface {
	RegisterRoutes(app fiber.Router)
}

type chatHandler struct {
	uc      ChatUsecase
	rbac    middleware.RBACMiddleware
	log     zerolog.Logger
	hub     *chatws.Hub
}

func NewChatHandler(uc ChatUsecase, rbac middleware.RBACMiddleware, log zerolog.Logger, hub *chatws.Hub) ChatHandler {
	return &chatHandler{uc: uc, rbac: rbac, log: log, hub: hub}
}

func (h *chatHandler) RegisterRoutes(app fiber.Router) {
	chats := app.Group("/chats", h.rbac.Protected())

	chats.Get("/:chatId", h.getChat)

    chats.Get("/:chatId/messages", h.listMessages)
    chats.Get("/:chatId/messages/:messageId", h.getMessage)
    chats.Post("/:chatId/messages", h.sendMessage)

    chats.Patch("/:chatId/messages/:messageId", h.editMessage)
    chats.Delete("/:chatId/messages/:messageId", h.deleteMessage)

    chats.Get("/:chatId/members", h.listMembers)
    chats.Post("/:chatId/members", h.addMembers)
    chats.Delete("/:chatId/members/:userId", h.removeMember)
	chats.Patch("/:chatId/members/:userId/role", h.ChangeRole)
	
    chats.Get("/:chatId/pins", h.listPins)
    chats.Post("/:chatId/pins/:messageId", h.pin)
    chats.Delete("/:chatId/pins/:messageId", h.unpin)

	chats.Post("/:chatId/read", h.markRead)
	chats.Get("/:chatId/read", h.getReadState)
	
	chats.Get("/:chatId/permissions", h.getPermissions)
	chats.Put("/:chatId/permissions/:role", h.updateRolePermissions)
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
	Attachments   []dtos.AttachmentInput   `json:"attachments,omitempty"`
	MentionedUserIDs []string `json:"mentionedUserIds,omitempty"`
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

func (h *chatHandler) getChat(c fiber.Ctx) error { return c.SendStatus(fiber.StatusNotImplemented) }

func (h *chatHandler) listMessages(c fiber.Ctx) error {
	chatID := c.Params("chatId")

	var before *string
	if b := c.Query("before"); b != "" {
		before = &b
	}
	limit, _ := strconv.Atoi(c.Query("limit")) // 0/invalid -> usecase applies default

	page, err := h.uc.ListMessages(c.Context(), &uc.ListMessagesInput{
		ChatID: chatID,
		Before: before,
		Limit:  limit,
	}, entities.BuildViewerFromCtx(c))
	if err != nil {
		h.log.Error().Err(err).Str("chat_id", chatID).Msg("list messages failed")
		return handleErr(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(page)
}

func (h *chatHandler) getMessage(c fiber.Ctx) error {
	chatID := c.Params("chatId")
	messageId := c.Params("messageId")

	message, err := h.uc.GetMessageById(c.Context(), chatID, messageId, entities.BuildViewerFromCtx(c))
	if err != nil {
		h.log.Error().Err(err).Str("chat_id", chatID).Msg("get message failed")
		return handleErr(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(message)
}

func (h *chatHandler) sendMessage(c fiber.Ctx) error {
	var req SendMessageRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.ErrBadRequest
	}

	chatID := c.Params("chatId")
 
	payload, err := h.uc.SendMessage(c.Context(), uc.SendMessageInput{
		ChatID:        chatID,
		Body:          req.Body,
		ReplyToID:     req.ReplyToID,
		Attachments:   req.Attachments,
		MentionedUserIDs: req.MentionedUserIDs,
	}, entities.BuildViewerFromCtx(c))
	if err != nil {
		h.log.Error().Err(err).Str("chat_id", chatID).Msg("send message failed")
		return handleErr(c, err)
	}
 
	// Best-effort: the message is already durably saved, so a publish
	// failure shouldn't fail the request — connected clients just miss
	// the live push and catch up on their next fetch or reconnect.
	if err := h.hub.PublishEvent(chatID, chatws.EventMessage, payload); err != nil {
		h.log.Error().Err(err).Str("chat_id", chatID).Msg("failed to publish message event")
	}
 
	return c.Status(fiber.StatusCreated).JSON(payload)
}

func (h *chatHandler) editMessage(c fiber.Ctx) error { return c.SendStatus(fiber.StatusNotImplemented) }
func (h *chatHandler) deleteMessage(c fiber.Ctx) error { return c.SendStatus(fiber.StatusNotImplemented) }
func (h *chatHandler) listMembers(c fiber.Ctx) error { return c.SendStatus(fiber.StatusNotImplemented) }
func (h *chatHandler) addMembers(c fiber.Ctx) error { return c.SendStatus(fiber.StatusNotImplemented) }
func (h *chatHandler) removeMember(c fiber.Ctx) error { return c.SendStatus(fiber.StatusNotImplemented) }
func (h *chatHandler) ChangeRole(c fiber.Ctx) error { return c.SendStatus(fiber.StatusNotImplemented) }
func (h *chatHandler) listPins(c fiber.Ctx) error { 
	chatID := c.Params("chatId")
	
	pins, err := h.uc.ListPinned(c.Context(), chatID, entities.BuildViewerFromCtx(c))
	if err != nil {
		h.log.Error().Err(err).Str("chat_id", chatID).Msg("list pins failed")
		return handleErr(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(pins)
}
func (h *chatHandler) pin(c fiber.Ctx) error { return c.SendStatus(fiber.StatusNotImplemented) }
func (h *chatHandler) unpin(c fiber.Ctx) error { return c.SendStatus(fiber.StatusNotImplemented) }
func (h *chatHandler) markRead(c fiber.Ctx) error { return c.SendStatus(fiber.StatusNotImplemented) }
func (h *chatHandler) getReadState(c fiber.Ctx) error { return c.SendStatus(fiber.StatusNotImplemented) }
func (h *chatHandler) getPermissions(c fiber.Ctx) error {
	chatID := c.Params("chatId")
	perms, err := h.uc.GetPermissions(c.Context(), chatID, entities.BuildViewerFromCtx(c))
	if err != nil {
		h.log.Error().Err(err).Str("chat_id", chatID).Msg("get permissions failed")
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	return c.Status(fiber.StatusOK).JSON(perms)
}
func (h *chatHandler) updateRolePermissions(c fiber.Ctx) error { return c.SendStatus(fiber.StatusNotImplemented) }