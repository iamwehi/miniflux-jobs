# Build stage
FROM docker.io/golang:latest AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod tidy && go mod download

COPY src/ ./src/

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o miniflux-jobs ./src/

FROM alpine:latest

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/miniflux-jobs .

RUN adduser -D -u 1000 appuser
USER appuser

ENTRYPOINT ["./miniflux-jobs"]
CMD ["-config", "/app/rules.yaml"]
