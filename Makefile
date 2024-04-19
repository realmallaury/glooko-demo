.PHONY: run build deps

export MONGODB_URI="mongodb://localhost:27017"
export MONGODB_NAME="glooko"
export SERVER_PORT="8080"

run:
	go run cmd/main.go

deps:
	go mod tidy
	go get -u all