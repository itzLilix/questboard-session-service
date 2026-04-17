package middleware

import (
	"github.com/gofiber/fiber/v3"
	"github.com/itzLilix/questboard-shared/dtos"
	"github.com/rs/zerolog"
)

type RBACMiddleware interface {
	Protected() fiber.Handler
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