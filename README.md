# Glooko API Project

This project provides the backend API for the Glooko platform, handling device data management and providing device overviews to customer support staff.

## Prerequisites

- Go (Golang) - Ensure you have Go installed on your system to run and test the application.
- Docker - Required for running a MongoDB container locally.
- MongoDB - Used as the primary data store.
- Mockery - Used for generating mocks in testing.

## Environment Variables

Before running the application, make sure to set the necessary environment variables:

- `MONGODB_URI`: The URI connection string for MongoDB. (e.g. mongodb://127.0.0.1:27017)
- `MONGODB_NAME`: The database name for MongoDB. (e.g. glooko)
- `SERVER_PORT`: The port on which the server will listen. (e.g. :8080)

## Makefile Commands

The Makefile includes several commands that facilitate running, testing, and managing the application and its dependencies:

### `make run`

Runs the main application. It requires the MongoDB connection to be available at the specified `MONGODB_URI`.

### `make test`

Runs all unit tests in the project to ensure the application behaves as expected.

### `make seed`

Executes a seeding script to populate the MongoDB database with initial data, useful for setting up a development environment with sample data.

### `make deps`

Updates and tidies project dependencies using Go modules.

### `make mocks`

Generates mock implementations for repository interfaces using Mockery, facilitating unit testing of components that interact with these repositories.

### `make run-mongo`

Starts a MongoDB container locally using Docker. This is useful for development purposes, providing a MongoDB instance running on the default port.

### `make stop-mongo`

Stops and removes the MongoDB container. Use this to clean up after development sessions.
