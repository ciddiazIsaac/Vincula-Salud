.PHONY: help setup test build run lint proto emulators-up

help:
	@echo "Comandos disponibles: setup, test, build, run, lint, proto, emulators-up"

setup:
	go mod download
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

test:
	go test -race -coverprofile=coverage.out ./...

build:
	go build -o bin/clinical-record ./cmd/server

run:
	go run ./cmd/server

lint:
	golangci-lint run

proto:
	protoc --go_out=. --go-grpc_out=. api/v1/*.proto

emulators-up:
	gcloud beta emulators pubsub start --project=vincula-salud-dev --host-port=localhost:8085 &
	gcloud beta emulators spanner start --host-port=localhost:9010 &
