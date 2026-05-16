package handlers

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/itzLilix/questboard-session-service/internal/middleware"
	usecase "github.com/itzLilix/questboard-session-service/internal/usecases"
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
    Search       string     `query:"search"`
    Format       string     `query:"format"`
    Type         string     `query:"type"`
    City         string     `query:"city"`
    SystemID     string     `query:"systemId"`
    HasFreeSeats bool       `query:"hasFreeSeats"`

    PriceMin     *float64   `query:"priceMin"`
    PriceMax     *float64   `query:"priceMax"`
    DateFrom     *time.Time `query:"dateFrom"`
    DateTo       *time.Time `query:"dateTo"`

    Sort         string     `query:"sort"`
    SortOrder    string     `query:"order"`
    Cursor       string     `query:"cursor"`
    Limit        int        `query:"limit"`
}

func (h *sessionHandler) RegisterRoutes(app *fiber.App) {
	s := app.Group("/sessions")
	
    s.Get("/",       h.rbac.Optional(), h.list)
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

func (h *sessionHandler) list(c fiber.Ctx) error         { return c.SendStatus(fiber.StatusNotImplemented) }
func (h *sessionHandler) getByID(c fiber.Ctx) error      { return c.SendStatus(fiber.StatusNotImplemented) }
func (h *sessionHandler) create(c fiber.Ctx) error       { return c.SendStatus(fiber.StatusNotImplemented) }
func (h *sessionHandler) edit(c fiber.Ctx) error         { return c.SendStatus(fiber.StatusNotImplemented) }
func (h *sessionHandler) delete(c fiber.Ctx) error       { return c.SendStatus(fiber.StatusNotImplemented) }
func (h *sessionHandler) changeStatus(c fiber.Ctx) error { return c.SendStatus(fiber.StatusNotImplemented) }

// --- players ----------------------------------------------------------------

func (h *sessionHandler) listPlayers(c fiber.Ctx) error    { return c.SendStatus(fiber.StatusNotImplemented) }
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
