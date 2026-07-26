.PHONY: build test web

build: web
	go build -o leaseboard ./cmd/leaseboard

test:
	go test ./...
	cd web && npm run typecheck

web:
	cd web && npm ci && npm run build
