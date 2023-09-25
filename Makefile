MIGRATIONS_SOURCE = "file://migrations"
DATABASE_DSN = "postgres://ecom-service:qazwsx@127.0.0.1/ecom-service?sslmode=disable"

run:
	./build/bin/gateway -config ./config/config.yml

all:
	go build -o ./build/bin/ ./cmd/gateway/
	go build -o ./build/bin/ ./cmd/loadcurrencies/
	go build -o ./build/bin/ ./cmd/loadrates/
	go build -o ./build/bin/ ./cmd/migrate/
	go build -o ./build/bin/ ./cmd/updatetransactions/

test:
	go test ./cmd/gateway/

migrate:
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

swag:
	go install github.com/swaggo/swag/cmd/swag@latest

swag-init:
	swag init -d ./cmd/gateway/ --parseDependency

migrate-up:
	migrate -source ${MIGRATIONS_SOURCE} -database ${DATABASE_DSN} up

migrate-down:
	migrate -source ${MIGRATIONS_SOURCE} -database ${DATABASE_DSN} down 1
