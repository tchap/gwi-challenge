# gwi-api

This executable package contains the main GWI challenge executable, i.e. the API.

## Build

You can build the executable as is using Go (1.12 needed for Go modules),
but there is also a Dockerfile available, which contains everything needed.

Last but not least, there is a Docker Compose file that can be used
to build and run the API backed by Postgres without much effort.

## Usage

The API is a backend service. It must be configured properly
using environment variables. This is not documented explicitly.
Please consult `config.go`, `Config` struct there is populated from the env
on startup. Just check the `envconfig` tags and prepend `GWI_API_`.
You can also see what values are required and what values are the default.

`docker-compose.yml` can be also used to check how to configure the service.

In case the service is being deployed using the PostgreSQL store,
initial migrations must be performed using `gwi-api db migrate` command.
The same environmental variables as for the main service command are expected
to be present. This is being taken care of in case of Docker Compose.

## API Documentation

The API is not documented as of now (talking Swagger and friends).
This can be added, but for out purposes I hope that it is enough to just check `test.bash`
and see clearly what endpoints are called with what payloads.

Diving in code itself, `initEcho` in `main.go` contains endpoint registration
logic where all API paths can be seen clearly.

## Tests

Tests are not complete in the way that they do not cover the whole API,
but relevant pieces are implemented for demonstration.
There are a few test files, namely:

* `api/api_test.go` for unit tests,
* `api/api_integration_test.go` for some integration tests.
# `api/stores/postgrestore/store_test.go` for unit tests for the Postgres store.

Once started, the service can be tested from outside somehow using `test.bash`
(no other black box testing implemented currently).
Please make sure you have `jq` installed before running the script.
The script itself just uses `curl` to call all APIs available and print results.

## Design (Bits and Pieces)

### API and Data Management

There is a single `api.API` object that has its methods bound to Echo HTTP handlers,
i.e. every `api.API` method is an `echo.HandlerFunc`. The API object accepts a store
in the constructor, which is an interface used for storing data.

There is current an in-memory store, mock store for testing and Postgres-backed store implemented.
There is this extra level of indirection (api.Store interface) to make writing tests easier.
It is much more pleasant to mock an object than to mock raw SQL. And also having a simple
in-memory store for testing is also handy.

### Authentication

`gwi-api` uses JWT tokens for authentication. Calling `/v1/volunteers/login`
returns `token` in JSON body on success. This token must be then passed in subsequent
calls as `Authentation: Bearer <token-goes-here>`.

### Stats API

`gwi-api` contains Stats API under `/v1/stats`, which is protected by Basic auth
and which can be used to fetch some stats considering the service.
For our purposes it is used to fetch the team member counts for the cron job.
It is much easier to implement an extra endpoint for what is needed in this case
than to implement more regular endpoints with pagination and what not,
although my opinion here is not carved into stone.

## What is Missing

### Logging

There is not much loggin anywhere. I usually tend to log as much as possible
considering errors, as close to the error source as possible.

### Metrics

I am used to Prometheus metrics, but haven't implemented any here.