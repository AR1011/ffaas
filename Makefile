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
	@rm -rf bin/run
	@rm -rf bin/wasmserver
	@rm ./examples/*/*.wasm

goex:
	GOOS=wasip1 GOARCH=wasm go build -o examples/go/endpoint/endpoint.wasm examples/go/endpoint/main.go 
	GOOS=wasip1 GOARCH=wasm go build -o examples/go/cron/cron.wasm examples/go/cron/main.go
	GOOS=wasip1 GOARCH=wasm go build -o examples/go/process/process.wasm examples/go/process/main.go

jsex:
	javy compile examples/js/index.js -o examples/js/index.wasm

redis-up:
	docker compose -f ./docker/redis.yml up -d

redis-down:
	docker compose -f ./docker/redis.yml down

deploy-up:
	docker compose -f ./docker/deploy.yml up -d

deploy-down:
	docker compose -f ./docker/deploy.yml down

build-containers:
	docker build -t api-server -f ./docker/api.Dockerfile .
	docker build -t wasm-server -f ./docker/wasm.Dockerfile .
	