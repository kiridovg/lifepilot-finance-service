.PHONY: generate build lint

generate:
	buf generate
	sqlc generate

build:
	go build ./...

lint:
	buf lint
	go vet ./...
