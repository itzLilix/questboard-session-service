package chatws

import (
	"context"

	"github.com/gofiber/contrib/v3/websocket"
	"github.com/gofiber/fiber/v3"
)

// AccessChecker abstracts your existing access logic — the
// IsCampaignMember/IsPlayer + chat_members-fallback rules already built
// for [[ttrpg-chat-system]] — into one yes/no gate for a given chatID.
type AccessChecker interface {
	CanAccessChat(ctx context.Context, chatID, userID string) (bool, error)
}

// RegisterRoutes wires the chat websocket endpoint onto app.
// Assumes an auth middleware upstream has already set c.Locals("userID").
// Pass perms as nil if you haven't decided on permission checks yet.
func RegisterRoutes(app *fiber.App, hub *Hub, store Persister, access AccessChecker, perms PermissionChecker) {
	app.Use("/ws/chats/:chatID", func(c fiber.Ctx) error {
		if !websocket.IsWebSocketUpgrade(c) {
			return fiber.ErrUpgradeRequired
		}

		userID, ok := c.Locals("userID").(string)
		if !ok || userID == "" {
			return fiber.ErrUnauthorized
		}

		chatID := c.Params("chatID")

		// fiber.Ctx satisfies context.Context in v3, so it can be passed
		// straight into your repo call here.
		allowed, err := access.CanAccessChat(c, chatID, userID)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "access check failed")
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
		userID, _ := conn.Locals("userID").(string)
		chatID, _ := conn.Locals("chatID").(string)
		ServeClient(hub, store, perms, conn, chatID, userID)
	}, cfg))
}
