package handlers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/itzLilix/questboard-session-service/internal/middleware"
	usecase "github.com/itzLilix/questboard-session-service/internal/usecases"
	"github.com/rs/zerolog"
)

type CharacterHandler interface{ RegisterRoutes(app *fiber.App) }

type characterHandler struct {
	uc   usecase.CharacterUsecase
	rbac middleware.RBACMiddleware
	log  zerolog.Logger
}

func NewCharacterHandler(uc usecase.CharacterUsecase, rbac middleware.RBACMiddleware, log zerolog.Logger) CharacterHandler {
	return &characterHandler{uc: uc, rbac: rbac, log: log}
}

func (h *characterHandler) RegisterRoutes(app *fiber.App) {
	ch := app.Group("/characters")

	ch.Get("/me", h.rbac.Protected(), h.listMine)
	ch.Post("/", h.rbac.Protected(), h.create)
	ch.Get("/:id", h.rbac.Protected(), h.getByID)
	ch.Patch("/:id", h.rbac.Protected(), h.edit)
	ch.Delete("/:id", h.rbac.Protected(), h.delete)
}

func (h *characterHandler) listMine(c fiber.Ctx) error { return c.SendStatus(fiber.StatusNotImplemented) }
func (h *characterHandler) create(c fiber.Ctx) error   { return c.SendStatus(fiber.StatusNotImplemented) }
func (h *characterHandler) getByID(c fiber.Ctx) error  { return c.SendStatus(fiber.StatusNotImplemented) }
func (h *characterHandler) edit(c fiber.Ctx) error     { return c.SendStatus(fiber.StatusNotImplemented) }
func (h *characterHandler) delete(c fiber.Ctx) error   { return c.SendStatus(fiber.StatusNotImplemented) }
