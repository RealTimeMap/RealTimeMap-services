# syntax=docker/dockerfile:1.7
# ==========================================================
# Универсальный Dockerfile для Go-сервисов монорепы.
#
# Сборка:
#   docker build --build-arg SERVICE=mark-service -f build/Dockerfile .
# ==========================================================
ARG GO_VERSION=1.26.1
ARG ALPINE_VERSION=3.22

# ==========================================================
# Stage 1: deps — слой зависимостей, общий для всех сервисов
# ==========================================================
FROM golang:${GO_VERSION}-alpine AS deps
WORKDIR /build

# Отдельный слой: инвалидируется только при правке go.mod/go.sum,
# а не при любом изменении кода сервиса.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# ==========================================================
# Stage 2: builder — компиляция конкретного сервиса
# ==========================================================
FROM deps AS builder
ARG SERVICE
ARG TARGETARCH

COPY pkg/ ./pkg/
COPY services/${SERVICE}/ ./services/${SERVICE}/

# Кэш-маунты общие для всех сервисов монорепы:
# /go/pkg/mod    — скачанные модули
# /root/.cache/go-build — кэш компиляции пакетов
# -trimpath убирает пути сборочной машины из бинарника
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH:-amd64} \
    go build -trimpath -ldflags='-w -s' \
    -o /out/service \
    ./services/${SERVICE}/cmd/${SERVICE}

# ==========================================================
# Stage 3: runtime
# ==========================================================
FROM alpine:${ALPINE_VERSION} AS runtime
ARG SERVICE

RUN apk add --no-cache ca-certificates tzdata wget && \
    addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup && \
    mkdir -p /app/store /app/config && \
    chown -R appuser:appgroup /app

WORKDIR /app

COPY --from=builder --chown=appuser:appgroup /out/service /app/service
COPY --chown=appuser:appgroup services/${SERVICE}/config/config.yaml ./config/config.yaml

USER appuser
EXPOSE 8080
ENTRYPOINT ["/app/service"]
