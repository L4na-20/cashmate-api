# =========================
# Stage 1 — Build Binary
# =========================
FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

# Cache dependency downloads
COPY go.mod go.sum ./
RUN go mod download

# Copy seluruh source code
COPY . .

# Build binary (static, tanpa CGO agar ringan di alpine)
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o cashmate-api .

# =========================
# Stage 2 — Runner Minimalis
# =========================
FROM alpine:latest

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy binary dari stage sebelumnya
COPY --from=builder /app/cashmate-api .

EXPOSE 8096

ENTRYPOINT ["./cashmate-api"]
