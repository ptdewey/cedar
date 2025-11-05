[private]
default:
    @just --list

run:
    @go run main.go

build:
    @go build

test:
    @go test ./... -cover -coverprofile=cover.out
