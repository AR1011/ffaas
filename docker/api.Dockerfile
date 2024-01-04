FROM golang:1.21.4 as builder

WORKDIR /builder

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o api-server ./cmd/api/main.go

FROM alpine:latest as production

EXPOSE 3000

RUN apk --no-cache add ca-certificates

WORKDIR /api

COPY --from=builder /builder .

COPY config.toml .

CMD ["./api-server", "--seed"]
