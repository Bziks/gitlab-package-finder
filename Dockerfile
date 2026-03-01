FROM golang:1.26-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /app ./cmd

FROM alpine:3

RUN addgroup -g 1000 appgroup && \
    adduser -S -u 1000 -G appgroup appuser

COPY --from=builder --chown=appuser:appgroup /app /app/app
COPY --from=builder --chown=appuser:appgroup /build/migrations /app/migrations
COPY --from=builder --chown=appuser:appgroup /build/public /app/public

USER 1000
WORKDIR /app

ENTRYPOINT ["/app/app"]
CMD ["api"]
