# Tools Manager - minimal build
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o tools-manager ./cmd/server

FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=builder /app/tools-manager /app/tools-manager
COPY --from=builder /app/migrations /app/migrations
EXPOSE 8087
USER nonroot:nonroot
ENTRYPOINT ["/app/tools-manager"]
