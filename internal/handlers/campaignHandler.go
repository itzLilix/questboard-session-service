package handlers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/itzLilix/questboard-session-service/internal/entities"
	"github.com/itzLilix/questboard-session-service/internal/middleware"
	uc "github.com/itzLilix/questboard-session-service/internal/usecase"
	"github.com/itzLilix/questboard-shared/dtos"
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

type CampaignFilter struct {
	Search    string `query:"search"`
	MasterID  string `query:"masterId"`
	SystemID  string `query:"systemId"`
	Status    string `query:"status"`
	Sort      string `query:"sort"`
	SortOrder string `query:"order"`
	Cursor    string `query:"cursor"`
	Limit     int    `query:"limit"`
}

type CreateCampaignRequest struct {
	Title        string                    `json:"title"`
	Description  *string                   `json:"description,omitempty"`
	Availability dtos.SessionAvailability `json:"availability"`
	SystemID     string                    `json:"systemId"`
}

type EditCampaignRequest struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	Availability *dtos.SessionAvailability `json:"availability,omitempty"`
	SystemID    *string `json:"systemId,omitempty"`
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
	c.Put("/:id/sessions/order", h.rbac.Protected(), h.reorderSessions)

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
// @Param        sort      query   string  false "Sort field" Enums(created_at, title, status)
// @Param        order     query   string  false "Sort order" Enums(ASC, DESC)
// @Param        cursor    query   string  false "Pagination cursor"
// @Param        limit     query   integer false "Page size"
// @Success      200  {object}  object{items=[]dtos.Campaign,nextCursor=string}
// @Failure      400  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /v1/campaigns [get]
func (h *campaignHandler) list(c fiber.Ctx) error {
	var f CampaignFilter
	if err := c.Bind().Query(&f); err != nil {
		h.log.Warn().Err(err).Msg("invalid list campaigns query")
		return handleErr(c, uc.ErrInvalidData)
	}

	page, err := h.uc.List(c.Context(), uc.ListCampaignsInput{
		Search:    f.Search,
		MasterID:  f.MasterID,
		SystemID:  f.SystemID,
		Status:    f.Status,
		Sort:      f.Sort,
		SortOrder: f.SortOrder,
		Cursor:    f.Cursor,
		Limit:     f.Limit,
	}, entities.BuildViewerFromCtx(c))
	if err != nil {
		h.log.Error().Err(err).Msg("list campaigns failed")
		return handleErr(c, err)
	}
	if page.Items == nil {
		page.Items = []dtos.Campaign{}
	}
	return c.Status(fiber.StatusOK).JSON(page)
}

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
	id := c.Params("id")
	campaign, err := h.uc.GetByID(c.Context(), id, entities.BuildViewerFromCtx(c))
	if err != nil {
		h.log.Error().Err(err).Str("sessionId", id).Msg("get session by id failed")
		return handleErr(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(campaign)
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
func (h *campaignHandler) create(c fiber.Ctx) error { 
	var q CreateCampaignRequest
	if err := c.Bind().Body(&q); err != nil {
		h.log.Error().Err(err).Msg("invalid request body in CreateCampaignRequest")
		return handleErr(c, uc.ErrInvalidData)
	}

	campaign, err := h.uc.Create(c.Context(), uc.CampaignInput{
		Title:        &q.Title,
		Description:  q.Description,
		SystemID:     &q.SystemID,
		Availability: &q.Availability,
	}, entities.BuildViewerFromCtx(c))
	if err != nil {
		return handleErr(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(campaign)
}

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
func (h *campaignHandler) edit(c fiber.Ctx) error {
	var req EditCampaignRequest
	if err := c.Bind().Body(&req); err != nil {
		h.log.Warn().Err(err).Msg("invalid edit campaign body")
		return handleErr(c, uc.ErrInvalidData)
	}

	id := c.Params("id")
	campaign, err := h.uc.Edit(c.Context(), id, entities.BuildViewerFromCtx(c), uc.CampaignInput{
		Title:        req.Title,
		Description:  req.Description,
		Availability: req.Availability,
		SystemID:     req.SystemID,
	})
	if err != nil {
		h.log.Error().Err(err).Str("campaignId", id).Msg("edit campaign failed")
		return handleErr(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(campaign)
}

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
func (h *campaignHandler) delete(c fiber.Ctx) error {
	id := c.Params("id")
	if err := h.uc.Delete(c.Context(), id, entities.BuildViewerFromCtx(c)); err != nil {
		h.log.Error().Err(err).Str("campaignId", id).Msg("delete campaign failed")
		return handleErr(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

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
	var req ChangeCampaignStatusRequest
	if err := c.Bind().Body(&req); err != nil || req.Status == "" {
		return handleErr(c, uc.ErrInvalidData)
	}

	id := c.Params("id")
	if err := h.uc.ChangeStatus(c.Context(), id, entities.BuildViewerFromCtx(c), dtos.CampaignStatus(req.Status)); err != nil {
		h.log.Error().Err(err).Str("campaignId", id).Str("status", req.Status).Msg("change campaign status failed")
		return handleErr(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// --- sessions in campaign ---------------------------------------------------

// @Summary      List sessions tied to a campaign
// @Tags         campaigns
// @Produce      json
// @Param        id   path     string  true  "Campaign ID"
// @Success      200  {array}  dtos.Session
// @Failure      403  {object} ErrorResponse
// @Failure      404  {object} ErrorResponse
// @Router       /v1/campaigns/{id}/sessions [get]
func (h *campaignHandler) listSessions(c fiber.Ctx) error {
	id := c.Params("id")
	sessions, err := h.uc.ListSessions(c.Context(), id, entities.BuildViewerFromCtx(c))
	if err != nil {
		h.log.Error().Err(err).Str("campaignId", id).Msg("list campaign sessions failed")
		return handleErr(c, err)
	}
	if sessions == nil {
		sessions = []dtos.Session{}
	}
	return c.Status(fiber.StatusOK).JSON(sessions)
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
	campaignId := c.Params("id")

	var req TieSessionRequest
	c.Bind().Body(&req)

	err := h.uc.TieSession(c.Context(), campaignId, uc.TieSessionInput{
		SessionID: req.SessionID,
		OrderIndex: req.OrderIndex,
		BriefDescription: req.BriefDescription,
	}, entities.BuildViewerFromCtx(c))
	if err != nil {
		return handleErr(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
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
	campaignId := c.Params("id")
	sessionId := c.Params("sessionId")

	if err := h.uc.UntieSession(c.Context(), campaignId, sessionId, entities.BuildViewerFromCtx(c)); err != nil {
		return handleErr(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// @Summary      Update campaign-session tie metadata
// @Tags         campaigns
// @Accept       json
// @Param        id         path  string          true  "Campaign ID"
// @Param        sessionId  path  string          true  "Session ID"
// @Param        body       body  EditTieRequest  true  "Tie metadata"
// @Success      200  {object} dtos.CampaignSessionTie
// @Failure      400  {object} ErrorResponse
// @Failure      401  {object} ErrorResponse
// @Failure      403  {object} ErrorResponse
// @Failure      404  {object} ErrorResponse
// @Security     CookieAuth
// @Router       /v1/campaigns/{id}/sessions/{sessionId} [patch]
func (h *campaignHandler) editTie(c fiber.Ctx) error {
	campaignId := c.Params("id")
	sessionId := c.Params("sessionId")

	var req EditTieRequest
	if err := c.Bind().Body(&req); err != nil {
		h.log.Warn().Err(err).Msg("invalid edit tie body")
		return handleErr(c, uc.ErrInvalidData)
	}

	tie, err := h.uc.EditTie(c.Context(), campaignId, sessionId, entities.BuildViewerFromCtx(c), uc.EditTieInput{
		BriefDescription: req.BriefDescription,
	})
	if err != nil {
		return handleErr(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(tie)
}

// @Summary      Set campaign-session ties order
// @Tags         campaigns
// @Accept       json
// @Param        id         path  string          true  "Campaign ID"
// @Param        body       body  []string        true  "Sessions IDs in desired order"
// @Success      200  {array}  dtos.CampaignSessionTie
// @Failure      400  {object} ErrorResponse
// @Failure      401  {object} ErrorResponse
// @Failure      403  {object} ErrorResponse
// @Failure      404  {object} ErrorResponse
// @Security     CookieAuth
// @Router       /v1/campaigns/{id}/sessions/order [put]
func (h *campaignHandler) reorderSessions(c fiber.Ctx) error {
	campaignId := c.Params("id")

	var body []string
	if err := c.Bind().Body(&body); err != nil {
		h.log.Warn().Err(err).Msg("invalid reorder sessions body")
		return handleErr(c, uc.ErrInvalidData)
	}

	ties, err := h.uc.ReorderSessions(c.Context(), campaignId, body, entities.BuildViewerFromCtx(c))
	if err != nil {
		return handleErr(c, err)
	}
	if ties == nil {
		ties = []dtos.CampaignSessionTie{}
	}
	return c.Status(fiber.StatusOK).JSON(ties)
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
