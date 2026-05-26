.PHONY: all build build-go build-frontend clean run dev-frontend

all: build

build: build-frontend build-go

build-go:
	@mkdir -p bin
	go build -o bin/application ./cmd/application
	go build -o bin/control ./cmd/control
	go build -o bin/net ./cmd/net
	@echo "Binaires Go compilés dans bin/"

build-frontend:
	@cd frontend && npm install --silent && npm run build
	@echo "Frontend compilé dans web/"

dev-frontend:
	@cd frontend && npm run dev

run: build
	@chmod +x scripts/local_net.sh
	./scripts/local_net.sh 4444

run-ctl: 
	build
	@chmod +x scripts/local_ctl.sh
	./scripts/local_ctl.sh 4444

check:
	go vet ./cmd/... ./internal/... ./pkg/...

clean:
	rm -rf bin/
