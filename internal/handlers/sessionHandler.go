package handlers

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/itzLilix/questboard-session-service/internal/entities"
	"github.com/itzLilix/questboard-session-service/internal/middleware"
	"github.com/itzLilix/questboard-session-service/internal/usecase"
	"github.com/itzLilix/questboard-shared/dtos"
	"github.com/rs/zerolog"
)

type SessionHandler interface {
	RegisterRoutes(app fiber.Router)
}

type sessionHandler struct {
	rbac middleware.RBACMiddleware
	log  zerolog.Logger
	uc SessionUsecase
}

func NewSessionHandler(uc SessionUsecase, rbac middleware.RBACMiddleware, log zerolog.Logger) SessionHandler {
	return &sessionHandler{
		uc:   uc,
		rbac: rbac,
		log:  log,
	}
}

type SessionFilter struct {
	Scope    dtos.SessionScope `query:"scope"`
	MasterID string            `query:"masterId"`
	PlayerID string            `query:"playerId"`
	Status   []string          `query:"status"`

	Search       string   `query:"search"`
	Format       string   `query:"format"`
	Type         string   `query:"type"`
	City         string   `query:"city"`
	GSIncluded   []string `query:"systemIncluded"`
	GSExcluded   []string `query:"systemExcluded"`
	FreeSeats 	 int      `query:"freeSeats"`

	PriceMin *float64   `query:"priceMin"`
	PriceMax *float64   `query:"priceMax"`
	DateFrom *time.Time `query:"dateFrom"`
	DateTo   *time.Time `query:"dateTo"`

	Sort      string `query:"sort"`
	SortOrder string `query:"order"`
	Cursor    string `query:"cursor"`
	Limit     int    `query:"limit"`
}

type CreateSessionRequest struct {
	Title         string                    `json:"title"`
	Format        dtos.SessionFormat        `json:"format"`
	SystemID      string                    `json:"systemId"`
	MaxSeats      uint8                     `json:"maxSeats"`
	ScheduledAt   *time.Time                `json:"scheduledAt,omitempty"`
	Description   *string                   `json:"description,omitempty"`
	Location      *dtos.Location            `json:"location,omitempty"`
	DurationHours *float64                  `json:"durationHours,omitempty"`
	Price         *float64                  `json:"price,omitempty"`
	Availability  *dtos.SessionAvailability `json:"availability,omitempty"`
}

type EditSessionRequest struct {
	Title         *string                   `json:"title,omitempty"`
	Description   *string                   `json:"description,omitempty"`
	Location      *dtos.Location            `json:"location,omitempty"`
	PreviewURL    *string                   `json:"previewUrl,omitempty"`
	Format        *dtos.SessionFormat       `json:"format,omitempty"`
	Availability  *dtos.SessionAvailability `json:"availability,omitempty"`
	SystemID      *string                   `json:"systemId,omitempty"`
	ScheduledAt   *time.Time                `json:"scheduledAt,omitempty"`
	DurationHours *float64                  `json:"durationHours,omitempty"`
	MaxSeats      *int16                    `json:"maxSeats,omitempty"`
	Price         *float64                  `json:"price,omitempty"`
}

type ChangeStatusRequest struct {
	Status dtos.SessionStatus `json:"status"`
}

func (h *sessionHandler) RegisterRoutes(app fiber.Router) {
	s := app.Group("/sessions")
	
    s.Get("/",       h.rbac.Optional(), h.list)
    s.Get("/cards",  h.rbac.Optional(), h.getCardData)
    s.Get("/:id",    h.rbac.Optional(), h.getByID)
    s.Post("/",      h.rbac.Protected(), h.create)
    s.Patch("/:id",  h.rbac.Protected(), h.edit)
    s.Delete("/:id", h.rbac.Protected(), h.delete)
	s.Patch("/:id/status", h.rbac.Protected(), h.changeStatus)
	
	s.Get("/:id/players", h.rbac.Optional(), h.listPlayers)
	s.Post("/:id/join",   h.rbac.Protected(), h.join)
    s.Delete("/:id/leave", h.rbac.Protected(), h.leave)
	s.Delete("/:id/players/:playerId", h.rbac.Protected(), h.kickPlayer)

	s.Patch("/:id/players/me", h.rbac.Protected(), h.setMyCharacter)

	s.Post("/:id/applications",   h.rbac.Protected(), h.apply)
    s.Get("/:id/applications",    h.rbac.Protected(), h.listApplications)
    s.Patch("/:id/applications/:applicationId", h.rbac.Protected(), h.resolveApplication)

	s.Get("/:id/files", h.rbac.Protected(), h.listFiles)
	s.Post("/:id/files", h.rbac.Protected(), h.uploadFile)
	s.Delete("/:id/files/:fileId", h.rbac.Protected(), h.deleteFile)

	s.Get("/:id/comments", h.rbac.Optional(), h.listComments)
	s.Post("/:id/comments", h.rbac.Protected(), h.addComment)
	s.Delete("/:id/comments/:commentId", h.rbac.Protected(), h.deleteComment)
	s.Patch("/:id/comments/:commentId", h.rbac.Protected(), h.editComment)
}

// --- sessions ---------------------------------------------------------------

// @Summary      List sessions
// @Tags         sessions
// @Produce      json
// @Param        scope        query  string   false "Scope"                                        Enums(catalog, mastering, playing)
// @Param        masterId     query  string   false "Filter by master user ID"
// @Param        playerId     query  string   false "Filter by player user ID"
// @Param        status       query  []string false "Status filter; use 'public' for published/ongoing/completed"
// @Param        search       query  string   false "Full-text search"
// @Param        format       query  string   false "Format"                                       Enums(online, offline)
// @Param        type         query  string   false "Type"                                         Enums(oneshot, campaign)
// @Param        city         query  string   false "City (offline sessions only)"
// @Param        systemExcluded     query  []string   false "Exclude game system by ID"
// @Param        systemIncluded     query  []string   false "Include only game system by ID"
// @Param        freeSeats 	  query  integer  false "Minimum available seats"
// @Param        priceMin     query  number   false "Minimum price"
// @Param        priceMax     query  number   false "Maximum price"
// @Param        dateFrom     query  string   false "Start date filter (RFC3339)"
// @Param        dateTo       query  string   false "End date filter (RFC3339)"
// @Param        sort         query  string   false "Sort field"                                   Enums(scheduled_at, created_at, price, title, system)
// @Param        order        query  string   false "Sort order"                                   Enums(ASC, DESC)
// @Param        cursor       query  string   false "Pagination cursor"
// @Param        limit        query  integer  false "Page size (default 20)"
// @Success      200  {object} dtos.SessionListResponse
// @Failure      400  {object} ErrorResponse
// @Failure      500  {object} ErrorResponse
// @Router       /v1/sessions [get]
func (h *sessionHandler) list(c fiber.Ctx) error {
	var f SessionFilter
	if err := c.Bind().Query(&f); err != nil {
		h.log.Warn().Err(err).Msg("invalid list session query")
		return handleErr(c, usecase.ErrInvalidData)
	}

	resp, err := h.uc.List(c.Context(), usecase.ListSessionsInput{
		Scope:        f.Scope,
		MasterID:     f.MasterID,
		PlayerID:     f.PlayerID,
		Status:       f.Status,
		Search:       f.Search,
		Format:       f.Format,
		Type:         f.Type,
		City:         f.City,
		GSIncluded:   f.GSIncluded,
		GSExcluded:   f.GSExcluded,
		FreeSeats: 	  f.FreeSeats,
		PriceMin:     f.PriceMin,
		PriceMax:     f.PriceMax,
		DateFrom:     f.DateFrom,
		DateTo:       f.DateTo,
		Sort:         f.Sort,
		SortOrder:    f.SortOrder,
		Cursor:       f.Cursor,
		Limit:        f.Limit,
	}, entities.BuildViewerFromCtx(c))
	if err != nil {
		h.log.Error().Err(err).Msg("list sessions failed")
		return handleErr(c, err)
	}
	if resp.Items == nil {
		resp.Items = []dtos.Session{}
	}
	if resp.Users == nil {
		resp.Users = map[string]dtos.UserBrief{}
	}
	return c.Status(fiber.StatusOK).JSON(resp)
}

// @Summary      Get session card data for master profiles
// @Tags         sessions
// @Produce      json
// @Param        masterId  query    []string  false  "Master user IDs"
// @Success      200       {array}  dtos.SessionCardData
// @Failure      400       {object} ErrorResponse
// @Failure      500       {object} ErrorResponse
// @Router       /v1/sessions/cards [get]
func (h *sessionHandler) getCardData(c fiber.Ctx) error {
	type cardDataQuery struct {
		MasterIDs []string `query:"masterId"`
	}
	var q cardDataQuery
	if err := c.Bind().Query(&q); err != nil {
		h.log.Warn().Err(err).Msg("invalid card data query")
		return handleErr(c, usecase.ErrInvalidData)
	}

	cards, err := h.uc.GetCardData(c.Context(), q.MasterIDs)
	if err != nil {
		h.log.Error().Err(err).Msg("get session card data failed")
		return handleErr(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(cards)
}

// @Summary      Get session by ID
// @Tags         sessions
// @Produce      json
// @Param        id   path     string  true  "Session ID"
// @Success      200  {object} dtos.SessionResponse
// @Failure      403  {object} ErrorResponse
// @Failure      404  {object} ErrorResponse
// @Failure      500  {object} ErrorResponse
// @Router       /v1/sessions/{id} [get]
func (h *sessionHandler) getByID(c fiber.Ctx) error {
	id := c.Params("id")
	session, err := h.uc.GetByID(c.Context(), id, entities.BuildViewerFromCtx(c))
	if err != nil {
		h.log.Error().Err(err).Str("sessionId", id).Msg("get session by id failed")
		return handleErr(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(session)
}

// @Summary      Create a session
// @Tags         sessions
// @Accept       json
// @Produce      json
// @Param        body  body     CreateSessionRequest  true  "Session data"
// @Success      201   {object} dtos.Session
// @Failure      400   {object} ErrorResponse
// @Failure      401   {object} ErrorResponse
// @Failure      500   {object} ErrorResponse
// @Security     CookieAuth
// @Router       /v1/sessions [post]
func (h *sessionHandler) create(c fiber.Ctx) error {
	var req CreateSessionRequest
	if err := c.Bind().Body(&req); err != nil {
		h.log.Warn().Err(err).Msg("invalid create session body")
		return handleErr(c, usecase.ErrInvalidData)
	}

	maxSeats := int16(req.MaxSeats)
	in := usecase.SessionInput{
		Title:         &req.Title,
		Format:        &req.Format,
		SystemID:      &req.SystemID,
		ScheduledAt:   req.ScheduledAt,
		MaxSeats:      &maxSeats,
		Description:   req.Description,
		//PreviewURL:    &req.PreviewURL,
		Price:         req.Price,
		Availability:  req.Availability,
		DurationHours: req.DurationHours,
	}
	if req.Location != nil {
		in.Address = &req.Location.Address
		in.Lat = &req.Location.Lat
		in.Lng = &req.Location.Lng
	}

	session, err := h.uc.Create(c.Context(), in, entities.BuildViewerFromCtx(c))
	if err != nil {
		h.log.Error().Err(err).Msg("create session failed")
		return handleErr(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(session)
}

// @Summary      Edit a session
// @Tags         sessions
// @Accept       json
// @Produce      json
// @Param        id    path     string             true  "Session ID"
// @Param        body  body     EditSessionRequest true  "Fields to update"
// @Success      200   {object} dtos.Session
// @Failure      400   {object} ErrorResponse
// @Failure      401   {object} ErrorResponse
// @Failure      403   {object} ErrorResponse
// @Failure      404   {object} ErrorResponse
// @Failure      500   {object} ErrorResponse
// @Security     CookieAuth
// @Router       /v1/sessions/{id} [patch]
func (h *sessionHandler) edit(c fiber.Ctx) error {
	var req EditSessionRequest
	if err := c.Bind().Body(&req); err != nil {
		h.log.Warn().Err(err).Msg("invalid edit session body")
		return handleErr(c, usecase.ErrInvalidData)
	}

	in := usecase.SessionInput{
		Title:         req.Title,
		Description:   req.Description,
		//PreviewURL:    req.PreviewURL,
		Format:        req.Format,
		Availability:  req.Availability,
		SystemID:      req.SystemID,
		ScheduledAt:   req.ScheduledAt,
		DurationHours: req.DurationHours,
		MaxSeats:      req.MaxSeats,
		Price:         req.Price,
	}
	if req.Location != nil {
		addr := req.Location.Address
		lat := req.Location.Lat
		lng := req.Location.Lng
		in.Address = &addr
		in.Lat = &lat
		in.Lng = &lng
	}

	id := c.Params("id")
	session, err := h.uc.Edit(c.Context(), id, entities.BuildViewerFromCtx(c), in)
	if err != nil {
		h.log.Error().Err(err).Str("sessionId", id).Msg("edit session failed")
		return handleErr(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(session)
}

// @Summary      Delete a session
// @Tags         sessions
// @Param        id   path  string  true  "Session ID"
// @Success      204
// @Failure      401  {object} ErrorResponse
// @Failure      403  {object} ErrorResponse
// @Failure      404  {object} ErrorResponse
// @Failure      500  {object} ErrorResponse
// @Security     CookieAuth
// @Router       /v1/sessions/{id} [delete]
func (h *sessionHandler) delete(c fiber.Ctx) error {
	id := c.Params("id")
	if err := h.uc.Delete(c.Context(), id, entities.BuildViewerFromCtx(c)); err != nil {
		h.log.Error().Err(err).Str("sessionId", id).Msg("delete session failed")
		return handleErr(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// @Summary      Change session status
// @Tags         sessions
// @Accept       json
// @Produce      json
// @Param        id    path     string               true  "Session ID"
// @Param        body  body     ChangeStatusRequest  true  "New status"
// @Success      200   {object} dtos.Session
// @Failure      400   {object} ErrorResponse
// @Failure      401   {object} ErrorResponse
// @Failure      403   {object} ErrorResponse
// @Failure      404   {object} ErrorResponse
// @Failure      500   {object} ErrorResponse
// @Security     CookieAuth
// @Router       /v1/sessions/{id}/status [patch]
func (h *sessionHandler) changeStatus(c fiber.Ctx) error {
	var req ChangeStatusRequest
	if err := c.Bind().Body(&req); err != nil || req.Status == "" {
		return handleErr(c, usecase.ErrInvalidData)
	}

	id := c.Params("id")
	session, err := h.uc.ChangeStatus(c.Context(), id, entities.BuildViewerFromCtx(c), req.Status)
	if err != nil {
		h.log.Error().Err(err).Str("sessionId", id).Str("status", string(req.Status)).Msg("change session status failed")
		return handleErr(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(session)
}

// --- players ----------------------------------------------------------------

// @Summary      List session players
// @Tags         sessions
// @Produce      json
// @Param        id   path     string  true  "Session ID"
// @Success      200  {object} dtos.SessionPlayersResponse
// @Failure      403  {object} ErrorResponse
// @Failure      404  {object} ErrorResponse
// @Failure      500  {object} ErrorResponse
// @Router       /v1/sessions/{id}/players [get]
func (h *sessionHandler) listPlayers(c fiber.Ctx) error {
	id := c.Params("id")
	resp, err := h.uc.ListPlayers(c.Context(), id, entities.BuildViewerFromCtx(c))
	if err != nil {
		h.log.Error().Err(err).Str("sessionId", id).Msg("list session players failed")
		return handleErr(c, err)
	}
	if resp.Players == nil {
		resp.Players = []dtos.SessionPlayer{}
	}
	if resp.Users == nil {
		resp.Users = map[string]dtos.UserBrief{}
	}
	return c.Status(fiber.StatusOK).JSON(resp)
}
// @Summary      Join a session
// @Tags         sessions
// @Param        id   path  string  true  "Session ID"
// @Success      200
// @Failure      400  {object} ErrorResponse
// @Failure      401  {object} ErrorResponse
// @Failure      404  {object} ErrorResponse
// @Failure      409  {object} ErrorResponse
// @Security     CookieAuth
// @Router       /v1/sessions/{id}/join [post]
func (h *sessionHandler) join(c fiber.Ctx) error {
	sessionId := c.Params("id")
	err := h.uc.Join(c.Context(), sessionId, entities.BuildViewerFromCtx(c))
	if err != nil {
		h.log.Error().Err(err).Str("sessionId", sessionId).Msg("join session request failed")
		return handleErr(c, err)
	}
	return c.SendStatus(fiber.StatusNotImplemented)
}

// @Summary      Leave a session
// @Tags         sessions
// @Param        id   path  string  true  "Session ID"
// @Success      204
// @Failure      401  {object} ErrorResponse
// @Failure      404  {object} ErrorResponse
// @Security     CookieAuth
// @Router       /v1/sessions/{id}/leave [delete]
func (h *sessionHandler) leave(c fiber.Ctx) error { return c.SendStatus(fiber.StatusNotImplemented) }

// @Summary      Kick a player from a session
// @Tags         sessions
// @Param        id        path  string  true  "Session ID"
// @Param        playerId  path  string  true  "Player user ID"
// @Success      204
// @Failure      401  {object} ErrorResponse
// @Failure      403  {object} ErrorResponse
// @Failure      404  {object} ErrorResponse
// @Security     CookieAuth
// @Router       /v1/sessions/{id}/players/{playerId} [delete]
func (h *sessionHandler) kickPlayer(c fiber.Ctx) error { return c.SendStatus(fiber.StatusNotImplemented) }

// @Summary      Set character for the current player slot
// @Tags         sessions
// @Accept       json
// @Param        id   path  string  true  "Session ID"
// @Success      200
// @Failure      401  {object} ErrorResponse
// @Failure      404  {object} ErrorResponse
// @Security     CookieAuth
// @Router       /v1/sessions/{id}/players/me [patch]
func (h *sessionHandler) setMyCharacter(c fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNotImplemented)
}

// --- applications -----------------------------------------------------------

// @Summary      Apply to join a session
// @Tags         sessions
// @Accept       json
// @Produce      json
// @Param        id   path     string  true  "Session ID"
// @Success      201  {object} dtos.SessionApplication
// @Failure      400  {object} ErrorResponse
// @Failure      401  {object} ErrorResponse
// @Failure      404  {object} ErrorResponse
// @Failure      409  {object} ErrorResponse
// @Security     CookieAuth
// @Router       /v1/sessions/{id}/applications [post]
func (h *sessionHandler) apply(c fiber.Ctx) error { return c.SendStatus(fiber.StatusNotImplemented) }

// @Summary      List applications for a session
// @Tags         sessions
// @Produce      json
// @Param        id   path     string  true  "Session ID"
// @Success      200  {array}  dtos.SessionApplication
// @Failure      401  {object} ErrorResponse
// @Failure      403  {object} ErrorResponse
// @Failure      404  {object} ErrorResponse
// @Security     CookieAuth
// @Router       /v1/sessions/{id}/applications [get]
func (h *sessionHandler) listApplications(c fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNotImplemented)
}

// @Summary      Accept or reject an application
// @Tags         sessions
// @Accept       json
// @Param        id             path  string  true  "Session ID"
// @Param        applicationId  path  string  true  "Application ID"
// @Success      204
// @Failure      400  {object} ErrorResponse
// @Failure      401  {object} ErrorResponse
// @Failure      403  {object} ErrorResponse
// @Failure      404  {object} ErrorResponse
// @Security     CookieAuth
// @Router       /v1/sessions/{id}/applications/{applicationId} [patch]
func (h *sessionHandler) resolveApplication(c fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNotImplemented)
}

// --- files ------------------------------------------------------------------

// @Summary      List session files
// @Tags         sessions
// @Produce      json
// @Param        id   path     string  true  "Session ID"
// @Success      200  {array}  dtos.SessionFile
// @Failure      401  {object} ErrorResponse
// @Failure      403  {object} ErrorResponse
// @Failure      404  {object} ErrorResponse
// @Security     CookieAuth
// @Router       /v1/sessions/{id}/files [get]
func (h *sessionHandler) listFiles(c fiber.Ctx) error { return c.SendStatus(fiber.StatusNotImplemented) }

// @Summary      Upload a file to a session
// @Tags         sessions
// @Accept       multipart/form-data
// @Produce      json
// @Param        id    path      string  true  "Session ID"
// @Param        file  formData  file    true  "File to upload"
// @Success      201   {object}  dtos.SessionFile
// @Failure      400   {object}  ErrorResponse
// @Failure      401   {object}  ErrorResponse
// @Failure      403   {object}  ErrorResponse
// @Failure      413   {object}  ErrorResponse
// @Security     CookieAuth
// @Router       /v1/sessions/{id}/files [post]
func (h *sessionHandler) uploadFile(c fiber.Ctx) error { return c.SendStatus(fiber.StatusNotImplemented) }

// @Summary      Delete a session file
// @Tags         sessions
// @Param        id      path  string  true  "Session ID"
// @Param        fileId  path  string  true  "File ID"
// @Success      204
// @Failure      401  {object} ErrorResponse
// @Failure      403  {object} ErrorResponse
// @Failure      404  {object} ErrorResponse
// @Security     CookieAuth
// @Router       /v1/sessions/{id}/files/{fileId} [delete]
func (h *sessionHandler) deleteFile(c fiber.Ctx) error { return c.SendStatus(fiber.StatusNotImplemented) }

// --- comments ---------------------------------------------------------------

// @Summary      List session comments
// @Tags         sessions
// @Produce      json
// @Param        id   path     string  true  "Session ID"
// @Success      200  {array}  dtos.SessionCommentary
// @Failure      403  {object} ErrorResponse
// @Failure      404  {object} ErrorResponse
// @Router       /v1/sessions/{id}/comments [get]
func (h *sessionHandler) listComments(c fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNotImplemented)
}

// @Summary      Add a comment to a session
// @Tags         sessions
// @Accept       json
// @Produce      json
// @Param        id   path     string  true  "Session ID"
// @Success      201  {object} dtos.SessionCommentary
// @Failure      400  {object} ErrorResponse
// @Failure      401  {object} ErrorResponse
// @Failure      404  {object} ErrorResponse
// @Security     CookieAuth
// @Router       /v1/sessions/{id}/comments [post]
func (h *sessionHandler) addComment(c fiber.Ctx) error { return c.SendStatus(fiber.StatusNotImplemented) }

// @Summary      Edit a comment
// @Tags         sessions
// @Accept       json
// @Produce      json
// @Param        id         path     string  true  "Session ID"
// @Param        commentId  path     string  true  "Comment ID"
// @Success      200        {object} dtos.SessionCommentary
// @Failure      400        {object} ErrorResponse
// @Failure      401        {object} ErrorResponse
// @Failure      403        {object} ErrorResponse
// @Failure      404        {object} ErrorResponse
// @Security     CookieAuth
// @Router       /v1/sessions/{id}/comments/{commentId} [patch]
func (h *sessionHandler) editComment(c fiber.Ctx) error { return c.SendStatus(fiber.StatusNotImplemented) }

// @Summary      Delete a comment
// @Tags         sessions
// @Param        id         path  string  true  "Session ID"
// @Param        commentId  path  string  true  "Comment ID"
// @Success      204
// @Failure      401  {object} ErrorResponse
// @Failure      403  {object} ErrorResponse
// @Failure      404  {object} ErrorResponse
// @Security     CookieAuth
// @Router       /v1/sessions/{id}/comments/{commentId} [delete]
func (h *sessionHandler) deleteComment(c fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNotImplemented)
}
