// Package kafka принимает доменные события других сервисов и ставит письма
// в очередь.
//
// Хендлер не отправляет писем сам: он делает INSERT и отдаёт управление, после
// чего consumer коммитит offset. Отправка внутри обработчика заблокировала бы
// партицию на время SMTP-диалога — сотни миллисекунд в норме и секунды
// таймаута при недоступности провайдера.
package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"
	userclient "github.com/RealTimeMap/RealTimeMap-backend/pkg/clients/user"
	pkgkafka "github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka/consumer"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka/events"
	"github.com/RealTimeMap/RealTimeMap-backend/services/smtp-service/internal/domain/email"
	segmentio "github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// Enqueuer ставит письмо в очередь. Интерфейс объявлен на стороне
// потребителя; *email.Service удовлетворяет ему напрямую.
type Enqueuer interface {
	Enqueue(ctx context.Context, in email.EnqueueInput) (*email.EnqueueResult, error)
}

// UserResolver отдаёт пользователя вместе с email.
//
// Нужен событиям, которые несут только идентификатор адресата: класть email в
// топик ради них нельзя — он живёт там по retention. Регистрация остаётся
// исключением: там адрес приходит в событии, потому что в момент отправки
// приветствия его больше неоткуда взять.
type UserResolver interface {
	GetUserByID(ctx context.Context, id int64) (*userclient.User, error)
}

type Handler struct {
	emails Enqueuer
	users  UserResolver

	// frontendURL — база для ссылок в письмах.
	frontendURL string

	logger *zap.Logger
}

func NewHandler(emails Enqueuer, users UserResolver, frontendURL string, logger *zap.Logger) *Handler {
	return &Handler{
		emails:      emails,
		users:       users,
		frontendURL: frontendURL,
		logger:      logger,
	}
}

// HandleMessage разбирает сообщение и направляет его обработчику по типу.
func (h *Handler) HandleMessage(ctx context.Context, msg segmentio.Message) error {
	// Тип события берётся из тела: заголовки может потерять промежуточный
	// компонент (mirror-maker, прокси), тело — нет.
	//
	// Читаются оба ключа: "type" — общий конверт, "event_type" — плоский
	// формат, в котором auth-сервис слал регистрацию до перехода на конверт.
	// Заголовок остаётся последним запасным вариантом.
	var envelope struct {
		Type      string `json:"type"`
		EventType string `json:"event_type"`
	}
	if err := json.Unmarshal(msg.Value, &envelope); err != nil {
		return consumer.Skip(fmt.Errorf("unmarshal envelope: %w", err))
	}

	eventType := envelope.Type
	if eventType == "" {
		eventType = envelope.EventType
	}
	if eventType == "" {
		eventType = pkgkafka.ExtractMeta(msg).EventType
	}

	switch eventType {
	case EventUserRegistered:
		return h.handleUserRegistered(ctx, msg)
	case EventCommentCreated:
		return h.handleCommentCreated(ctx, msg)
	case EventUserVerifyRequested:
		return h.handleVerifyRequested(ctx, msg)
	case EventUserPasswordForgotten:
		return h.handlePasswordForgotten(ctx, msg)
	case EventUserPasswordChanged:
		return h.handlePasswordChanged(ctx, msg)
	case EventUserLoggedIn:
		return h.handleLoggedIn(ctx, msg)
	default:
		// Топик может содержать события, до которых сервису нет дела.
		return nil
	}
}

func (h *Handler) handleUserRegistered(ctx context.Context, msg segmentio.Message) error {
	event, err := decodeUserRegistered(msg.Value)
	if err != nil {
		return consumer.Skip(err)
	}

	res, err := h.emails.Enqueue(ctx, email.EnqueueInput{
		TemplateID: "welcome",
		ToEmail:    event.Email,
		UserID:     &event.UserID,
		Data: map[string]any{
			"username":                event.Username,
			"mapUrl":                  h.frontendPath("/map"),
			"friendsUrl":              h.frontendPath("/profile/friends"),
			"notificationSettingsUrl": h.frontendPath("/settings/notifications"),
			"unsubscribeUrl":          h.frontendPath("/unsubscribe"),
		},
		// Явный ключ вместо хеша содержимого: смысл «одно приветственное
		// письмо на пользователя» не должен зависеть от набора полей события.
		// Хеш включил бы registered_at и рассыпался бы, начни продюсер слать
		// его иначе.
		IdempotencyKey: fmt.Sprintf("%s:%d", EventUserRegistered, event.UserID),
		TraceID:        traceID(msg),
	})
	if err != nil {
		return h.classifyEnqueueError(err, event)
	}

	if res.Duplicate {
		h.logger.Debug("welcome email already queued",
			zap.Uint64("user_id", event.UserID),
			zap.String("email_id", res.EmailID.String()),
		)
		return nil
	}

	h.logger.Info("welcome email queued",
		zap.Uint64("user_id", event.UserID),
		zap.String("email_id", res.EmailID.String()),
		zap.String("to", email.MaskEmail(event.Email)),
	)

	return nil
}

// classifyEnqueueError решает, коммитить offset или перечитать сообщение.
//
// Разница существенная: Skip коммитит и идёт дальше, Retryable оставляет
// offset на месте и останавливает партицию до восстановления. Ошибку данных
// нельзя объявлять retryable — партиция встанет навсегда.
func (h *Handler) classifyEnqueueError(err error, event UserRegistered) error {
	log := h.logger.With(
		zap.Uint64("user_id", event.UserID),
		zap.String("to", email.MaskEmail(event.Email)),
		zap.Error(err),
	)

	// Битый адрес, отсутствующий шаблон, нехватка данных — повтором не
	// лечатся: сообщение в топике не изменится.
	var domainErr apperror.DomainError
	if errors.As(err, &domainErr) && domainErr.HTTPStatus() < 500 {
		log.Warn("skipping user.registered: email cannot be built from this event")
		return consumer.Skip(err)
	}

	// Остальное — недоступность БД: сообщение перечитается.
	//
	// Retryable останавливает партицию до восстановления, поэтому им нельзя
	// помечать ошибки данных: топик встал бы навсегда на одном письме.
	log.Error("failed to queue welcome email, will retry")
	return consumer.Retryable(err)
}

// handleCommentCreated ставит письмо об ответе на комментарий.
//
// Письмо уходит только на ответ (parentId заполнен) и только автору
// родительского комментария. Комментарий верхнего уровня уведомлять некого:
// у сущности нет одного владельца, известного этому сервису.
func (h *Handler) handleCommentCreated(ctx context.Context, msg segmentio.Message) error {
	payload, err := decodeComment(msg.Value)
	if err != nil {
		return consumer.Skip(err)
	}

	// Не ответ — уведомлять некого.
	if payload.ParentID == nil || payload.ParentUserID == nil {
		return nil
	}

	// Ответ на собственный комментарий: письмо самому себе бессмысленно.
	if *payload.ParentUserID == payload.UserID {
		return nil
	}

	recipient, err := h.users.GetUserByID(ctx, int64(*payload.ParentUserID))
	if err != nil {
		return h.classifyLookupError(err, *payload.ParentUserID)
	}

	userID := uint64(*payload.ParentUserID)
	res, err := h.emails.Enqueue(ctx, email.EnqueueInput{
		TemplateID: "commentReply",
		ToEmail:    recipient.Email,
		UserID:     &userID,
		Data: map[string]any{
			"username":                recipient.Username,
			"authorName":              authorName(payload.Username),
			"commentText":             payload.Content,
			"commentUrl":              h.commentURL(payload.EntityID, payload.CommentID),
			"notificationSettingsUrl": h.frontendPath("/settings/notifications"),
		},
		// Ключ по комментарию-ответу: одно письмо на один ответ, независимо
		// от того, сколько раз событие приехало.
		IdempotencyKey: fmt.Sprintf("%s:%d", EventCommentCreated, payload.CommentID),
		TraceID:        traceID(msg),
	})
	if err != nil {
		return h.classifyEnqueueErrorFor(err, "comment reply", zap.Uint("comment_id", payload.CommentID))
	}

	if res.Duplicate {
		h.logger.Debug("comment reply email already queued", zap.Uint("comment_id", payload.CommentID))
		return nil
	}

	h.logger.Info("comment reply email queued",
		zap.Uint("comment_id", payload.CommentID),
		zap.String("to", email.MaskEmail(recipient.Email)),
	)
	return nil
}

// classifyLookupError решает, что делать при неудачном резолве адресата.
func (h *Handler) classifyLookupError(err error, userID uint) error {
	log := h.logger.With(zap.Uint("user_id", userID), zap.Error(err))

	// Пользователя нет — повтор не поможет: событие в топике не изменится.
	if errors.Is(err, userclient.ErrNotFound) {
		log.Warn("skipping comment event: recipient not found")
		return consumer.Skip(err)
	}

	// Сервис недоступен — перечитаем сообщение позже.
	log.Error("failed to resolve comment reply recipient, will retry")
	return consumer.Retryable(err)
}

// classifyEnqueueErrorFor — тот же разбор, что и для регистрации, но для
// произвольного события.
func (h *Handler) classifyEnqueueErrorFor(err error, what string, fields ...zap.Field) error {
	log := h.logger.With(append(fields, zap.Error(err))...)

	var domainErr apperror.DomainError
	if errors.As(err, &domainErr) && domainErr.HTTPStatus() < 500 {
		log.Warn("skipping " + what + ": email cannot be built from this event")
		return consumer.Skip(err)
	}

	log.Error("failed to queue " + what + ", will retry")
	return consumer.Retryable(err)
}

// authorName подставляет запасное имя, если продюсер его не прислал.
//
// Шаблон требует authorName непустым, и без подстановки письмо не ушло бы
// вовсе — а имя автора здесь не главное.
func authorName(username string) string {
	if strings.TrimSpace(username) == "" {
		return "Участник"
	}
	return username
}

// commentURL собирает ссылку на комментарий на фронтенде.
func (h *Handler) commentURL(entityID, commentID uint) string {
	return fmt.Sprintf("%s/marks/%d#comment-%d", strings.TrimRight(h.frontendURL, "/"), entityID, commentID)
}

// frontendPath приклеивает путь к базовому адресу фронтенда.
//
// Ссылки в письмах строятся здесь, а не зашиты в вёрстку: домен отличается
// между окружениями и меняется через конфиг, не через правку шаблонов.
func (h *Handler) frontendPath(path string) string {
	return strings.TrimRight(h.frontendURL, "/") + path
}

// --- письма аккаунта ---
//
// Все четыре события приходят от auth-сервиса и несут адрес получателя прямо
// в payload: письмо про смену пароля или вход должно уйти даже когда
// UserService недоступен, а токены всё равно генерирует auth.

func (h *Handler) handleVerifyRequested(ctx context.Context, msg segmentio.Message) error {
	event, err := decodePayload[events.UserVerifyRequestedPayload](msg.Value)
	if err != nil {
		return consumer.Skip(err)
	}

	return h.enqueueAccountEmail(ctx, msg, accountEmail{
		template:  "verifyEmail",
		eventType: EventUserVerifyRequested,
		userID:    event.UserID,
		toEmail:   event.Email,
		// Ключ включает ссылку: пользователь может запросить подтверждение
		// повторно, и второе письмо с новым токеном обязано уйти.
		key: fmt.Sprintf("%s:%d:%s", EventUserVerifyRequested, event.UserID, event.VerifyURL),
		data: map[string]any{
			"email":      event.Email,
			"code":       event.Code,
			"ttlMinutes": ttlOr(event.TTLMinutes, 30),
			"verifyUrl":  event.VerifyURL,
		},
	})
}

func (h *Handler) handlePasswordForgotten(ctx context.Context, msg segmentio.Message) error {
	event, err := decodePayload[events.UserPasswordForgottenPayload](msg.Value)
	if err != nil {
		return consumer.Skip(err)
	}

	ttl := ttlOr(event.TTLMinutes, 60)
	requested := orNow(event.RequestedAt)

	return h.enqueueAccountEmail(ctx, msg, accountEmail{
		template:  "passwordReset",
		eventType: EventUserPasswordForgotten,
		userID:    event.UserID,
		toEmail:   event.Email,
		// Ссылка входит в ключ: повторный запрос сброса должен дойти.
		key: fmt.Sprintf("%s:%d:%s", EventUserPasswordForgotten, event.UserID, event.ResetURL),
		data: map[string]any{
			"username":    event.Username,
			"resetUrl":    event.ResetURL,
			"ttlMinutes":  ttl,
			"requestedAt": formatMoment(requested),
			"device":      deviceOr(event.Device, event.IPAddress),
			"expiresAt":   formatTime(requested.Add(time.Duration(ttl) * time.Minute)),
			"securityUrl": h.frontendPath("/security"),
		},
	})
}

func (h *Handler) handlePasswordChanged(ctx context.Context, msg segmentio.Message) error {
	event, err := decodePayload[events.UserPasswordChangedPayload](msg.Value)
	if err != nil {
		return consumer.Skip(err)
	}

	changed := orNow(event.ChangedAt)

	return h.enqueueAccountEmail(ctx, msg, accountEmail{
		template:  "passwordChanged",
		eventType: EventUserPasswordChanged,
		userID:    event.UserID,
		toEmail:   event.Email,
		key:       fmt.Sprintf("%s:%d:%d", EventUserPasswordChanged, event.UserID, changed.Unix()),
		data: map[string]any{
			"username":    event.Username,
			"changedAt":   formatMoment(changed),
			"device":      deviceOr(event.Device, event.IPAddress),
			"ipAddress":   valueOr(event.IPAddress, "неизвестен"),
			"resetUrl":    h.frontendPath("/password/reset"),
			"securityUrl": h.frontendPath("/security"),
		},
	})
}

func (h *Handler) handleLoggedIn(ctx context.Context, msg segmentio.Message) error {
	event, err := decodePayload[events.UserLoggedInPayload](msg.Value)
	if err != nil {
		return consumer.Skip(err)
	}

	signedIn := orNow(event.SignedInAt)

	return h.enqueueAccountEmail(ctx, msg, accountEmail{
		template:  "newSignIn",
		eventType: EventUserLoggedIn,
		userID:    event.UserID,
		toEmail:   event.Email,
		// Ключ по моменту входа: каждый вход — отдельное письмо, но повторная
		// доставка того же события второго письма не создаёт.
		key: fmt.Sprintf("%s:%d:%d", EventUserLoggedIn, event.UserID, signedIn.Unix()),
		data: map[string]any{
			"username":    event.Username,
			"signedInAt":  formatMoment(signedIn),
			"device":      deviceOr(event.Device, event.IPAddress),
			"location":    valueOr(event.Location, "не определено"),
			"ipAddress":   valueOr(event.IPAddress, "неизвестен"),
			"sessionsUrl": h.frontendPath("/security/sessions"),
			"securityUrl": h.frontendPath("/security"),
		},
	})
}

// accountEmail — общая часть постановки письма аккаунта в очередь.
type accountEmail struct {
	template  string
	eventType string
	userID    uint64
	toEmail   string
	key       string
	data      map[string]any
}

// enqueueAccountEmail ставит письмо и разбирает ошибку одинаково для всех
// событий аккаунта: у них различаются только шаблон и данные.
func (h *Handler) enqueueAccountEmail(ctx context.Context, msg segmentio.Message, in accountEmail) error {
	if strings.TrimSpace(in.toEmail) == "" {
		// Без адреса письмо построить не из чего, и повтор не поможет.
		h.logger.Warn("skipping account email: event carries no recipient",
			zap.String("event_type", in.eventType),
			zap.Uint64("user_id", in.userID),
		)
		return consumer.Skip(errors.New("event carries no recipient address"))
	}

	res, err := h.emails.Enqueue(ctx, email.EnqueueInput{
		TemplateID:     in.template,
		ToEmail:        in.toEmail,
		UserID:         &in.userID,
		Data:           in.data,
		IdempotencyKey: in.key,
		TraceID:        traceID(msg),
	})
	if err != nil {
		return h.classifyEnqueueErrorFor(err, in.template,
			zap.String("event_type", in.eventType),
			zap.Uint64("user_id", in.userID),
		)
	}

	if res.Duplicate {
		h.logger.Debug("account email already queued",
			zap.String("template", in.template),
			zap.Uint64("user_id", in.userID),
		)
		return nil
	}

	h.logger.Info("account email queued",
		zap.String("template", in.template),
		zap.Uint64("user_id", in.userID),
		zap.String("to", email.MaskEmail(in.toEmail)),
	)
	return nil
}

// traceID достаёт идентификатор трассировки из заголовков сообщения.
func traceID(msg segmentio.Message) string {
	if id := pkgkafka.GetHeader(msg, "X-Trace-Id"); id != "" {
		return id
	}
	return pkgkafka.GetHeader(msg, "trace_id")
}

// orNow подставляет текущее время вместо нулевого: продюсер может не заполнить
// момент события, а письмо без даты бесполезно как сигнал безопасности.
func orNow(t time.Time) time.Time {
	if t.IsZero() {
		return time.Now().UTC()
	}
	return t
}

func ttlOr(v, fallback uint) uint {
	if v == 0 {
		return fallback
	}
	return v
}

func valueOr(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

// deviceOr описывает устройство, откатываясь на IP, когда User-Agent не
// разобран: шаблон требует поле непустым, а «неизвестно» пользователю
// бесполезно, если адрес всё же известен.
func deviceOr(device, ip string) string {
	if d := strings.TrimSpace(device); d != "" {
		return d
	}
	if ip = strings.TrimSpace(ip); ip != "" {
		return "устройство с адреса " + ip
	}
	return "неизвестное устройство"
}

// formatMoment и formatTime дают человекочитаемые дату и время в письме.
func formatMoment(t time.Time) string {
	return t.Format("02.01.2006, 15:04")
}

func formatTime(t time.Time) string {
	return t.Format("15:04")
}
