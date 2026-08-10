package app

import (
	"fmt"

	"github.com/RealTimeMap/RealTimeMap-backend/services/smtp-service/internal/config"
	"github.com/RealTimeMap/RealTimeMap-backend/services/smtp-service/internal/domain/email"
	domaintemplate "github.com/RealTimeMap/RealTimeMap-backend/services/smtp-service/internal/domain/template"
	"github.com/RealTimeMap/RealTimeMap-backend/services/smtp-service/internal/infrastructure/mailer"
	"github.com/RealTimeMap/RealTimeMap-backend/services/smtp-service/internal/infrastructure/persistence/postgres"
	embedtemplate "github.com/RealTimeMap/RealTimeMap-backend/services/smtp-service/internal/infrastructure/template"
	"github.com/RealTimeMap/RealTimeMap-backend/services/smtp-service/internal/worker"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
	"gorm.io/gorm"
)

// Container держит собранные зависимости сервиса.
//
// Здесь же расположены точки подмены реализаций при переходе к фазе 2:
// шаблоны переезжают из embed.FS в Postgres, а адрес получателя начинает
// резолвиться по gRPC вместо чтения из события. Ни очередь, ни воркеры при
// этом не меняются.
type Container struct {
	Config *config.Config
	Logger *zap.Logger
	DB     *gorm.DB

	Emails   email.Repository
	Events   email.EventRepository
	Renderer *domaintemplate.Renderer

	// Emailer — единственный вход в очередь: и Kafka, и HTTP ставят письма
	// через него, поэтому валидация и дедупликация работают одинаково.
	Emailer *email.Service

	// NewSender создаёт отправителя. Именно фабрика, а не готовый экземпляр:
	// клиент go-mail сериализует отправку внутренним мьютексом, поэтому общий
	// sender превратил бы пул воркеров в очередь из одного. Каждый воркер
	// получает собственное соединение.
	NewSender func() (email.Sender, error)

	// Workers и Reaper реализуют runner.Server и останавливаются вместе с
	// остальными серверами по общему сигналу.
	Workers *worker.Pool
	Reaper  *worker.Reaper
}

func NewContainer(cfg *config.Config, db *gorm.DB, logger *zap.Logger) (*Container, error) {
	// Точка подмены для фазы 2: pgProvider поверх таблицы шаблонов.
	templates, err := embedtemplate.NewProvider()
	if err != nil {
		return nil, fmt.Errorf("load email templates: %w", err)
	}

	// Точка подмены для фазы 2: sender поверх API SES/SendGrid.
	mailerCfg := mailer.Config{
		Host:          cfg.SMTP.Host,
		Port:          cfg.SMTP.Port,
		UseSSL:        cfg.SMTP.UseSSL,
		Username:      cfg.SMTP.Username,
		Password:      cfg.SMTP.Password,
		From:          cfg.SMTP.From,
		FromName:      cfg.SMTP.FromName,
		AllowInsecure: cfg.SMTP.AllowInsecure,
		Timeout:       cfg.SMTP.Timeout,
		IdleTimeout:   cfg.SMTP.IdleTimeout,
	}

	// Проверяем настройки сразу: ошибка конфигурации должна валить старт, а не
	// всплывать на первом письме.
	if _, err := mailer.NewSender(mailerCfg, logger); err != nil {
		return nil, fmt.Errorf("configure smtp sender: %w", err)
	}

	emails := postgres.NewPgEmailRepository(db, logger)
	events := postgres.NewPgEmailEventRepository(db, logger)
	renderer := domaintemplate.NewRenderer(templates)

	newSender := func() (email.Sender, error) { return mailer.NewSender(mailerCfg, logger) }

	// Лимитер по домену получателя: крупные почтовые службы отвечают
	// временными отказами, если один отправитель льёт им письма подряд.
	var limiter worker.RateLimiter
	if cfg.Worker.DomainRate > 0 {
		limiter = mailer.NewDomainRateLimiter(rate.Limit(cfg.Worker.DomainRate), cfg.Worker.DomainBurst)
	}

	workers := worker.NewPool(worker.Config{
		Count:        cfg.Worker.Count,
		ClaimBatch:   cfg.Worker.ClaimBatch,
		PollInterval: cfg.Worker.PollInterval,
		LockTimeout:  cfg.Worker.LockTimeout,
		Backoff:      cfg.Worker.Backoff,
		DailyLimit:   cfg.Worker.DailyLimit,
	}, emails, events, limiter, newSender, logger)

	return &Container{
		Config:    cfg,
		Logger:    logger,
		DB:        db,
		Emails:    emails,
		Events:    events,
		Renderer:  renderer,
		Emailer:   email.NewService(emails, events, renderer, cfg.Worker.MaxAttempt, logger),
		NewSender: newSender,
		Workers:   workers,
		Reaper:    worker.NewReaper(cfg.Worker.ReaperInterval, emails, logger),
	}, nil
}
