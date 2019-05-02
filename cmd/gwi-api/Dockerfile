# Builder stage to build statically-linked Go executable
FROM golang:1.12-alpine AS builder

# Install git to be able to fetch dependencies.
RUN apk update && apk add --no-cache git

# Download dependencies into vendor folder.
# This is here before copying the whole folder to improve Docker caching.
WORKDIR /tmp/gwi-api
COPY go.mod .
COPY go.sum .
RUN go mod download

# Copy the remaining sources into the builder.
COPY . .

# Compute the executable, make sure cgo is disabled
# since that would precent statically-linked executable.
RUN CGO_ENABLED=0 go build -o /gwi-api

# It would be nice to use FROM scratch,
# but we need /bin/sh to run entrypoint.sh
FROM alpine:3.9

# Listen on all interfaces.
ENV GWI_API_HTTP_HOST "0.0.0.0"

# Just copy the executable from the builder and we are done.
COPY --from=builder /gwi-api /gwi-api
COPY docker_entrypoint.sh /entrypoint.sh
ENTRYPOINT [ "/entrypoint.sh" ]

# Set up healthcheck.
HEALTHCHECK --interval=30s \
            --timeout=30s \
            --start-period=5s \
            --retries=3 \
            CMD ["/gwi-api", "healthcheck"]