.PHONY: build server agent clean test lint migrate-up migrate-down generate

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

test-integration:
	go test ./tests/... -v -count=1

lint:
	go vet ./...

generate:
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/agent/agent.proto

migrate-up:
	migrate -path internal/store/migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path internal/store/migrations -database "$(DATABASE_URL)" down 1

run-server:
	go run ./cmd/server/

run-agent:
	go run ./cmd/agent/
