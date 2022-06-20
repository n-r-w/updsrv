.PHONY: build test run runbuild codegen rebuild tidy race docker-up docker-up_d docker-down docker-attach

build:	
	go build -v -o . ./cmd/updsrv.go

rebuild:
	wire ./internal/di
	go build -a -v -o . ./cmd/updsrv.go

race:
	wire ./internal/di
	go run -race ./cmd/updsrv.go -config-path ./config.toml

run:	
	go run ./cmd/updsrv.go -config-path ./config.toml

runbuild:
	./bin/updsrv

codegen:
	wire ./internal/di

tidy:
	go mod tidy

docker-up:
	docker-compose up --build

docker-up_d:
	docker-compose up -d --build

docker-down:
	docker-compose down

docker-attach:
	docker attach --sig-proxy=false updsrv-updsrv-1

.DEFAULT_GOAL := run
