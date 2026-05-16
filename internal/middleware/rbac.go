package middleware

import (
	"github.com/gofiber/fiber/v3"
	"github.com/itzLilix/questboard-shared/dtos"
	"github.com/rs/zerolog"
)

type RBACMiddleware interface {
	Protected() fiber.Handler
	Optional() fiber.Handler
}

type rbacMiddleware struct {
	tokenParser TokenParser
	log zerolog.Logger
}

type TokenParser interface {
    ParseToken(tokenString string) (*dtos.TokenClaims, error)
}

const (
    LocalsUserID   = "userID"
    LocalsUserRole = "userRole"
)

func NewRBACMiddleware(tokenParser TokenParser, log zerolog.Logger) RBACMiddleware {
	return &rbacMiddleware{tokenParser: tokenParser, log: log}
}

func (r* rbacMiddleware) Protected() fiber.Handler {
    return func(c fiber.Ctx) error {
        token := c.Cookies("access_token")
        if token == "" {
            return c.SendStatus(fiber.StatusUnauthorized)
        }
        claims, err := r.tokenParser.ParseToken(token)
        if err != nil {
            r.log.Warn().Err(err).Str("path", c.Path()).Msg("unauthorized")
            return c.SendStatus(fiber.StatusUnauthorized)
        }
        c.Locals(LocalsUserID, claims.UserID)
        c.Locals(LocalsUserRole, claims.Role)
        return c.Next()
    }
}

func (r *rbacMiddleware) Optional() fiber.Handler {
	return func(c fiber.Ctx) error {
		token := c.Cookies("access_token")
		if token == "" {
			return c.Next()
		}
		claims, err := r.tokenParser.ParseToken(token)
		if err != nil {
			r.log.Warn().Err(err).Str("path", c.Path()).Msg("optional auth: invalid token")
			return c.Next()
		}
		c.Locals(LocalsUserID, claims.UserID)
		c.Locals(LocalsUserRole, claims.Role)
		return c.Next()
	}
}

// func (r* rbacMiddleware) OwnedOrAdmin(uc usecase.SessionUsecase) fiber.Handler {
// 	return func(c fiber.Ctx) error {
// 		userID := c.Locals(LocalsUserID)
// 		userRole := c.Locals(LocalsUserRole)
// 		if userRole == "admin" {
// 			return c.Next()
// 		}
// 		resourceOwnerID := uc.GetSessionOwnerID(c.Params("id"))
// 		if resourceOwnerID == "" || userID != resourceOwnerID {
// 			return c.SendStatus(fiber.StatusForbidden)
// 		}
// 		return c.Next()
// 	}
// }