package handlers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/itzLilix/questboard-session-service/internal/middleware"
	"github.com/rs/zerolog"
)

type CharacterHandler interface{ RegisterRoutes(app fiber.Router) }

type characterHandler struct {
	uc   CharacterUsecase
	rbac middleware.RBACMiddleware
	log  zerolog.Logger
}

func NewCharacterHandler(uc CharacterUsecase, rbac middleware.RBACMiddleware, log zerolog.Logger) CharacterHandler {
	return &characterHandler{uc: uc, rbac: rbac, log: log}
}

type CreateCharacterRequest struct {
	Name        string  `json:"name"`
	Class       *string `json:"class,omitempty"`
	Level       *int    `json:"level,omitempty"`
	AvatarURL   *string `json:"avatarUrl,omitempty"`
	Description *string `json:"description,omitempty"`
	SheetURL    *string `json:"sheetUrl,omitempty"`
}

type EditCharacterRequest struct {
	Name        *string `json:"name,omitempty"`
	Class       *string `json:"class,omitempty"`
	Level       *int    `json:"level,omitempty"`
	AvatarURL   *string `json:"avatarUrl,omitempty"`
	Description *string `json:"description,omitempty"`
	SheetURL    *string `json:"sheetUrl,omitempty"`
}

func (h *characterHandler) RegisterRoutes(app fiber.Router) {
	ch := app.Group("/characters")

	ch.Get("/me", h.rbac.Protected(), h.listMine)
	ch.Post("/", h.rbac.Protected(), h.create)
	ch.Get("/:id", h.rbac.Protected(), h.getByID)
	ch.Patch("/:id", h.rbac.Protected(), h.edit)
	ch.Delete("/:id", h.rbac.Protected(), h.delete)
}

// @Summary      List my characters
// @Tags         characters
// @Produce      json
// @Success      200  {array}  dtos.Character
// @Failure      401  {object} ErrorResponse
// @Failure      500  {object} ErrorResponse
// @Security     CookieAuth
// @Router       /v1/characters/me [get]
func (h *characterHandler) listMine(c fiber.Ctx) error { return c.SendStatus(fiber.StatusNotImplemented) }

// @Summary      Create a character
// @Tags         characters
// @Accept       json
// @Produce      json
// @Param        body  body     CreateCharacterRequest  true  "Character data"
// @Success      201   {object} dtos.Character
// @Failure      400   {object} ErrorResponse
// @Failure      401   {object} ErrorResponse
// @Failure      500   {object} ErrorResponse
// @Security     CookieAuth
// @Router       /v1/characters [post]
func (h *characterHandler) create(c fiber.Ctx) error { return c.SendStatus(fiber.StatusNotImplemented) }

// @Summary      Get character by ID
// @Tags         characters
// @Produce      json
// @Param        id   path     string  true  "Character ID"
// @Success      200  {object} dtos.Character
// @Failure      401  {object} ErrorResponse
// @Failure      403  {object} ErrorResponse
// @Failure      404  {object} ErrorResponse
// @Security     CookieAuth
// @Router       /v1/characters/{id} [get]
func (h *characterHandler) getByID(c fiber.Ctx) error { return c.SendStatus(fiber.StatusNotImplemented) }

// @Summary      Edit a character
// @Tags         characters
// @Accept       json
// @Produce      json
// @Param        id    path     string               true  "Character ID"
// @Param        body  body     EditCharacterRequest true  "Fields to update"
// @Success      200   {object} dtos.Character
// @Failure      400   {object} ErrorResponse
// @Failure      401   {object} ErrorResponse
// @Failure      403   {object} ErrorResponse
// @Failure      404   {object} ErrorResponse
// @Security     CookieAuth
// @Router       /v1/characters/{id} [patch]
func (h *characterHandler) edit(c fiber.Ctx) error { return c.SendStatus(fiber.StatusNotImplemented) }

// @Summary      Delete a character
// @Tags         characters
// @Param        id   path  string  true  "Character ID"
// @Success      204
// @Failure      401  {object} ErrorResponse
// @Failure      403  {object} ErrorResponse
// @Failure      404  {object} ErrorResponse
// @Security     CookieAuth
// @Router       /v1/characters/{id} [delete]
func (h *characterHandler) delete(c fiber.Ctx) error { return c.SendStatus(fiber.StatusNotImplemented) }
