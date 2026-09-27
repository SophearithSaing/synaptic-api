BINARY := bin/api

.PHONY: build run test test-integration test-race vet fmt fmt-check tidy \
	ci clean docker-up docker-down docker-logs

build:
	go build -o $(BINARY) ./cmd/api

run:
	go run ./cmd/api

test:
	go test -short ./...

test-integration:
	go test ./...

test-race:
	go test -short -race ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

fmt-check:
	@test -z "$$(gofmt -l .)" || (gofmt -l . && exit 1)

tidy:
	go mod tidy

ci: fmt-check vet test-race build

clean:
	rm -rf bin coverage.out

docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f mongo
