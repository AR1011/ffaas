.PHONY: proto

build:
	@go build -o bin/api cmd/api/main.go 
	@go build -o bin/wasmserver cmd/wasmserver/main.go 

wasmserver: build
	@./bin/wasmserver

api: build
	@./bin/api --seed

test:
	@go test ./pkg/* -v

proto:
	protoc --go_out=. --go_opt=paths=source_relative --proto_path=. proto/types.proto

clean:
	@rm -rf bin/ffaas
	@rm -rf bin/wasmserver
	@rm ./examples/*/*.wasm

goex:
	GOOS=wasip1 GOARCH=wasm go build -o examples/go-endpoint/endpoint.wasm examples/go-endpoint/main.go 
	GOOS=wasip1 GOARCH=wasm go build -o examples/go-cron/cron.wasm examples/go-cron/main.go
	GOOS=wasip1 GOARCH=wasm go build -o examples/go-process/process.wasm examples/go-process/main.go

redis-up:
	docker compose -f ./redis.yml up -d

redis-down:
	docker compose -f ./redis.yml down