.PHONY: run test migrate-up migrate-down

run:
	go run ./cmd/api

test:
	go test ./... -v

dbcheck:
	go run ./cmd/dbcheck

migrate-up:
	docker exec -i go-url-shortener-postgres \
		psql -v ON_ERROR_STOP=1 -U app -d urlshortener \
		< migrations/001_create_urls.up.sql

migrate-down:
	docker exec -i go-url-shortener-postgres \
		psql -v ON_ERROR_STOP=1 -U app -d urlshortener \
		< migrations/001_create_urls.down.sql
