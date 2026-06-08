.PHONY: deps format lint test build clean migrate-status migrate-up serve worker

BIN := bin/backend-go.exe

ifeq ($(cov),1)
TEST_FLAGS := -v -coverprofile=coverage.out ./...
else
TEST_FLAGS := -v ./...
endif

deps:
	go mod download
	go mod tidy

format:
	go fmt ./...

lint:
	go vet ./...

test:
	go test $(TEST_FLAGS)
	@if [ "$(cov)" = "1" ]; then go tool cover -html=coverage.out -o coverage.html; fi

build:
	GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o $(BIN) ./cmd/backend-go

migrate-status:
	go run ./cmd/backend-go migrate status

migrate-up:
	go run ./cmd/backend-go migrate up

serve:
	go run ./cmd/backend-go serve

worker:
	go run ./cmd/backend-go worker

clean:
	rm -rf bin coverage.out coverage.html
