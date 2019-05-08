.PHONY: build test docker/build compose/build example

DOCKER_TAG ?= gwi-api

# build builds the service executable while passing the build number into the executable.
build: BUILD_NUMBER ?= $(shell git rev-parse --abbrev-ref HEAD)-$(shell git rev-parse --short HEAD)
build:
	go build -ldflags="-X main.BuildVersion=$(BUILD_NUMBER)"

# test runs all available tests.
test:
	go test -v ./...

# docker/build builds the Docker image, also parring the build number inside.
docker/build: BUILD_NUMBER ?= $(shell git rev-parse --abbrev-ref HEAD)-$(shell git rev-parse --short HEAD)
docker/build:
	docker build --tag $(DOCKER_TAG) --build-arg BUILD_NUMBER="$(BUILD_NUMBER)" $(DOCKER_FLAGS) .

# compose/build builds the Docker Compose service.
compose/build: BUILD_NUMBER ?= $(shell git rev-parse --abbrev-ref HEAD)-$(shell git rev-parse --short HEAD)
compose/build:
	docker-compose build --build-arg BUILD_NUMBER="$(BUILD_NUMBER)" api

# example sets everything up, then runs the example script, then tearing everything down.
#
# Starting the service is hacked together here.
# docker-compose run is called twice since the first call may not start the service
# since the DB is not ready yet. But the API will PROBABLY come up on the second call.
# This is usually solved by waiting on the DB port using dockerize command or something,
# but omitting it here for time reasons.
example: compose/build
	@echo '---> Start the service'
	@docker-compose run --detach --service-ports api
	@sleep 5
	@docker-compose run --detach --service-ports api
	@echo '---> Running the example script'
	@./test.bash
	@echo '---> Cleaning up'
	@docker-compose rm -sf