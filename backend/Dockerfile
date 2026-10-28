# syntax=docker/dockerfile:1.7
FROM golang:1.27.2-alpine3.24 AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

FROM golang:1.27.2-alpine3.24 AS migration-builder
RUN GOBIN=/out go install -tags="no_clickhouse no_libsql no_mssql no_mysql no_sqlite3 no_vertica no_ydb" github.com/pressly/goose/v3/cmd/goose@v3.29.0

FROM alpine:3.24 AS migrations
RUN apk add --no-cache ca-certificates
COPY --from=migration-builder /out/goose /usr/local/bin/goose
COPY db/migrations /migrations
ENTRYPOINT ["goose", "-dir", "/migrations", "postgres"]

FROM alpine:3.24 AS runtime
RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S app \
    && adduser -S -G app app \
    && mkdir -p /var/lib/barbershop/logos \
    && chown app:app /var/lib/barbershop/logos \
    && chmod 0700 /var/lib/barbershop/logos
USER app
COPY --from=builder /out/api /usr/local/bin/api
EXPOSE 8080
ENTRYPOINT ["api"]
