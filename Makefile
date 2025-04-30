migrate-down:
	migrate down -all

migrate-up:
	migrate up

# Requires a migration name following `make migration`. Ex: make migration descriptive_name
migration:
	migrate create -ext sql -dir db/migrations $(filter-out $@,$(MAKECMDGOALS))

.PHONY: migrate-down migrate-up migration
