FROM golang:1.22-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o auth ./cmd/main.go

FROM alpine:3.17
WORKDIR /app

RUN apk add --no-cache postgresql-client

COPY --from=builder /app/auth /auth

COPY --from=builder /app/migrations /app/migrations
COPY --from=builder /app/entrypoint.sh /entrypoint.sh

RUN chmod +x /entrypoint.sh

EXPOSE 8081
ENV AUTH_PORT=8081

ENTRYPOINT ["/entrypoint.sh"]

