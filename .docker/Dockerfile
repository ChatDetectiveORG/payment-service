# syntax=docker/dockerfile:1.4

# --- Build stage ---
FROM golang:1.24-bookworm AS builder
WORKDIR /workspace
COPY . ./
WORKDIR /workspace/payment-service
RUN CGO_ENABLED=0 GOOS=linux go build -o /workspace/app ./src

# --- Production image ---
FROM alpine:3.19 AS prod
WORKDIR /app
COPY --from=builder /workspace/app /app/

RUN apk --no-cache add ca-certificates

ENTRYPOINT ["/app/app"]

# --- Development image ---
FROM golang:1.24-bookworm AS dev
WORKDIR /workspace

RUN go install github.com/air-verse/air@latest

COPY . ./
WORKDIR /workspace/payment-service

CMD ["air"]
