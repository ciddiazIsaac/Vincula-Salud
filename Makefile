.PHONY: help setup test build run lint proto emulators-up spanner-init run-bridge

help:
	@echo "Comandos disponibles: setup, test, build, run, lint, proto, emulators-up, spanner-init, run-bridge"

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

spanner-init:
	SPANNER_EMULATOR_HOST=localhost:9010 gcloud spanner instances create dev-instance --config=emulator-config --description="Dev Instance" --nodes=1 --project=vincula-salud-dev
	SPANNER_EMULATOR_HOST=localhost:9010 gcloud spanner databases create clinical-db --instance=dev-instance --project=vincula-salud-dev --ddl="CREATE TABLE ClinicalEvents (PatientRun STRING(MAX) NOT NULL, EventId STRING(MAX) NOT NULL, EventType STRING(MAX), EventDataJson BYTES(MAX), AuthorCredential STRING(MAX), RecordedAt TIMESTAMP, EventTimestamp TIMESTAMP) PRIMARY KEY(PatientRun, EventId)"

run-bridge:
	go run cmd/legacy_bridge/main.go
