FROM golang:1.22-alpine3.19 AS builder

LABEL maintainer="DevOps Team" \
      version="1.0.0" \
      description="Dropi Order Status Service" \
      org.opencontainers.image.source="https://github.com/juancollazo-ch/dropi-order-status-service"

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -trimpath -o main ./cmd

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /app/main /main

ENV PORT=8080 TZ=UTC GIN_MODE=release

EXPOSE 8080

USER 65534:65534

ENTRYPOINT ["/main"]
