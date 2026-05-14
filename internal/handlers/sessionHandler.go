package handlers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/itzLilix/questboard-session-service/internal/middleware"
	"github.com/rs/zerolog"
)

type SessionHandler interface {
	RegisterRoutes(app *fiber.App)
}

type sessionHandler struct {
	rbac middleware.RBACMiddleware
	log  zerolog.Logger
}

func NewHandler() *sessionHandler {
	return &sessionHandler{}
}

func (h *sessionHandler) RegisterRoutes(app *fiber.App) {
	games := app.Group("/sessions")
	games.Get("/", h.getGames)
	games.Get("/:id", h.getGameById)
	games.Post("/", h.rbac.Protected(), h.createGame)
	games.Patch("/:id", h.editGame)
	games.Delete("/:id", h.deleteGameById)
	games.Post("/:id/join", h.rbac.Protected(), h.addPlayerToGame)
}

func (h *sessionHandler) getGames(c fiber.Ctx) error {
	return nil;
}

func (h *sessionHandler) getGameById(c fiber.Ctx) error {
	return nil;
}

func (h *sessionHandler) createGame(c fiber.Ctx) error {
	return nil;
}

func (h *sessionHandler) editGame(c fiber.Ctx) error {
	return nil;
}

func (h *sessionHandler) deleteGameById(c fiber.Ctx) error {
	return nil;
}

func (h *sessionHandler) addPlayerToGame(c fiber.Ctx) error {
	return nil;
}