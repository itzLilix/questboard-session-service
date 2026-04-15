package handlers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/itzLilix/questboard-session-service/internal/middleware"
	"github.com/rs/zerolog"
)

type Handler interface {
	RegisterRoutes(app *fiber.App)
}

type handler struct {
	rbac middleware.RBACMiddleware
	log  zerolog.Logger
}

func NewHandler() *handler {
	return &handler{}
}

func (h *handler) RegisterRoutes(app *fiber.App) {
	games := app.Group("/games")
	games.Get("/", h.getGames)
	games.Get("/:id", h.getGameById)
	games.Post("/", h.rbac.Protected(), h.createGame)
	games.Patch("/:id", h.editGame)
	games.Delete("/:id", h.deleteGameById)
	games.Post("/:id/join", h.addPlayerToGame)
}

func (h *handler) getGames(c fiber.Ctx) error {
	return nil;
}

func (h *handler) getGameById(c fiber.Ctx) error {
	return nil;
}

func (h *handler) createGame(c fiber.Ctx) error {
	return nil;
}

func (h *handler) editGame(c fiber.Ctx) error {
	return nil;
}

func (h *handler) deleteGameById(c fiber.Ctx) error {
	return nil;
}

func (h *handler) addPlayerToGame(c fiber.Ctx) error {
	return nil;
}