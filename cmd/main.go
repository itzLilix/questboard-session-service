package main

import (
	"os"
	"time"

	sq "github.com/Masterminds/squirrel"
	swaggo "github.com/gofiber/contrib/v3/swaggo"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/static"
	_ "github.com/itzLilix/questboard-session-service/docs"
	"github.com/itzLilix/questboard-session-service/internal/config"
	"github.com/itzLilix/questboard-session-service/internal/handlers"
	"github.com/itzLilix/questboard-session-service/internal/infrastructure"
	"github.com/itzLilix/questboard-session-service/internal/middleware"
	"github.com/itzLilix/questboard-session-service/internal/usecase"
	"github.com/itzLilix/questboard-shared/jwt"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// @title          Session Zero Session Service
// @version        1.0
// @description    Session, game system, campaign and character API for Session Zero
// @host           localhost:3001
// @BasePath       /
// @securityDefinitions.apikey  CookieAuth
// @in             cookie
// @name           access_token
func main() {
	log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}).
		With().Timestamp().Logger()

	err := godotenv.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("error loading .env file")
	}

	cfg := config.Load()
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	app := fiber.New()
	app.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowCredentials: true,
		AllowHeaders:     []string{"Content-Type"},
	}))
	app.Use(middleware.Logger(log.Logger))

	app.Get("/uploads/*", static.New(cfg.UploadDir))
	if cfg.Env != config.ProdEnv {
   		app.Get("/swagger/*", swaggo.HandlerDefault)
	}

	conn, err := infrastructure.Connect(cfg.DatabaseURL, int32(cfg.MinPoolSize), int32(cfg.MaxPoolSize))
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	log.Info().Msg("successfully connected to database")
	defer conn.Close()

	err = infrastructure.RunMigrations(cfg.MigrateURL)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to run migrations")
	}
	log.Info().Msg("migrations ran successfully")

	tokenParser := jwt.NewParser([]byte(cfg.JWTSecret))
	rbacMiddleware := middleware.NewRBACMiddleware(tokenParser, log.Logger)

	gameSystemsRepo := infrastructure.NewGameSystemsRepository(conn, psql)
	sessionRepo := infrastructure.NewSessionRepository(conn, psql)
	campaignRepo := infrastructure.NewCampaignRepository(conn, psql)
	profileClient := infrastructure.NewHTTPProfileClient(cfg.ProfileServiceURL, cfg.InternalToken)

	gameSystemsUsecase := usecase.NewGameSystemsUsecase(gameSystemsRepo)
	sessionUsecase := usecase.NewSessionUsecase(sessionRepo, profileClient, profileClient)
	campaignUsecase := usecase.NewCampaignUsecase(campaignRepo)
	characterUsecase := usecase.NewCharacterUsecase()

	v1 := app.Group("/v1")

	handlers.NewGameSystemsHandler(gameSystemsUsecase, log.Logger, rbacMiddleware).RegisterRoutes(v1)
	handlers.NewSessionHandler(sessionUsecase, rbacMiddleware, log.Logger).RegisterRoutes(v1)
	handlers.NewCampaignHandler(campaignUsecase, rbacMiddleware, log.Logger).RegisterRoutes(v1)
	handlers.NewCharacterHandler(characterUsecase, rbacMiddleware, log.Logger).RegisterRoutes(v1)

	log.Fatal().Err(app.Listen(":" + cfg.ServerPort)).Msg("server stopped")
}