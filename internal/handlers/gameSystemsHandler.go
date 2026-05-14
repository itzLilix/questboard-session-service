package handlers

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/itzLilix/questboard-session-service/internal/middleware"
	usecase "github.com/itzLilix/questboard-session-service/internal/usecases"
	"github.com/rs/zerolog"
)

type GameSystemsHandler interface {
	RegisterRoutes(app *fiber.App)
}

type gameSystemsHandler struct {
	uc  usecase.GameSystemsUsecase
	log zerolog.Logger
	rbac middleware.RBACMiddleware
}

func NewGameSystemsHandler(uc usecase.GameSystemsUsecase, log zerolog.Logger, rbac middleware.RBACMiddleware) GameSystemsHandler {
	return &gameSystemsHandler{uc: uc, log: log, rbac: rbac}
}

func (h *gameSystemsHandler) RegisterRoutes(app *fiber.App) {
	g := app.Group("/game-systems")
	g.Get("/curated", h.getCurated)
	g.Get("/", h.search)
	g.Post("/", h.rbac.Protected(), h.addUserSystem)
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

func (h *gameSystemsHandler) addUserSystem(c fiber.Ctx) error {
	type CreateGameSystemRequest struct {
    	Name string `json:"name"`
	}
	var system CreateGameSystemRequest
	if err := c.Bind().Body(&system); err != nil {
		h.log.Error().Err(err).Msg("invalid request body in addUserSystem")
		return c.SendStatus(fiber.StatusBadRequest)
    }

	added, err := h.uc.AddUserSystem(&usecase.CreateGameSystemInput{
		Name: system.Name,
	})
	if err != nil {
		if errors.Is(err, usecase.ErrSystemAlreadyExists) {
			return c.SendStatus(fiber.StatusConflict)
		}
		return c.SendStatus(fiber.StatusInternalServerError)
	}
	return c.Status(fiber.StatusOK).JSON(added)
}