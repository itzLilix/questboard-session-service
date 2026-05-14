package handlers

import (
	"github.com/gofiber/fiber/v3"
	usecase "github.com/itzLilix/questboard-session-service/internal/usecases"
	"github.com/rs/zerolog"
)

type GameSystemsHandler interface {
	RegisterRoutes(app *fiber.App)
}

type gameSystemsHandler struct {
	uc  usecase.GameSystemsUsecase
	log zerolog.Logger
}

func NewGameSystemsHandler(uc usecase.GameSystemsUsecase, log zerolog.Logger) GameSystemsHandler {
	return &gameSystemsHandler{uc: uc, log: log}
}

func (h *gameSystemsHandler) RegisterRoutes(app *fiber.App) {
	g := app.Group("/game-systems")
	g.Get("/curated", h.getCurated)
	g.Get("/search", h.search)
}

func (h *gameSystemsHandler) getCurated(c fiber.Ctx) error {
	systems, err := h.uc.GetCurated()
	if err != nil {
		h.log.Error().Err(err).Msg("failed to get curated systems")
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	return c.Status(fiber.StatusOK).JSON(systems)
}

func (h *gameSystemsHandler) search(c fiber.Ctx) error {
	q := c.Query("q")
	if q == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "query parameter is required"})
	}

	systems, err := h.uc.Search(q)
	if err != nil {
		h.log.Error().Err(err).Msg("failed to search systems")
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	return c.Status(fiber.StatusOK).JSON(systems)
}
