# Services

Директория содержит независимые микросервисы. Каждый сервис — изолированный bounded context с собственной бизнес-логикой, базой данных и API.

## Архитектура

Все сервисы следуют принципам **Domain-Driven Design (DDD)** и **Clean Architecture**.

### Структура сервиса

```
service-name/
├── cmd/
│   └── main.go                 # Точка входа, DI, запуск серверов
├── internal/
│   ├── domain/                 # Ядро домена (не зависит ни от чего)
│   │   ├── entity/             # Сущности с бизнес-логикой
│   │   ├── valueobject/        # Value Objects
│   │   ├── repository/         # Интерфейсы репозиториев
│   │   ├── service/            # Доменные сервисы
│   │   └── event/              # Доменные события
│   ├── application/            # Use Cases / Application Services
│   │   ├── command/            # Команды (изменение состояния)
│   │   ├── query/              # Запросы (чтение данных)
│   │   └── dto/                # Data Transfer Objects
│   ├── infrastructure/         # Внешние зависимости
│   │   ├── persistence/        # Реализация репозиториев (PostgreSQL)
│   │   ├── grpc/               # gRPC клиенты к другим сервисам
│   │   └── messaging/          # Kafka, RabbitMQ (если нужно)
│   ├── transport/             # Входные точки
│   │   ├── http/               # Gin handlers
│   │   │   ├── handler/
│   │   │   ├── router/
│   │   │   └── middleware/
│   │   └── grpc/               # gRPC server handlers
│   └── config/
│       └── config.go
├── migrations/
├── config.yaml
├── Dockerfile
└── Makefile
```

### Слои и зависимости

```
┌─────────────────────────────────────────────────────┐
│                    Transport                        │
│              (HTTP, gRPC handlers)                   │
└──────────────────────┬──────────────────────────────┘
                       │ вызывает
                       ▼
┌─────────────────────────────────────────────────────┐
│                   Application                        │
│            (Use Cases, Commands, Queries)            │
└──────────────────────┬──────────────────────────────┘
                       │ использует
                       ▼
┌─────────────────────────────────────────────────────┐
│                     Domain                           │
│     (Entities, Value Objects, Domain Services)       │
└──────────────────────┬──────────────────────────────┘
                       │ абстракции реализует
                       ▼
┌─────────────────────────────────────────────────────┐
│                 Infrastructure                       │
│        (PostgreSQL, gRPC clients, Messaging)         │
└─────────────────────────────────────────────────────┘
```

**Правило зависимостей:** внутренние слои не знают о внешних. Domain не импортирует ничего кроме стандартной библиотеки.

## Требования к коду

### Конфигурация и секреты

**ЗАПРЕЩЕНО:**
```go
// Никогда так не делай
db, _ := sql.Open("postgres", "host=localhost user=admin password=secret123 dbname=application")
apiKey := "sk-live-abc123"
```

**ОБЯЗАТЕЛЬНО:**
```go
// Все чувствительные данные из переменных окружения
type Config struct {
    Database DatabaseConfig `mapstructure:"database"`
}

type DatabaseConfig struct {
    Host     string `mapstructure:"host" env:"DB_HOST"`
    Password string `mapstructure:"password" env:"DB_PASSWORD"`
}
```

Секреты передаются через:
- Переменные окружения
- Docker secrets
- Kubernetes Secrets / ConfigMaps
- Vault (для production)

### Переиспользование кода

**Общие пакеты** размещаются в `pkg/`:
```go
import (
    "github.com/company/monorepo/pkg/logger"
    "github.com/company/monorepo/pkg/database"
    "github.com/company/monorepo/pkg/middleware"
    "github.com/company/monorepo/pkg/pb/user"
)
```

**Доменная логика** не дублируется между сервисами. Если два сервиса нуждаются в одной логике — это сигнал пересмотреть границы bounded contexts.

### Интерфейсы и Dependency Injection

Все зависимости инжектятся через интерфейсы:

```go
// domain/repository/user.go
type UserRepository interface {
    Save(ctx context.Context, user *entity.User) error
    FindByID(ctx context.Context, id uuid.UUID) (*entity.User, error)
}

// application/command/create_user.go
type CreateUserHandler struct {
    repo   repository.UserRepository  // интерфейс, не реализация
    hasher PasswordHasher
}
```

### Обработка ошибок

Доменные ошибки определяются в domain слое:

```go
// domain/entity/validation.go
var (
    ErrUserNotFound      = errors.New("user not found")
    ErrEmailAlreadyTaken = errors.New("email already taken")
    ErrInvalidEmail      = errors.New("invalid email format")
)
```

Инфраструктурные ошибки оборачиваются:

```go
// infrastructure/persistence/user.go
func (r *userRepo) FindByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
    var user entity.User
    err := r.db.GetContext(ctx, &user, query, id)
    if errors.Is(err, sql.ErrNoRows) {
        return nil, entity.ErrUserNotFound  // маппинг на доменную ошибку
    }
    if err != nil {
        return nil, fmt.Errorf("failed to find user: %w", err)
    }
    return &user, nil
}
```

### Валидация

**Value Objects** инкапсулируют валидацию:

```go
// domain/valueobject/email.go
type Email struct {
    value string
}

func NewEmail(value string) (Email, error) {
    if !isValidEmail(value) {
        return Email{}, ErrInvalidEmail
    }
    return Email{value: strings.ToLower(value)}, nil
}

func (e Email) String() string { return e.value }
```

**DTO валидация** на уровне interfaces:

```go
// interfaces/http/handler/dto.go
type CreateUserRequest struct {
    Email    string `json:"email" binding:"required,email"`
    Name     string `json:"name" binding:"required,min=2,max=100"`
    Password string `json:"password" binding:"required,min=8"`
}
```

### Логирование

Используй структурированный логгер из `pkg/logger`:

```go
logger.Info("user created",
    "user_id", user.ID,
    "email", user.Email,
)

logger.Error("failed to create user",
    "error", err,
    "email", input.Email,
)
```

**ЗАПРЕЩЕНО** логировать:
- Пароли и токены
- Полные номера карт
- Персональные данные (в production)

### Тестирование

```
service-name/
├── internal/
│   ├── domain/
│   │   └── entity/
│   │       ├── user.go
│   │       └── user_test.go        # Unit тесты домена
│   ├── application/
│   │   └── command/
│   │       ├── create_user.go
│   │       └── create_user_test.go # Unit тесты с моками
│   └── infrastructure/
│       └── persistence/
│           ├── user.go
│           └── user_test.go        # Integration тесты
└── tests/
    └── e2e/                        # End-to-end тесты
```

Моки генерируются через `mockgen`:
```bash
mockgen -source=internal/domain/repository/user.go -destination=internal/mocks/user_repository.go
```

## Взаимодействие между сервисами

### gRPC

Сервисы общаются **только** через gRPC. HTTP API — для внешних клиентов.

```go
// infrastructure/grpc/user_client.go
type UserClient struct {
    client pb.UserServiceClient
}

func NewUserClient(conn *grpc.ClientConn) *UserClient {
    return &UserClient{client: pb.NewUserServiceClient(conn)}
}

func (c *UserClient) GetUser(ctx context.Context, id int64) (*entity.User, error) {
    resp, err := c.client.GetUser(ctx, &pb.GetUserRequest{Id: id})
    if err != nil {
        return nil, fmt.Errorf("grpc get user: %w", err)
    }
    return mapProtoToEntity(resp.User), nil
}
```

### Паттерн Anti-Corruption Layer

При интеграции с другим сервисом создавай ACL:

```go
// infrastructure/grpc/user_acl.go
type UserACL struct {
    client *UserClient
}

// Преобразует внешнюю модель в доменную
func (a *UserACL) GetUserInfo(ctx context.Context, id int64) (*valueobject.UserInfo, error) {
    user, err := a.client.GetUser(ctx, id)
    if err != nil {
        return nil, err
    }
    return valueobject.NewUserInfo(user.Name, user.Email)
}
```

## Чеклист перед созданием сервиса

- [ ] Определены границы bounded context
- [ ] Описаны доменные сущности и value objects
- [ ] Определены use cases (commands/queries)
- [ ] Создан proto-файл в `/proto/{service}/`
- [ ] Сгенерированы pb файлы: `make proto`
- [ ] Написаны миграции БД
- [ ] Настроен Dockerfile
- [ ] Добавлен сервис в docker-compose.yml
- [ ] Написаны unit тесты для domain слоя
- [ ] Написаны integration тесты для persistence
- [ ] Документирован API (Swagger/OpenAPI)

## Команды

```bash
# Из директории сервиса
make build          # Сборка бинарника
make test           # Запуск тестов
make lint           # Линтер
make migrate-up     # Применить миграции
make migrate-down   # Откатить миграции

# Из корня монорепо
make proto          # Генерация всех proto
make build-all      # Сборка всех сервисов
make up             # Запуск через docker-compose
```

## Добавление нового сервиса

1. Создай структуру директорий:
```bash
mkdir -p services/new-service/{cmd,internal/{domain/{entity,valueobject,repository,service},application/{command,query,dto},infrastructure/{persistence,grpc},interfaces/{http/handler,grpc},config},migrations}
```

2. Добавь proto-контракт в `/proto/new-service/`

3. Сгенерируй pb файлы:
```bash
make proto
```

4. Реализуй слои снизу вверх:
    - Domain (entities, value objects, repository interfaces)
    - Application (use cases)
    - Infrastructure (repository implementations, grpc clients)
    - Interfaces (http/grpc handlers)

5. Добавь сервис в `docker-compose.yml`

6. Документируй API