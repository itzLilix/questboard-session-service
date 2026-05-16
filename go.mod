module github.com/itzLilix/questboard-session-service

go 1.26.1

require github.com/itzLilix/questboard-shared v1.0.0

replace github.com/itzLilix/questboard-shared => ./../questboard-shared

require (
	github.com/gofiber/fiber/v3 v3.1.0
	github.com/jackc/pgx/v5 v5.5.4
)

require (
	github.com/golang-jwt/jwt/v5 v5.3.1 // indirect
	github.com/gosimple/slug v1.15.0
	github.com/gosimple/unidecode v1.0.1 // indirect
	github.com/jackc/pgerrcode v0.0.0-20220416144525-469b46aa5efa // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
	github.com/lann/builder v0.0.0-20180802200727-47ae307949d0 // indirect
	github.com/lann/ps v0.0.0-20150810152359-62de8c46ede0 // indirect
	golang.org/x/sync v0.20.0 // indirect
)

require (
	github.com/Masterminds/squirrel v1.5.4
	github.com/andybalholm/brotli v1.2.0 // indirect
	github.com/gofiber/schema v1.7.0 // indirect
	github.com/gofiber/utils/v2 v2.0.2 // indirect
	github.com/golang-migrate/migrate/v4 v4.19.1
	github.com/google/uuid v1.6.0 // indirect
	github.com/joho/godotenv v1.5.1
	github.com/klauspost/compress v1.18.4 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/philhofer/fwd v1.2.0 // indirect
	github.com/rs/zerolog v1.35.0
	github.com/tinylib/msgp v1.6.3 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasthttp v1.69.0 // indirect
	golang.org/x/crypto v0.50.0 // indirect
	golang.org/x/net v0.52.0 // indirect
	golang.org/x/sys v0.43.0 // indirect
	golang.org/x/text v0.36.0 // indirect
)
