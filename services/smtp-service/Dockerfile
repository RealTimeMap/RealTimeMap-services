# syntax=docker/dockerfile:1.7

# ===============================================
# Stage 1: Base builder (кэширование зависимостей)
# ===============================================
FROM golang:1.26.1-alpine AS base-builder

RUN apk add --no-cache git gcc musl-dev

WORKDIR /build

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download && go mod verify

# ===============================================
# Stage 2: Service builder
# ===============================================
FROM base-builder AS service-builder

# Общие пакеты нужны целиком: сервис опирается на transport, database,
# logger, runner и apperror.
COPY pkg/ ./pkg/
COPY services/smtp-service/ ./services/smtp-service/

# Шаблоны писем встраиваются в бинарник через embed.FS, поэтому отдельно
# в runtime-образ их копировать не нужно.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
    -ldflags="-w -s -extldflags '-static'" \
    -a -installsuffix cgo \
    -o /app/smtp-service \
    ./services/smtp-service/cmd/smtp-service/main.go

# ===============================================
# Stage 3: Runtime
# ===============================================
FROM alpine:3.19 AS runtime

# ca-certificates обязателен: SMTP-соединение с провайдером идёт по TLS,
# и без корневых сертификатов проверка подлинности сервера не пройдёт.
RUN apk add --no-cache ca-certificates tzdata wget && \
    addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup && \
    mkdir -p /app/config && \
    chown -R appuser:appgroup /app

WORKDIR /app

COPY --from=service-builder --chown=appuser:appgroup /app/smtp-service ./smtp-service

# Конфиг с секретами в образ намеренно НЕ копируется: рабочий config.yaml
# содержит пароль SMTP, а образ уходит в реестр и читается всеми, у кого есть
# доступ к нему. Вместо файла — docker-config.yaml без секретов; пароль, креды
# БД и брокеры приходят из окружения и перекрывают его (ENV читается после
# файла, см. pkg/config).
COPY --chown=appuser:appgroup services/smtp-service/config/docker-config.yaml ./config/config.yaml

USER appuser

EXPOSE 8080

# Health отражает и доступность БД, и состояние очереди: отправка не имеет
# обратной связи в интерфейсе, и вставшая очередь иначе остаётся незамеченной.
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget --quiet --tries=1 --spider http://localhost:8080/smtp/health || exit 1

ENTRYPOINT ["/app/smtp-service"]
