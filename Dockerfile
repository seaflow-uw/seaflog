# syntax=docker/dockerfile:1

# Build the application from source
FROM golang:1.21 AS build-stage

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY seaflog.go seaflog_test.go event_definitions.json ./
COPY cmd ./cmd
RUN CGO_ENABLED=0 GOOS=linux go build -o "/seaflog" cmd/seaflog/main.go

# Run the tests in the container
FROM build-stage AS run-test-stage
RUN go test -v ./... >/seaflog-test.log 2>&1

# Deploy the application binary into a lean image
FROM gcr.io/distroless/base-debian11 AS build-release-stage

WORKDIR /

COPY --from=build-stage /seaflog /usr/local/bin/seaflog
COPY --from=run-test-stage /seaflog-test.log /seaflog-test.log

CMD ["/user/local/bin/seaflog"]