# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /microtip-app ./cmd/server/main.go

# Run stage
FROM alpine:3.19

WORKDIR /root/
COPY --from=builder /microtip-app .

EXPOSE 8080
CMD ["./microtip-app"]