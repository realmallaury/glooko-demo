.PHONY: run build deps

export MONGODB_URI=mongodb://127.0.0.1:27017
export MONGODB_NAME=glooko
export SERVER_PORT=:8080

# MongoDB settings
MONGO_CONTAINER=mongo-dev
MONGO_PORT=27017

run:
	@go run cmd/main.go

test:
	@go test ./...

seed:
	@go run cmd/seed/main.go

deps:
	go mod tidy
	go get -u all

mocks:
	@./mockery --name UserRepository --output mocks --outpkg mocks --case underscore --dir=internal/ports
	@./mockery --name DeviceRepository --output mocks --outpkg mocks --case underscore --dir=internal/ports
	@./mockery --name ReadingRepository --output mocks --outpkg mocks --case underscore --dir=internal/ports

run-mongo:
	@echo "Starting MongoDB container..."
	@sh -c "docker run --name $(MONGO_CONTAINER) -p 27017:$(MONGO_PORT) -d mongo:latest"
	@echo "MongoDB is running on port $(MONGO_PORT)"

stop-mongo:
	@echo "Stopping MongoDB container..."
	@docker stop $(MONGO_CONTAINER)
	@docker rm $(MONGO_CONTAINER)
	@echo "MongoDB container stopped"
