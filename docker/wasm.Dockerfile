FROM golang:1.21.4 as builder

WORKDIR /builder

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o wasm-server ./cmd/wasmserver/main.go

FROM alpine:latest as production

EXPOSE 5000

RUN apk --no-cache add ca-certificates

WORKDIR /wasm

COPY --from=builder /builder .

COPY config.toml .

CMD ["./wasm-server"]
