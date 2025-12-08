# RealTimeMap Backend

[![Go](https://img.shields.io/badge/Go-1.23-00ADD8?style=flat&logo=go)](https://go.dev/)
[![Gin](https://img.shields.io/badge/Gin-1.10-00ADD8?style=flat&logo=go)](https://gin-gonic.com/)
[![gRPC](https://img.shields.io/badge/gRPC-1.68-244c5a?style=flat&logo=google)](https://grpc.io/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-336791?style=flat&logo=postgresql)](https://www.postgresql.org/)
[![Redis](https://img.shields.io/badge/Redis-7-DC382D?style=flat&logo=redis)](https://redis.io/)

---

## Содержание

- [1. О проекте](#1-о-проекте)
- [2. Архитектура](#2-архитектура)
- [3. Технологический стек](#3-технологический-стек)
- [4. Структура репозитория](#4-структура-репозитория)
- [5. Микросервисы](#5-микросервисы)
- [6. Быстрый старт](#6-быстрый-старт)
- [7. Разработка](#7-разработка)

---

## 1. О проекте

**RealTimeMap** — backend геосоциальной платформы, позволяющей создавать временные метки на карте и взаимодействовать с происходящим вокруг в режиме реального времени.

### Что это такое?

Платформа позволяет оставить "цифровой след" в любом месте — отметить событие, встречу, находку или предупреждение, видимое другим пользователям поблизости:

- Спонтанные мероприятия и встречи
- Интересные места и находки
- Важные уведомления для района
- Точки для общения с людьми рядом

Каждая метка существует ограниченное время (от 12 часов до недели), что делает платформу живой и актуальной.

### Как это работает?

1. Откройте карту и посмотрите, что происходит вокруг
2. Создайте метку с фото, категорией и длительностью
3. Общайтесь через комментарии и реакции
4. Зарабатывайте опыт за активность
5. Получайте уведомления о новых метках в реальном времени

### Возможности платформы

| Модуль | Описание |
|--------|----------|
| **Метки** | Геолокационные метки с фото, категориями, автоудалением |
| **Пользователи** | Аутентификация, профили, настройки |
| **Real-time** | Мгновенные уведомления через WebSocket |
| **Комментарии** | Вложенные ответы и реакции |
| **Чат** | Личные сообщения между пользователями |
| **Gamification** | Опыт, уровни, награды за активность |
| **Подписки** | Премиум-возможности, интеграция с YooKassa |
| **Модерация** | Управление контентом, система банов |
| **Админ-панель** | Управление платформой |

---

## 2. Архитектура

Проект построен на **микросервисной архитектуре** с применением принципов **Domain-Driven Design (DDD)**.

```
┌─────────────────────────────────────────────────────────────┐
│                        Clients                               │
│                  (Mobile, Web, Admin)                        │
└──────────────────────────┬──────────────────────────────────┘
                           │ HTTP/WebSocket
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                      API Gateway                             │
│                    (Gin, REST API)                           │
└──────────────────────────┬──────────────────────────────────┘
                           │ gRPC
        ┌──────────────────┼──────────────────┐
        ▼                  ▼                  ▼
┌──────────────┐   ┌──────────────┐   ┌──────────────┐
│ user-service │   │marker-service│   │ chat-service │
└──────┬───────┘   └──────┬───────┘   └──────┬───────┘
       │                  │                  │
       ▼                  ▼                  ▼
┌──────────────┐   ┌──────────────┐   ┌──────────────┐
│  PostgreSQL  │   │PostgreSQL    │   │  PostgreSQL  │
│              │   │  + PostGIS   │   │              │
└──────────────┘   └──────────────┘   └──────────────┘
```

### Принципы

- **Независимость сервисов** — каждый микросервис имеет собственную БД и может деплоиться отдельно
- **gRPC для внутренней коммуникации** — типизированные контракты, высокая производительность
- **REST API для клиентов** — через API Gateway на Gin
- **Event-driven** — асинхронное взаимодействие через Redis Pub/Sub / Kafka
- **DDD** — чёткие границы bounded contexts

---

## 3. Технологический стек

| Категория | Технологии |
|-----------|------------|
| **Язык** | Go 1.23 |
| **HTTP Framework** | Gin |
| **Межсервисная связь** | gRPC + Protocol Buffers |
| **База данных** | PostgreSQL 16 + PostGIS |
| **Кеширование** | Redis 7 |
| **Очереди сообщений** | Redis Pub/Sub / Kafka |
| **Real-time** | WebSocket (gorilla/websocket) |
| **Контейнеризация** | Docker, Docker Compose |
| **Оркестрация** | Kubernetes (production) |

---

## 4. Структура репозитория

```
realtimemap-backend/
├── proto/                          # gRPC контракты
│   ├── user/
│   │   └── user.proto
│   ├── marker/
│   │   └── marker.proto
│   ├── geo/
│   │   └── geo.proto
│   ├── notification/
│   │   └── notification.proto
│   ├── comment/
│   │   └── comment.proto
│   ├── chat/
│   │   └── chat.proto
│   ├── gamification/
│   │   └── gamification.proto
│   ├── subscription/
│   │   └── subscription.proto
│   └── moderation/
│       └── moderation.proto
│
├── pkg/                            # Общие пакеты
│   ├── pb/                         # Сгенерированные protobuf файлы
│   ├── logger/                     # Структурированное логирование (zap)
│   ├── config/                     # Загрузка конфигурации
│   ├── database/                   # Подключение к PostgreSQL
│   ├── redis/                      # Redis клиент
│   ├── middleware/                 # Общие middleware (auth, logging, recovery)
│   └── errors/                     # Стандартизированные ошибки
│
├── services/
│   ├── gateway/                    # API Gateway (HTTP → gRPC)
│   ├── user-service/               # Пользователи, аутентификация
│   ├── marker-service/             # Метки на карте
│   ├── geo-service/                # Геопространственные запросы
│   ├── notification-service/       # Real-time уведомления
│   ├── comment-service/            # Комментарии и реакции
│   ├── chat-service/               # Личные сообщения
│   ├── gamification-service/       # Опыт, уровни, награды
│   ├── subscription-service/       # Подписки, платежи
│   └── moderation-service/         # Модерация контента
│
├── deployments/
│   ├── docker-compose.yml          # Локальный запуск
│   ├── docker-compose.dev.yml      # Разработка с hot reload
│   └── k8s/                        # Kubernetes манифесты
│
├── scripts/
│   ├── generate-proto.sh           # Генерация pb файлов
│   └── migrate.sh                  # Запуск миграций
│
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## 5. Микросервисы

### user-service

Управление пользователями и аутентификация.

- Регистрация и авторизация (JWT)
- Профили пользователей
- Настройки и предпочтения
- OAuth интеграции

### marker-service

Управление метками на карте.

- CRUD операции с метками
- Загрузка и хранение фото
- Категоризация
- Автоматическое удаление по TTL

### geo-service

Геопространственные операции.

- Поиск меток в радиусе
- Геохеширование для оптимизации
- Кластеризация меток
- Работа с PostGIS

### notification-service

Real-time уведомления.

- WebSocket соединения
- Push-уведомления
- Уведомления о новых метках поблизости
- Email-уведомления

### comment-service

Комментарии и реакции.

- Вложенные комментарии
- Реакции (лайки, эмодзи)
- Модерация комментариев

### chat-service

Личные сообщения.

- Диалоги между пользователями
- История сообщений
- Статусы прочтения

### gamification-service

Система геймификации.

- Начисление опыта
- Уровни пользователей
- Достижения и награды
- Лидерборды

### subscription-service

Подписки и платежи.

- Тарифные планы
- Интеграция с YooKassa
- Управление подписками
- История платежей

### moderation-service

Модерация контента.

- Жалобы на контент
- Система банов
- Автомодерация
- Админ-панель

---

## 6. Быстрый старт

### Требования

- [Docker](https://www.docker.com/get-started) и Docker Compose
- [Go 1.23+](https://go.dev/dl/) (для локальной разработки)
- [protoc](https://grpc.io/docs/protoc-installation/) (для генерации proto)
- [Make](https://www.gnu.org/software/make/)

### Настройка

1. **Клонируйте репозиторий**

```bash
git clone https://github.com/RealTimeMap/RealTimeMap-backend.git
cd RealTimeMap-backend
```

2. **Создайте файл конфигурации**

```bash
cp .env.example .env
```

3. **Настройте переменные окружения**

```env
# Database
DB_HOST=postgres
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_secure_password
DB_NAME=realtimemap

# Redis
REDIS_URL=redis://redis:6379

# JWT
JWT_SECRET=your_jwt_secret_key

# YooKassa (payments)
YOOKASSA_SHOP_ID=your_shop_id
YOOKASSA_SECRET_KEY=your_secret_key
```

> ⚠️ **Важно:** Никогда не коммитьте `.env` файл с реальными секретами в репозиторий.

4. **Сгенерируйте protobuf файлы**

```bash
make proto
```

### Запуск

```bash
# Запуск всех сервисов
docker-compose up -d --build

# Просмотр логов
docker-compose logs -f

# Остановка
docker-compose down
```

API Gateway будет доступен по адресу: **http://localhost:8080**

### Документация API

- **Swagger UI:** http://localhost:8080/swagger/index.html

---

## 7. Разработка

### Полезные команды

```bash
# Генерация protobuf
make proto

# Сборка всех сервисов
make build-all

# Запуск тестов
make test

# Линтер
make lint

# Миграции
make migrate-up
make migrate-down

# Запуск конкретного сервиса локально
cd services/user-service && go run cmd/main.go
```

### Добавление нового сервиса

1. Создайте proto-контракт в `proto/`
2. Сгенерируйте Go код: `make proto`
3. Создайте структуру сервиса в `services/`
4. Добавьте сервис в `docker-compose.yml`
5. Обновите API Gateway для проксирования запросов

Подробнее см. [services/README.md](services/README.md)

### Структура сервиса (DDD)

```
service-name/
├── cmd/
│   └── main.go
├── internal/
│   ├── domain/
│   │   ├── entity/
│   │   ├── valueobject/
│   │   ├── repository/
│   │   └── service/
│   ├── application/
│   │   ├── command/
│   │   ├── query/
│   │   └── dto/
│   ├── infrastructure/
│   │   ├── persistence/
│   │   └── grpc/
│   └── interfaces/
│       ├── http/
│       └── grpc/
├── migrations/
├── Dockerfile
└── Makefile
```

---

## Лицензия

MIT License. См. [LICENSE](LICENSE) для подробностей.