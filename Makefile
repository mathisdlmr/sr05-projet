.PHONY: all build clean run test

all: build

build:
	@mkdir -p bin
	go build -o bin/server ./cmd/server
	go build -o bin/application ./cmd/application
	go build -o bin/control ./cmd/control
	@echo "Binaires compilés dans bin/"

run: build
	@chmod +x scripts/local.sh
	./scripts/local.sh 4444

check: go vet ./...

clean: rm -rf bin/