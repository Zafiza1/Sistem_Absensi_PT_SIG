# --- Build stage -------------------------------------------------------
# Keep this in sync with the `go` directive in go.mod.
FROM golang:1.26-alpine AS builder

WORKDIR /src

# Cache module downloads separately from source changes.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO disabled -> fully static binary, no libc dependency in the final image.
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/migrate ./cmd/migrate

# --- Runtime stage -------------------------------------------------------
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S app && adduser -S app -G app

WORKDIR /app

COPY --from=builder /out/server ./server
COPY --from=builder /out/migrate ./migrate
COPY migrations ./migrations

USER app

EXPOSE 8080

ENTRYPOINT ["./server"]
