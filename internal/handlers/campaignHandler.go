package handlers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/itzLilix/questboard-session-service/internal/middleware"
	"github.com/rs/zerolog"
)

type CampaignHandler interface {
	RegisterRoutes(app fiber.Router)
}

type campaignHandler struct {
	rbac middleware.RBACMiddleware
	log  zerolog.Logger
	uc   CampaignUsecase
}

func NewCampaignHandler(uc CampaignUsecase, rbac middleware.RBACMiddleware, log zerolog.Logger) CampaignHandler {
	return &campaignHandler{
		uc:   uc,
		rbac: rbac,
		log:  log,
	}
}

type CreateCampaignRequest struct {
	Title       string  `json:"title"`
	SystemID    string  `json:"systemId"`
	Description *string `json:"description,omitempty"`
}

type EditCampaignRequest struct {
	Title       *string `json:"title,omitempty"`
	SystemID    *string `json:"systemId,omitempty"`
	Description *string `json:"description,omitempty"`
}

type ChangeCampaignStatusRequest struct {
	Status string `json:"status"`
}

type TieSessionRequest struct {
	SessionID        string  `json:"sessionId"`
	OrderIndex       *int    `json:"orderIndex,omitempty"`
	BriefDescription *string `json:"briefDescription,omitempty"`
}

type EditTieRequest struct {
	OrderIndex       *int    `json:"orderIndex,omitempty"`
	BriefDescription *string `json:"briefDescription,omitempty"`
}

func (h *campaignHandler) RegisterRoutes(app fiber.Router) {
	c := app.Group("/campaigns")

	c.Get("/", h.rbac.Optional(), h.list)
	c.Get("/:id", h.rbac.Optional(), h.getByID)
	c.Post("/", h.rbac.Protected(), h.create)
	c.Patch("/:id", h.rbac.Protected(), h.edit)
	c.Delete("/:id", h.rbac.Protected(), h.delete)
	c.Patch("/:id/status", h.rbac.Protected(), h.changeStatus)

	c.Get("/:id/sessions", h.rbac.Optional(), h.listSessions)
	c.Post("/:id/sessions", h.rbac.Protected(), h.tieSession)
	c.Delete("/:id/sessions/:sessionId", h.rbac.Protected(), h.untieSession)
	c.Patch("/:id/sessions/:sessionId", h.rbac.Protected(), h.editTie)

	c.Get("/:id/players", h.rbac.Optional(), h.listPlayers)
}

// --- campaigns --------------------------------------------------------------

// @Summary      List campaigns
// @Tags         campaigns
// @Produce      json
// @Param        search    query   string  false "Search query"
// @Param        masterId  query   string  false "Filter by master user ID"
// @Param        systemId  query   string  false "Filter by game system ID"
// @Param        status    query   string  false "Status filter" Enums(active, completed, cancelled, paused)
// @Param        sort      query   string  false "Sort field"
// @Param        order     query   string  false "Sort order" Enums(ASC, DESC)
// @Param        cursor    query   string  false "Pagination cursor"
// @Param        limit     query   integer false "Page size"
// @Success      200  {object}  object{items=[]dtos.Campaign,nextCursor=string}
// @Failure      500  {object}  ErrorResponse
// @Router       /v1/campaigns [get]
func (h *campaignHandler) list(c fiber.Ctx) error { return c.SendStatus(fiber.StatusNotImplemented) }

// @Summary      Get campaign by ID
// @Tags         campaigns
// @Produce      json
// @Param        id   path     string  true  "Campaign ID"
// @Success      200  {object} dtos.Campaign
// @Failure      403  {object} ErrorResponse
// @Failure      404  {object} ErrorResponse
// @Failure      500  {object} ErrorResponse
// @Router       /v1/campaigns/{id} [get]
func (h *campaignHandler) getByID(c fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNotImplemented)
}

// @Summary      Create a campaign
// @Tags         campaigns
// @Accept       json
// @Produce      json
// @Param        body  body     CreateCampaignRequest  true  "Campaign data"
// @Success      201   {object} dtos.Campaign
// @Failure      400   {object} ErrorResponse
// @Failure      401   {object} ErrorResponse
// @Failure      500   {object} ErrorResponse
// @Security     CookieAuth
// @Router       /v1/campaigns [post]
func (h *campaignHandler) create(c fiber.Ctx) error { return c.SendStatus(fiber.StatusNotImplemented) }

// @Summary      Edit a campaign
// @Tags         campaigns
// @Accept       json
// @Produce      json
// @Param        id    path     string               true  "Campaign ID"
// @Param        body  body     EditCampaignRequest  true  "Fields to update"
// @Success      200   {object} dtos.Campaign
// @Failure      400   {object} ErrorResponse
// @Failure      401   {object} ErrorResponse
// @Failure      403   {object} ErrorResponse
// @Failure      404   {object} ErrorResponse
// @Failure      500   {object} ErrorResponse
// @Security     CookieAuth
// @Router       /v1/campaigns/{id} [patch]
func (h *campaignHandler) edit(c fiber.Ctx) error { return c.SendStatus(fiber.StatusNotImplemented) }

// @Summary      Delete a campaign
// @Tags         campaigns
// @Param        id   path  string  true  "Campaign ID"
// @Success      204
// @Failure      401  {object} ErrorResponse
// @Failure      403  {object} ErrorResponse
// @Failure      404  {object} ErrorResponse
// @Failure      500  {object} ErrorResponse
// @Security     CookieAuth
// @Router       /v1/campaigns/{id} [delete]
func (h *campaignHandler) delete(c fiber.Ctx) error { return c.SendStatus(fiber.StatusNotImplemented) }

// @Summary      Change campaign status
// @Tags         campaigns
// @Accept       json
// @Param        id    path  string                       true  "Campaign ID"
// @Param        body  body  ChangeCampaignStatusRequest  true  "New status"
// @Success      204
// @Failure      400  {object} ErrorResponse
// @Failure      401  {object} ErrorResponse
// @Failure      403  {object} ErrorResponse
// @Failure      404  {object} ErrorResponse
// @Security     CookieAuth
// @Router       /v1/campaigns/{id}/status [patch]
func (h *campaignHandler) changeStatus(c fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNotImplemented)
}

// --- sessions in campaign ---------------------------------------------------

// @Summary      List sessions tied to a campaign
// @Tags         campaigns
// @Produce      json
// @Param        id   path     string  true  "Campaign ID"
// @Success      200  {array}  dtos.CampaignSessionTie
// @Failure      403  {object} ErrorResponse
// @Failure      404  {object} ErrorResponse
// @Router       /v1/campaigns/{id}/sessions [get]
func (h *campaignHandler) listSessions(c fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNotImplemented)
}

// @Summary      Tie a session to a campaign
// @Tags         campaigns
// @Accept       json
// @Param        id    path  string             true  "Campaign ID"
// @Param        body  body  TieSessionRequest  true  "Session tie data"
// @Success      204
// @Failure      400  {object} ErrorResponse
// @Failure      401  {object} ErrorResponse
// @Failure      403  {object} ErrorResponse
// @Failure      404  {object} ErrorResponse
// @Security     CookieAuth
// @Router       /v1/campaigns/{id}/sessions [post]
func (h *campaignHandler) tieSession(c fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNotImplemented)
}

// @Summary      Remove a session from a campaign
// @Tags         campaigns
// @Param        id         path  string  true  "Campaign ID"
// @Param        sessionId  path  string  true  "Session ID"
// @Success      204
// @Failure      401  {object} ErrorResponse
// @Failure      403  {object} ErrorResponse
// @Failure      404  {object} ErrorResponse
// @Security     CookieAuth
// @Router       /v1/campaigns/{id}/sessions/{sessionId} [delete]
func (h *campaignHandler) untieSession(c fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNotImplemented)
}

// @Summary      Update campaign-session tie metadata
// @Tags         campaigns
// @Accept       json
// @Param        id         path  string          true  "Campaign ID"
// @Param        sessionId  path  string          true  "Session ID"
// @Param        body       body  EditTieRequest  true  "Tie metadata"
// @Success      204
// @Failure      400  {object} ErrorResponse
// @Failure      401  {object} ErrorResponse
// @Failure      403  {object} ErrorResponse
// @Failure      404  {object} ErrorResponse
// @Security     CookieAuth
// @Router       /v1/campaigns/{id}/sessions/{sessionId} [patch]
func (h *campaignHandler) editTie(c fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNotImplemented)
}

// --- players ----------------------------------------------------------------

// @Summary      List players in a campaign
// @Tags         campaigns
// @Produce      json
// @Param        id   path     string  true  "Campaign ID"
// @Success      200  {array}  dtos.SessionPlayer
// @Failure      403  {object} ErrorResponse
// @Failure      404  {object} ErrorResponse
// @Router       /v1/campaigns/{id}/players [get]
func (h *campaignHandler) listPlayers(c fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNotImplemented)
}
