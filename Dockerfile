# Multi-stage build for Pushem

# Stage 1: Build frontend
FROM node:20-alpine AS frontend-builder

WORKDIR /app/web

COPY web/package*.json ./
RUN npm ci

COPY web/ ./
RUN npm run build

# Stage 2: Build backend
FROM golang:1.23-alpine AS backend-builder

# Install build dependencies for SQLite
RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ ./cmd/
COPY internal/ ./internal/

# Build with CGO enabled for SQLite
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o pushem cmd/server/main.go

# Stage 3: Final runtime image
FROM alpine:latest

RUN apk --no-cache add ca-certificates sqlite-libs

WORKDIR /app

# Copy backend binary
COPY --from=backend-builder /app/pushem .

# Copy frontend build
COPY --from=frontend-builder /app/web/dist ./web/dist

# Expose port
EXPOSE 8080

# Run the application
CMD ["./pushem"]
