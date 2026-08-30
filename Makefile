.PHONY: build build-frontend run test lint tidy fmt clean demo

BINARY := leakboard
MODULE := github.com/Laaaaksh/leakboard

build: build-frontend
	go build -o $(BINARY) ./cmd/leakboard

# internal/webui/dist is a committed, real build of web/ (so `go build`/`go
# install` work with no Node install); this target refreshes it. Commit the
# result after any change under web/ - CI fails if it's out of sync.
build-frontend:
	cd web && npm install && npm run build
	rm -rf internal/webui/dist
	cp -r web/dist internal/webui/dist

run:
	go run ./cmd/leakboard

# -p 1 because the Postgres-backed tests in internal/store and internal/api
# truncate shared tables and would race each other across packages otherwise.
test:
	go test ./... -race -cover -p 1

lint:
	golangci-lint run ./...
	cd web && npm run lint

tidy:
	go mod tidy
	cd web && npm install

fmt:
	gofmt -w .
	cd web && npx oxlint --fix src

clean:
	rm -f $(BINARY)
	rm -rf web/dist
	go clean -testcache

# Boots a fresh stack, seeds a demo repo, records the walkthrough, and
# converts it into docs/assets/. See scripts/record-demo/README.md.
demo:
	./scripts/record-demo/run.sh
