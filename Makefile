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

# Run Go tests with verbose output and coverage
test:
	go test -v -cover ./...

.PHONY: test server create-admin
