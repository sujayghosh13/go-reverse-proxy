# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o proxy .

# Final minimal stage
FROM alpine:3.19

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/proxy .
COPY --from=builder /app/config.yaml .
COPY --from=builder /app/cert.pem .
COPY --from=builder /app/key.pem .

EXPOSE 8080 9090

ENTRYPOINT ["./proxy"]
