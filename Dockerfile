FROM golang:1.27-alpine3.24 AS builder

WORKDIR /app

# Download modules in their own layer so code changes reuse the cache.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

# Alpine rather than distroless so container healthchecks can use wget.
FROM alpine:3.24

RUN apk --no-cache add ca-certificates tzdata \
    && addgroup -S app \
    && adduser -S -G app app \
    && mkdir -p /var/lib/csapi/mqtt \
    && chown -R app:app /var/lib/csapi

WORKDIR /app
COPY --from=builder /out/server /app/server

USER app
EXPOSE 8080 1883 9001

# Configure with environment variables (DATABASE_HOST, API_BASE_URL, ...) or a
# config.yaml mounted at /app/config.yaml. `docker run <image> migrate` applies
# pending database migrations and exits.
ENTRYPOINT ["/app/server"]
