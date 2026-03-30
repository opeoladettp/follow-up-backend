# Build stage
FROM golang:1.25.8-alpine AS builder

# Force cache bust
ENV CACHE_BUST=1

WORKDIR /app

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Ensure go.mod is tidy before building
RUN go mod tidy

# Build with optimisations: strip debug info, disable CGO
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o main .

# Final stage — minimal image
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/main .

# Fly.io sets PORT env var automatically; default to 8080
ENV PORT=8080
ENV GIN_MODE=release

EXPOSE 8080

CMD ["./main"]
