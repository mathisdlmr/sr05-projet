FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN mkdir -p bin && \
    go build -o bin/application ./cmd/application && \
    go build -o bin/control ./cmd/control

FROM alpine:3.19

WORKDIR /app

COPY --from=builder /app/bin/ ./bin/
COPY --from=builder /app/web/ ./web/
