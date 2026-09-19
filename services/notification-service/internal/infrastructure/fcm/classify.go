package fcm

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"firebase.google.com/go/v4/messaging"

	"github.com/RealTimeMap/RealTimeMap-backend/services/notification-service/internal/domain/notification"
)

// classify относит ошибку FCM к доменному классу.
//
// От класса зависит реакция воркера: повторить, бросить или ещё и отозвать
// токен. Ошибка отнесена неверно — либо уведомление теряется без нужды
// (временную сочли постоянной), либо сервис до упора ретраит отправку на
// токен, которого больше нет.
//
// Возвращает ошибку, обёрнутую в один из классов notification.Err*, сохраняя
// исходную для логов.
func classify(err error) error {
	if err == nil {
		return nil
	}

	switch {
	// Приложение удалено, переустановлено или токен обновился. Единственный
	// случай, когда мало бросить уведомление — токен надо отозвать.
	case messaging.IsUnregistered(err):
		return fmt.Errorf("%w: %v", notification.ErrTokenInvalid, err)

	// Испорченный токен или недопустимое поле в сообщении. Формально это
	// тоже негодный токен, но причина может быть и в самом сообщении,
	// поэтому отзывать его вслепую нельзя.
	case messaging.IsInvalidArgument(err):
		return fmt.Errorf("%w: FCM отверг запрос, испорченный токен или "+
			"недопустимое поле в сообщении: %v", notification.ErrPermanent, err)

	// Токен выдан другим Firebase-проектом, чем service account. Повтор не
	// поможет: либо клиент собран с чужим google-services.json, либо сервис
	// запущен с чужими кредами.
	case messaging.IsSenderIDMismatch(err):
		return fmt.Errorf("%w: токен выдан другим Firebase-проектом: %v",
			notification.ErrPermanent, err)

	// Квота проекта исчерпана — сама восстановится, но давить повторами
	// сейчас бессмысленно. Backoff очереди как раз и разведёт попытки.
	case messaging.IsQuotaExceeded(err):
		return fmt.Errorf("%w: квота FCM исчерпана: %v", notification.ErrTransient, err)

	// Сбой на стороне FCM.
	case messaging.IsUnavailable(err), messaging.IsInternal(err):
		return fmt.Errorf("%w: временный сбой FCM: %v", notification.ErrTransient, err)

	// FCM не смог авторизоваться в APNs: истёк или отозван .p8 ключ в
	// Firebase Console. До правки конфигурации ретраить бесполезно.
	case messaging.IsThirdPartyAuthError(err):
		return fmt.Errorf("%w: FCM не смог авторизоваться в APNs, проверьте "+
			".p8 ключ: %v", notification.ErrPermanent, err)

	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
		return fmt.Errorf("%w: запрос к FCM не уложился в таймаут: %v",
			notification.ErrTransient, err)
	}

	// Ошибки авторизации не покрыты хелперами messaging: они возникают до
	// вызова API, когда SDK меняет service account на OAuth2-токен. Самый
	// вероятный сбой при первом запуске, поэтому разбираем текст.
	if isCredentialsError(err) {
		return fmt.Errorf("%w: Google отверг service account — ключ удалён, "+
			"отозван или JSON от другого проекта: %v", notification.ErrPermanent, err)
	}

	// Класс не опознан. Считаем временной: недоставленное уведомление хуже
	// лишней попытки, а потолок max_attempt всё равно ограничит.
	return fmt.Errorf("%w: %v", notification.ErrTransient, err)
}

// isCredentialsError распознаёт отказ Google выдать OAuth2-токен по service
// account. Единого sentinel-значения для него нет — только текст.
func isCredentialsError(err error) bool {
	text := strings.ToLower(err.Error())
	for _, marker := range []string{
		"invalid_grant",
		"cannot fetch token",
		"could not find default credentials",
		"unauthorized_client",
	} {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}
