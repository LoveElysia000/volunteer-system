FROM golang:1.24-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/volunteer-system cmd/main.go

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata gettext

WORKDIR /app

COPY --from=builder /out/volunteer-system /app/volunteer-system
COPY scripts/docker-entrypoint.sh /app/docker-entrypoint.sh

RUN mkdir -p /app/config /app/logs /app/uploads && \
    chmod +x /app/volunteer-system /app/docker-entrypoint.sh

EXPOSE 1109

ENTRYPOINT ["/app/docker-entrypoint.sh"]
CMD ["./volunteer-system", "-c", "server"]
