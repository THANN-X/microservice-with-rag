module auth_service

go 1.25.4

require (
	authmiddleware v0.0.0
	database v0.0.0
	errs v0.0.0
	github.com/gofiber/fiber/v2 v2.52.10
	github.com/joho/godotenv v1.5.1
	golang.org/x/crypto v0.46.0
	gorm.io/gorm v1.31.1
	httpcore v0.0.0
	jwtutils v0.0.0
	logs v0.0.0
	gorm_helper v0.0.0
)

require (
	github.com/andybalholm/brotli v1.1.0 // indirect
	github.com/golang-jwt/jwt/v5 v5.3.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.7.6 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/klauspost/compress v1.17.9 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/rivo/uniseg v0.2.0 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasthttp v1.51.0 // indirect
	github.com/valyala/tcplisten v1.0.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.1 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/text v0.32.0 // indirect
	gorm.io/driver/postgres v1.6.0 // indirect
)

replace logs => ../../pkg/logs

replace errs => ../../pkg/utils/errs

replace jwtutils => ../../pkg/utils/jwtutils

replace database => ../../pkg/utils/database

replace httpcore => ../../pkg/utils/httpcore

replace gorm_helper => ../../pkg/utils/gorm_helper

replace authmiddleware => ../../pkg/middleware
