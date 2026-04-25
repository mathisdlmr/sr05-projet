MODULE  := github.com/sr05/loup-garou

.PHONY: all build clean run-3 test

all: build

build:
	@mkdir -p bin
	go build -o bin/server      ./cmd/server
	go build -o bin/application ./cmd/application
	go build -o bin/control     ./cmd/control
	@echo "Binaires compilés dans bin/"

run-3: build
	@chmod +x scripts/3-local.sh
	./scripts/3-local.sh 4444

check: go vet ./...

clean: rm -rf bin/