FROM golang:1.22-alpine as builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o gateway ./cmd/main.go

FROM alpine:3.17

WORKDIR /app
COPY --from=builder /app/gateway /gateway

EXPOSE 8080

ENV GATEWAY_PORT=8080

ENTRYPOINT ["/gateway"]
