.PHONY: build server agent clean test lint migrate-up migrate-down

BIN_DIR := ./bin
SERVER_BIN := $(BIN_DIR)/waf-server
AGENT_BIN := $(BIN_DIR)/waf-agent

build: server agent

server:
	go build -o $(SERVER_BIN) ./cmd/server/

agent:
	go build -o $(AGENT_BIN) ./cmd/agent/

clean:
	rm -rf $(BIN_DIR)

test:
	go test ./... -v -count=1

lint:
	go vet ./...

migrate-up:
	migrate -path internal/store/migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path internal/store/migrations -database "$(DATABASE_URL)" down 1

run-server:
	go run ./cmd/server/

run-agent:
	go run ./cmd/agent/
