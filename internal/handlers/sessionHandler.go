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
	sessions := app.Group("/sessions")
	sessions.Get("/", h.getSessions)
	sessions.Get("/:id", h.getSessionById)
	sessions.Post("/", h.rbac.Protected(), h.createSession)
	sessions.Patch("/:id", h.editSession)
	sessions.Delete("/:id", h.deleteSessionById)
	sessions.Post("/:id/join", h.rbac.Protected(), h.addPlayerToSession)
}

func (h *sessionHandler) getSessions(c fiber.Ctx) error {
	return nil;
}

func (h *sessionHandler) getSessionById(c fiber.Ctx) error {
	return nil;
}

func (h *sessionHandler) createSession(c fiber.Ctx) error {
	return nil;
}

func (h *sessionHandler) editSession(c fiber.Ctx) error {
	return nil;
}

func (h *sessionHandler) deleteSessionById(c fiber.Ctx) error {
	return nil;
}

func (h *sessionHandler) addPlayerToSession(c fiber.Ctx) error {
	return nil;
}