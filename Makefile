.PHONY: all build build-go build-frontend clean run dev-frontend

all: build

build: build-frontend build-go

build-go:
	@mkdir -p bin
	go build -o bin/application ./cmd/application
	go build -o bin/control ./cmd/control
	@echo "Binaires Go compilés dans bin/"

build-frontend:
	@cd frontend && npm install --silent && npm run build
	@echo "Frontend compilé dans web/"

dev-frontend:
	@cd frontend && npm run dev

run: build
	@chmod +x scripts/local.sh
	./scripts/local.sh 4444

check:
	go vet ./cmd/... ./internal/... ./pkg/...

clean:
	rm -rf bin/
