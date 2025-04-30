migrate-down:
	migrate down -all

migrate-up:
	migrate up

# Requires a migration name following `make migration`. Ex: make migration descriptive_name
migration:
	migrate create -ext sql -dir db/migrations $(filter-out $@,$(MAKECMDGOALS))

sqlc:
	sqlc generate

test:
	go test -count=1 -cover -race ./...

.PHONY: migrate-down migrate-up migration sqlc test
