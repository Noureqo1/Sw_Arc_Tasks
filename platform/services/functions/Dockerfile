FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o function-service ./cmd/main.go

FROM alpine:3.17
WORKDIR /app

RUN apk add --no-cache docker-cli

COPY --from=builder /app/function-service /function-service

EXPOSE 8082
ENTRYPOINT ["/function-service"]

