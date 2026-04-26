FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

FROM alpine:3.19
RUN apk add --no-cache ca-certificates curl && \
    curl -sSf https://atlasgo.sh | sh
COPY --from=builder /app/server /server
COPY --from=builder /app/internal/db/migrations /internal/db/migrations
EXPOSE 8080
CMD ["/server"]
