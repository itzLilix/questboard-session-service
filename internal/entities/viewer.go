package entities

import (
	"github.com/gofiber/fiber/v3"
	"github.com/itzLilix/questboard-session-service/internal/middleware"
	"github.com/itzLilix/questboard-shared/dtos"
)

type Viewer struct {
	UserID string
	Role   dtos.Role
}

func (v *Viewer) IsAuthenticated() bool { return v != nil && v.UserID != "" }
func (v *Viewer) IsAdmin() bool         { return v != nil && v.Role == dtos.AdminRole }
func (v *Viewer) Is(userID string) bool { return v != nil && v.UserID == userID }

func (v *Viewer) CanActAs(ownerID string) bool {
	return v.IsAuthenticated() && (v.UserID == ownerID || v.IsAdmin())
}

func BuildViewerFromCtx(c fiber.Ctx) *Viewer {
	userID, ok := c.Locals(middleware.LocalsUserID).(string)
	if !ok || userID == "" {
		return nil
	}
	role, _ := c.Locals(middleware.LocalsUserRole).(dtos.Role)
	return &Viewer{UserID: userID, Role: role}
}
