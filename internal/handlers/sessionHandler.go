package handlers

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/itzLilix/questboard-session-service/internal/entities"
	"github.com/itzLilix/questboard-session-service/internal/middleware"
	usecase "github.com/itzLilix/questboard-session-service/internal/usecases"
	"github.com/itzLilix/questboard-shared/dtos"
	"github.com/rs/zerolog"
)

type SessionHandler interface {
	RegisterRoutes(app *fiber.App)
}

type sessionHandler struct {
	rbac middleware.RBACMiddleware
	log  zerolog.Logger
	uc usecase.SessionUsecase
}

func NewSessionHandler(uc usecase.SessionUsecase, rbac middleware.RBACMiddleware, log zerolog.Logger) SessionHandler {
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

	Search       string `query:"search"`
	Format       string `query:"format"`
	Type         string `query:"type"`
	City         string `query:"city"`
	SystemID     string `query:"systemId"`
	HasFreeSeats bool   `query:"hasFreeSeats"`

	PriceMin *float64   `query:"priceMin"`
	PriceMax *float64   `query:"priceMax"`
	DateFrom *time.Time `query:"dateFrom"`
	DateTo   *time.Time `query:"dateTo"`

	Sort      string `query:"sort"`
	SortOrder string `query:"order"`
	Cursor    string `query:"cursor"`
	Limit     int    `query:"limit"`
}

func (h *sessionHandler) RegisterRoutes(app *fiber.App) {
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

func (h *sessionHandler) list(c fiber.Ctx) error {
	var f SessionFilter
	if err := c.Bind().Query(&f); err != nil {
		h.log.Warn().Err(err).Msg("invalid list session query")
		return c.SendStatus(fiber.StatusBadRequest)
	}

	resp, err := h.uc.List(c.Context(), usecase.ListSessionsInput{
		Viewer:       entities.BuildViewerFromCtx(c),
		Scope:        f.Scope,
		MasterID:     f.MasterID,
		PlayerID:     f.PlayerID,
		Status:       f.Status,
		Search:       f.Search,
		Format:       f.Format,
		Type:         f.Type,
		City:         f.City,
		SystemID:     f.SystemID,
		HasFreeSeats: f.HasFreeSeats,
		PriceMin:     f.PriceMin,
		PriceMax:     f.PriceMax,
		DateFrom:     f.DateFrom,
		DateTo:       f.DateTo,
		Sort:         f.Sort,
		SortOrder:    f.SortOrder,
		Cursor:       f.Cursor,
		Limit:        f.Limit,
	})
	if err != nil {
		h.log.Error().Err(err).Msg("list sessions failed")
		return c.SendStatus(statusFor(err))
	}
	if resp.Items == nil {
		resp.Items = []dtos.Session{}
	}
	if resp.Users == nil {
		resp.Users = map[string]dtos.UserBrief{}
	}
	fmt.Println(len(resp.Items))
	return c.Status(fiber.StatusOK).JSON(resp)
}

func (h *sessionHandler) getCardData(c fiber.Ctx) error {
	type cardDataQuery struct {
		MasterIDs []string `query:"masterId"`
	}
	var q cardDataQuery
	if err := c.Bind().Query(&q); err != nil {
		h.log.Warn().Err(err).Msg("invalid card data query")
		return c.SendStatus(fiber.StatusBadRequest)
	}

	cards, err := h.uc.GetCardData(c.Context(), q.MasterIDs)
	if err != nil {
		h.log.Error().Err(err).Msg("get session card data failed")
		return c.SendStatus(statusFor(err))
	}
	return c.Status(fiber.StatusOK).JSON(cards)
}

func (h *sessionHandler) getByID(c fiber.Ctx) error {
	id := c.Params("id")
	session, err := h.uc.GetByID(c.Context(), id, entities.BuildViewerFromCtx(c))
	if err != nil {
		h.log.Error().Err(err).Str("sessionId", id).Msg("get session by id failed")
		return c.SendStatus(statusFor(err))
	}
	return c.Status(fiber.StatusOK).JSON(session)
}

func (h *sessionHandler) create(c fiber.Ctx) error {
	type CreateSessionRequest struct {
		// required
		Title    string             `json:"title"`
		Format   dtos.SessionFormat `json:"format"`
		SystemID string             `json:"systemId"`
		MaxSeats uint8              `json:"maxSeats"`

		// optional
		ScheduledAt   *time.Time                `json:"scheduledAt,omitempty"`
		Description   *string                   `json:"description,omitempty"`
		Location      *dtos.Location            `json:"location,omitempty"`
		DurationHours *float64                  `json:"durationHours,omitempty"`
		Price         *float64                  `json:"price,omitempty"`
		Availability  *dtos.SessionAvailability `json:"availability,omitempty"`
		//PreviewURL    *string                   `json:"previewUrl,omitempty"`
		MasterNotes   *string                   `json:"masterNotes,omitempty"`
	}

	var req CreateSessionRequest
	if err := c.Bind().Body(&req); err != nil {
		h.log.Warn().Err(err).Msg("invalid create session body")
		return c.SendStatus(fiber.StatusBadRequest)
	}

	maxSeats := int16(req.MaxSeats)
	in := usecase.SessionInput{
		Title:         &req.Title,
		Format:        &req.Format,
		SystemID:      &req.SystemID,
		ScheduledAt:   req.ScheduledAt,
		MaxSeats:      &maxSeats,
		Description:   req.Description,
		MasterNotes:   req.MasterNotes,
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
		return c.SendStatus(statusFor(err))
	}
	return c.Status(fiber.StatusCreated).JSON(session)
}

func (h *sessionHandler) edit(c fiber.Ctx) error {
	type EditSessionRequest struct {
		Title         *string                   `json:"title,omitempty"`
		Description   *string                   `json:"description,omitempty"`
		Location      *dtos.Location            `json:"location,omitempty"`
		MasterNotes   *string                   `json:"masterNotes,omitempty"`
		PreviewURL    *string                   `json:"previewUrl,omitempty"`
		Format        *dtos.SessionFormat       `json:"format,omitempty"`
		Availability  *dtos.SessionAvailability `json:"availability,omitempty"`
		SystemID      *string                   `json:"systemId,omitempty"`
		ScheduledAt   *time.Time                `json:"scheduledAt,omitempty"`
		DurationHours *float64                  `json:"durationHours,omitempty"`
		MaxSeats      *int16                    `json:"maxSeats,omitempty"`
		Price         *float64                  `json:"price,omitempty"`
	}

	var req EditSessionRequest
	if err := c.Bind().Body(&req); err != nil {
		h.log.Warn().Err(err).Msg("invalid edit session body")
		return c.SendStatus(fiber.StatusBadRequest)
	}

	in := usecase.SessionInput{
		Title:         req.Title,
		Description:   req.Description,
		MasterNotes:   req.MasterNotes,
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
		return c.SendStatus(statusFor(err))
	}
	return c.Status(fiber.StatusOK).JSON(session)
}

func (h *sessionHandler) delete(c fiber.Ctx) error {
	id := c.Params("id")
	if err := h.uc.Delete(c.Context(), id, entities.BuildViewerFromCtx(c)); err != nil {
		h.log.Error().Err(err).Str("sessionId", id).Msg("delete session failed")
		return c.SendStatus(statusFor(err))
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *sessionHandler) changeStatus(c fiber.Ctx) error {
	type ChangeStatusRequest struct {
		Status dtos.SessionStatus `json:"status"`
	}

	var req ChangeStatusRequest
	if err := c.Bind().Body(&req); err != nil || req.Status == "" {
		return c.SendStatus(fiber.StatusBadRequest)
	}

	id := c.Params("id")
	session, err := h.uc.ChangeStatus(c.Context(), id, entities.BuildViewerFromCtx(c), req.Status)
	if err != nil {
		h.log.Error().Err(err).Str("sessionId", id).Str("status", string(req.Status)).Msg("change session status failed")
		return c.SendStatus(statusFor(err))
	}
	return c.Status(fiber.StatusOK).JSON(session)
}

// --- players ----------------------------------------------------------------

func (h *sessionHandler) listPlayers(c fiber.Ctx) error {
	id := c.Params("id")
	resp, err := h.uc.ListPlayers(c.Context(), id, entities.BuildViewerFromCtx(c))
	if err != nil {
		h.log.Error().Err(err).Str("sessionId", id).Msg("list session players failed")
		return c.SendStatus(statusFor(err))
	}
	if resp.Players == nil {
		resp.Players = []dtos.SessionPlayer{}
	}
	if resp.Users == nil {
		resp.Users = map[string]dtos.UserBrief{}
	}
	return c.Status(fiber.StatusOK).JSON(resp)
}
func (h *sessionHandler) join(c fiber.Ctx) error           { return c.SendStatus(fiber.StatusNotImplemented) }
func (h *sessionHandler) leave(c fiber.Ctx) error          { return c.SendStatus(fiber.StatusNotImplemented) }
func (h *sessionHandler) kickPlayer(c fiber.Ctx) error     { return c.SendStatus(fiber.StatusNotImplemented) }
func (h *sessionHandler) setMyCharacter(c fiber.Ctx) error { return c.SendStatus(fiber.StatusNotImplemented) }

// --- applications -----------------------------------------------------------

func (h *sessionHandler) apply(c fiber.Ctx) error              { return c.SendStatus(fiber.StatusNotImplemented) }
func (h *sessionHandler) listApplications(c fiber.Ctx) error   { return c.SendStatus(fiber.StatusNotImplemented) }
func (h *sessionHandler) resolveApplication(c fiber.Ctx) error { return c.SendStatus(fiber.StatusNotImplemented) }

// --- files ------------------------------------------------------------------

func (h *sessionHandler) listFiles(c fiber.Ctx) error  { return c.SendStatus(fiber.StatusNotImplemented) }
func (h *sessionHandler) uploadFile(c fiber.Ctx) error { return c.SendStatus(fiber.StatusNotImplemented) }
func (h *sessionHandler) deleteFile(c fiber.Ctx) error { return c.SendStatus(fiber.StatusNotImplemented) }

// --- comments ---------------------------------------------------------------

func (h *sessionHandler) listComments(c fiber.Ctx) error  { return c.SendStatus(fiber.StatusNotImplemented) }
func (h *sessionHandler) addComment(c fiber.Ctx) error    { return c.SendStatus(fiber.StatusNotImplemented) }
func (h *sessionHandler) editComment(c fiber.Ctx) error   { return c.SendStatus(fiber.StatusNotImplemented) }
func (h *sessionHandler) deleteComment(c fiber.Ctx) error { return c.SendStatus(fiber.StatusNotImplemented) }
