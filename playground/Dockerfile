# ==========================================
# STAGE 1: Build Environment Container
# ==========================================
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Ingest package configurations to pre-cache dependency tracking maps
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Compile optimized static standalone machine binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o server ./cmd/server

# ==========================================
# STAGE 2: Secure Production Runtime
# ==========================================
FROM scratch

WORKDIR /root/

# Copy the lightweight machine binary out of Stage 1
COPY --from=builder /app/server .

EXPOSE 8080

CMD ["./server"]
