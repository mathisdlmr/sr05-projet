FROM golang:1.23-alpine

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN mkdir -p bin \
    & go build -o bin/application ./cmd/application \
    & go build -o bin/control ./cmd/control

ENTRYPOINT ["/app/bin/application"]