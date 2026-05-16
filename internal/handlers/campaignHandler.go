package handlers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/itzLilix/questboard-session-service/internal/middleware"
	usecase "github.com/itzLilix/questboard-session-service/internal/usecases"
	"github.com/rs/zerolog"
)

type CampaignHandler interface {
	RegisterRoutes(app *fiber.App)
}

type campaignHandler struct {
	rbac middleware.RBACMiddleware
	log  zerolog.Logger
	uc   usecase.CampaignUsecase
}

func NewCampaignHandler(uc usecase.CampaignUsecase, rbac middleware.RBACMiddleware, log zerolog.Logger) CampaignHandler {
	return &campaignHandler{
		uc:   uc,
		rbac: rbac,
		log:  log,
	}
}

func (h *campaignHandler) RegisterRoutes(app *fiber.App) {
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
	// c.Put("/:id/sessions/order", h.rbac.Protected(), h.reorderSessions)

	c.Get("/:id/players", h.rbac.Optional(), h.listPlayers)
}

// --- campaigns --------------------------------------------------------------

func (h *campaignHandler) list(c fiber.Ctx) error         { return c.SendStatus(fiber.StatusNotImplemented) }
func (h *campaignHandler) getByID(c fiber.Ctx) error      { return c.SendStatus(fiber.StatusNotImplemented) }
func (h *campaignHandler) create(c fiber.Ctx) error       { return c.SendStatus(fiber.StatusNotImplemented) }
func (h *campaignHandler) edit(c fiber.Ctx) error         { return c.SendStatus(fiber.StatusNotImplemented) }
func (h *campaignHandler) delete(c fiber.Ctx) error       { return c.SendStatus(fiber.StatusNotImplemented) }
func (h *campaignHandler) changeStatus(c fiber.Ctx) error { return c.SendStatus(fiber.StatusNotImplemented) }

// --- sessions in campaign ---------------------------------------------------

func (h *campaignHandler) listSessions(c fiber.Ctx) error  { return c.SendStatus(fiber.StatusNotImplemented) }
func (h *campaignHandler) tieSession(c fiber.Ctx) error    { return c.SendStatus(fiber.StatusNotImplemented) }
func (h *campaignHandler) untieSession(c fiber.Ctx) error  { return c.SendStatus(fiber.StatusNotImplemented) }
func (h *campaignHandler) editTie(c fiber.Ctx) error       { return c.SendStatus(fiber.StatusNotImplemented) }

// --- players ----------------------------------------------------------------

func (h *campaignHandler) listPlayers(c fiber.Ctx) error { return c.SendStatus(fiber.StatusNotImplemented) }
