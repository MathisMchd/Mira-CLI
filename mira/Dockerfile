# syntax=docker/dockerfile:1

FROM golang:1.26-alpine AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/mira-api ./cmd/api

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
WORKDIR /app

COPY --from=builder /out/mira-api ./mira-api
COPY docs ./docs
COPY web ./web

EXPOSE 8080
ENTRYPOINT ["./mira-api"]
