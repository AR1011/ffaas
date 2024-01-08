.PHONY: proto

build:
	@go build -o bin/api cmd/api/main.go 
	@go build -o bin/wasmserver cmd/wasmserver/main.go 
	@go build -o bin/raptor cmd/cli/main.go 

wasmserver: build
	@./bin/wasmserver

api: build
	@./bin/api --seed --webui

test:
	@go test ./internal/* -v

proto:
	protoc --go_out=. --go_opt=paths=source_relative --proto_path=. proto/types.proto

clean:
	@rm -rf bin/api
	@rm -rf bin/wasmserver
	@rm -rf bin/raptor

goex:
	GOOS=wasip1 GOARCH=wasm go build -o examples/go/app.wasm examples/go/main.go 

jsex:
	javy compile examples/js/index.js -o examples/js/index.wasm

postgres-up:
	docker compose -f ./docker/postgres.yml up -d

postgres-down:
	docker compose -f ./docker/postgres.yml down

redis-up:
	docker compose -f ./docker/redis.yml up -d

redis-down:
	docker compose -f ./docker/redis.yml down