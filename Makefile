dev:
	docker compose up --build

down:
	docker compose down

run:
	go run cmd/server/main.go

migrate-up:
	docker compose run --rm migrate migrate -path /app/migrations -database "postgresql://neondb_owner:npg_2PIXQnyYwr1g@ep-wispy-butterfly-ae4z8a1x-pooler.c-2.us-east-2.aws.neon.tech/neondb?sslmode=require&channel_binding=require" up

migrate-down:
	docker compose run --rm migrate migrate -path /app/migrations -database "$${DATABASE_URL}" down

sqlc:
	sqlc generate

docker-build:
	docker compose build

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f api

docker-restart:
	docker compose restart api

docker-prod:
	docker build --target production -t campuscare:prod .