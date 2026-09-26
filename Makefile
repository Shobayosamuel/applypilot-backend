.PHONY: run build test vet fmt docker

run:            ## run the API with .env loaded
	set -a; [ -f .env ] && . ./.env; set +a; go run ./cmd/api

build:
	CGO_ENABLED=0 go build -o bin/api ./cmd/api

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

docker:         ## db + migrations + api (needs ../applypilot-migrations)
	docker compose up --build
