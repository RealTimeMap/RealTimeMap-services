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
# Ограничение нагрузки на CPU при сборке.
# BUILD_JOBS  — сколько пакетов Go компилирует одновременно (go build -p).
# GOMAXPROCS  — сколько ОС-потоков использует сам компилятор.
# Пусто = без ограничений (все ядра). Пример: --build-arg BUILD_JOBS=2
ARG BUILD_JOBS=2
ARG GOMAXPROCS=2

COPY pkg/ ./pkg/
COPY services/${SERVICE}/ ./services/${SERVICE}/

# Кэш-маунты общие для всех сервисов монорепы:
# /go/pkg/mod    — скачанные модули
# /root/.cache/go-build — кэш компиляции пакетов
# -trimpath убирает пути сборочной машины из бинарника
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    env CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH:-amd64} \
    ${GOMAXPROCS:+GOMAXPROCS=${GOMAXPROCS}} \
    go build -trimpath -ldflags='-w -s' \
    ${BUILD_JOBS:+-p ${BUILD_JOBS}} \
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

# config.yaml намеренно НЕ копируется в образ: он содержит реальные креды,
# лежит только на сервере и монтируется внутрь через volume (см. compose).
# Пустой /app/config создан выше — точка монтирования.

USER appuser
EXPOSE 8080
ENTRYPOINT ["/app/service"]
