# Start the PostgreSQL container using docker-compose
start-db:
	sudo docker-compose up -d

# Show logs from the PostgreSQL container
db-logs:
	sudo docker logs postgresDB

# Create a new PostgreSQL database named 'synapse'
create-db:
	sudo docker exec -it postgresDB createdb --username=postgres --password --owner=postgres synapse

# Drop the 'synapse' database
drop-db:
	sudo docker exec -it postgresDB dropdb --username=postgres --password synapse

# Open a psql shell into the 'synapse' database
db-shell:
	sudo docker exec -it postgresDB psql -U postgres --password -d synapse

# Open an interactive shell inside the Go development Docker container
go-shell:
	sudo docker exec -it go_dev_env bash

# Show the current migration version
migration-version:
	sudo docker run --rm -v ./db/migration/:/migration \
		--network host migrate/migrate \
		-source file:///migration \
		-database postgres://postgres:secret@localhost:5432/synapse?sslmode=disable \
		version

# Create a new migration file with a custom name (use `make new-migration name=create_users`)
new-migration:
	@if [ -z "$(name)" ]; then \
		echo "Error: You must provide a name for the migration (e.g., name=create_users)"; \
		exit 1; \
	fi; \
	echo "Creating new migration: $(name)"; \
	sudo docker run --rm \
		-v ./db/migration:/migration \
		migrate/migrate create -ext sql -dir /migration -seq $(name)

# Run all or N up migrations (use `make migration-up n=1` to run 1 migration)
migration-up:
	@if [ -z "$(n)" ]; then \
		echo "Running all up migrations..."; \
		sudo docker run --rm -v ./db/migration/:/migration \
			--network host migrate/migrate \
			-source file:///migration \
			-database postgres://postgres:secret@localhost:5432/synapse?sslmode=disable \
			up; \
	else \
		echo "Running $(n) up migrations..."; \
		sudo docker run --rm -v ./db/migration/:/migration \
			--network host migrate/migrate \
			-source file:///migration \
			-database postgres://postgres:secret@localhost:5432/synapse?sslmode=disable \
			up $(n); \
	fi

# Run all or N down migrations (use `make migration-down n=1` to roll back 1 migration)
migration-down:
	@if [ -z "$(n)" ]; then \
		echo "Running all down migrations..."; \
		sudo docker run --rm -v ./db/migration/:/migration \
			--network host migrate/migrate \
			-source file:///migration \
			-database postgres://postgres:secret@localhost:5432/synapse?sslmode=disable \
			down -all; \
	else \
		echo "Running $(n) down migrations..."; \
		sudo docker run --rm -v ./db/migration/:/migration \
			--network host migrate/migrate \
			-source file:///migration \
			-database postgres://postgres:secret@localhost:5432/synapse?sslmode=disable \
			down $(n); \
	fi

# Generate SQL code using sqlc
sqlc-generate:
	sudo docker run --rm -v ./:/src -w /src sqlc/sqlc generate

# Run Go tests with verbose output and coverage
test:
	go test -v -cover ./...

# Run Go Server
server:
	go run main.go

# Create a new admin user via CLI
create-admin:
	@if [ -z "$(name)" ] || [ -z "$(email)" ] || [ -z "$(password)" ]; then \
		echo "Usage: make create-admin name='Alice' email='alice@example.com' password='secret'"; \
		exit 1; \
	fi; \
	go run cmd/admin/main.go --name="$(name)" --email="$(email)" --password="$(password)"

.PHONY: start-db db-logs create-db drop-db db-shell migration-version migration-up migration-down sqlc-generate test new-migration server create-admin
