package handlers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/itzLilix/questboard-session-service/internal/middleware"
	usecase "github.com/itzLilix/questboard-session-service/internal/usecases"
	"github.com/rs/zerolog"
)

type GameSystemsHandler interface {
	RegisterRoutes(app fiber.Router)
}

type gameSystemsHandler struct {
	uc  usecase.GameSystemsUsecase
	log zerolog.Logger
	rbac middleware.RBACMiddleware
}

func NewGameSystemsHandler(uc usecase.GameSystemsUsecase, log zerolog.Logger, rbac middleware.RBACMiddleware) GameSystemsHandler {
	return &gameSystemsHandler{uc: uc, log: log, rbac: rbac}
}

type CreateGameSystemRequest struct {
	Name string `json:"name"`
}

func (h *gameSystemsHandler) RegisterRoutes(app fiber.Router) {
	g := app.Group("/game-systems")
	g.Get("/all", h.getAll)
	g.Get("/curated", h.getCurated)
	g.Get("/", h.search)
	g.Post("/", h.rbac.Protected(), h.addUserSystem)
}

// @Summary      Get all game systems
// @Tags         game-systems
// @Produce      json
// @Success      200  {array}  dtos.GameSystem
// @Failure      500  {object} ErrorResponse
// @Router       /v1/game-systems/all [get]
func (h *gameSystemsHandler) getAll(c fiber.Ctx) error {
	systems, err := h.uc.GetAll(c.Context())
	if err != nil {
		h.log.Error().Err(err).Msg("failed to get all systems")
		return handleErr(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(systems)
}

// @Summary      Get curated game systems
// @Tags         game-systems
// @Produce      json
// @Success      200  {array}  dtos.GameSystem
// @Failure      500  {object} ErrorResponse
// @Router       /v1/game-systems/curated [get]
func (h *gameSystemsHandler) getCurated(c fiber.Ctx) error {
	systems, err := h.uc.GetCurated(c.Context())
	if err != nil {
		h.log.Error().Err(err).Msg("failed to get curated systems")
		return handleErr(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(systems)
}

// @Summary      Search game systems
// @Tags         game-systems
// @Produce      json
// @Param        q   query    string  true  "Search query"
// @Success      200 {array}  dtos.GameSystem
// @Failure      400 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Router       /v1/game-systems [get]
func (h *gameSystemsHandler) search(c fiber.Ctx) error {
	q := c.Query("q")
	if q == "" {
		return handleErr(c, usecase.ErrInvalidData)
	}

	systems, err := h.uc.Search(c.Context(), q)
	if err != nil {
		h.log.Error().Err(err).Msg("failed to search systems")
		return handleErr(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(systems)
}

// @Summary      Add a custom game system
// @Tags         game-systems
// @Accept       json
// @Produce      json
// @Param        body  body     CreateGameSystemRequest  true  "Game system name"
// @Success      200   {object} dtos.GameSystem
// @Failure      400   {object} ErrorResponse
// @Failure      401   {object} ErrorResponse
// @Failure      409   {object} ErrorResponse
// @Security     CookieAuth
// @Router       /v1/game-systems [post]
func (h *gameSystemsHandler) addUserSystem(c fiber.Ctx) error {
	var system CreateGameSystemRequest
	if err := c.Bind().Body(&system); err != nil {
		h.log.Error().Err(err).Msg("invalid request body in addUserSystem")
		return handleErr(c, usecase.ErrInvalidData)
	}

	added, err := h.uc.AddUserSystem(c.Context(), &usecase.CreateGameSystemInput{
		Name: system.Name,
	})
	if err != nil {
		return handleErr(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(added)
}