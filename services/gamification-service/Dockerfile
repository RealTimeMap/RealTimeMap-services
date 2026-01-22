# ===============================================
# Stage 1: Base builder (кэширование зависимостей)
# ===============================================
FROM golang:1.25.1-alpine AS base-builder

# Установка зависимостей для компиляции
RUN apk add --no-cache git gcc musl-dev

WORKDIR /build

# Копируем go.mod и go.sum для кэширования зависимостей
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# ===============================================
# Stage 2: Service builder (компиляция сервиса)
# ===============================================
FROM base-builder AS service-builder

# Копируем общие пакеты и код сервиса
COPY pkg/ ./pkg/
COPY services/gamification-service/ ./services/gamification-service/

# Компиляция с оптимизациями
# CGO_ENABLED=0 для статической линковки
# -ldflags для уменьшения размера бинарника
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o /app/gamification-service \
    ./services/gamification-service/cmd/gamification-service/main.go

# ===============================================
# Stage 3: Runtime (минимальный образ)
# ===============================================
FROM alpine:3.19 AS runtime

# Установка базовых утилит
RUN apk add --no-cache ca-certificates tzdata wget && \
    # Создание непривилегированного пользователя
    addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup && \
    # Создание директорий для хранилища и конфигурации
    mkdir -p /app/store /app/config && \
    chown -R appuser:appgroup /app

WORKDIR /app

# Копируем скомпилированный бинарник из builder stage
COPY --from=service-builder --chown=appuser:appgroup /app/gamification-service .

# Копируем Docker config как default конфигурацию
COPY --chown=appuser:appgroup services/gamification-service/config/config.yaml ./config/config.yaml

# Переключаемся на непривилегированного пользователя
USER appuser

# Expose порт приложения
EXPOSE 8081

# Entrypoint
ENTRYPOINT ["/app/gamification-service"]
