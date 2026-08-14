package chatws

import (
	"strings"

	"github.com/gofiber/contrib/v3/websocket"
	"github.com/gofiber/fiber/v3"
	"github.com/itzLilix/questboard-session-service/internal/entities"
	"github.com/itzLilix/questboard-session-service/internal/middleware"
)

// RegisterRoutes wires the chat websocket endpoint onto app.
// Assumes an auth middleware upstream has already set c.Locals("userID").
func RegisterRoutes(app fiber.Router, hub *Hub, uc Usecase, rbac middleware.RBACMiddleware) {
	app.Use("/ws/chats/:chatID", rbac.Protected(), func(c fiber.Ctx) error {
		if !websocket.IsWebSocketUpgrade(c) {
			return fiber.ErrUpgradeRequired
		}

		userID, ok := c.Locals("userID").(string)
		if !ok || userID == "" {
			return fiber.ErrUnauthorized
		}

		// c.Params returns a zero-copy view into fasthttp's per-request buffer —
		// valid only for this handler's lifetime. ServeClient stores this on
		// *Client for the whole connection's duration, so it must be a real copy
		// before it escapes this handler, or fasthttp's buffer-pool reuse will
		// silently corrupt it out from under a long-lived connection.
		chatID := strings.Clone(c.Params("chatID"))

		viewer := entities.BuildViewerFromCtx(c)
		c.Locals("viewer", viewer)

		allowed, err := uc.CanAccessChat(c, chatID, viewer)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "membership check failed")
		}
		if !allowed {
			return fiber.ErrForbidden
		}

		c.Locals("chatID", chatID)
		return c.Next()
	})

	cfg := websocket.Config{
		// The recover middleware doesn't work with this websocket
		// middleware — this is the documented way to catch panics inside
		// the connection handler instead.
		RecoverHandler: func(conn *websocket.Conn) {
			if err := recover(); err != nil {
				_ = conn.WriteJSON(fiber.Map{"type": "error", "error": "internal error"})
			}
			conn.Close()
		},
	}

	app.Get("/ws/chats/:chatID", websocket.New(func(conn *websocket.Conn) {
		chatID, _ := conn.Locals("chatID").(string)
		viewer, _ := conn.Locals("viewer").(*entities.Viewer)
		ServeClient(hub, uc, conn, chatID, viewer)
	}, cfg))
}
