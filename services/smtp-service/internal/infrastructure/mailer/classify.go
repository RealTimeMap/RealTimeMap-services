package mailer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"

	"github.com/RealTimeMap/RealTimeMap-backend/services/smtp-service/internal/domain/email"
	"github.com/wneessen/go-mail"
)

// classify относит ошибку отправки к временной или постоянной.
//
// От этого зависит, будет ли повторная попытка. Ошибка класса выбрана
// неверно — либо письмо теряется без нужды (временную сочли постоянной),
// либо сервис долбится в несуществующий адрес: повторные попытки доставки на
// такие адреса портят репутацию отправителя, и провайдер начинает резать весь
// трафик, включая валидный.
//
// Возвращает ошибку, обёрнутую в email.ErrTransient или email.ErrPermanent,
// сохраняя исходную для логов.
func classify(err error) error {
	if err == nil {
		return nil
	}

	// Сетевые отказы и таймауты — всегда временные: сервер недоступен сейчас,
	// но письмо валидно.
	//
	// Проверяется до разбора SMTP-кода, потому что go-mail определяет
	// временность по первому символу текста ошибки (см. isTempError), а у
	// "dial tcp: i/o timeout" там не '4' — библиотека сочла бы её постоянной.
	if isNetworkError(err) {
		return fmt.Errorf("%w: %v", email.ErrTransient, err)
	}

	var sendErr *mail.SendError
	if errors.As(err, &sendErr) {
		// RSET отправляется уже после того, как сервер принял письмо
		// (ответил на завершение DATA). Ошибка на этом шаге означает, что
		// доставка состоялась, а сорвалась лишь очистка сессии — обычно
		// потому, что сервер закрыл соединение сразу после приёма.
		//
		// Повторять такое письмо нельзя: адресат получит его дважды.
		// Проверяется до кода ответа, потому что код здесь относится к RSET,
		// а не к доставке.
		if sendErr.Reason == mail.ErrSMTPReset {
			return fmt.Errorf("%w: delivered but session cleanup failed: %v",
				email.ErrPermanent, err)
		}

		// Код ответа сервера — самый надёжный сигнал.
		// 4xx: «попробуй позже», 5xx: «никогда не получится».
		switch code := sendErr.ErrorCode(); {
		case code >= 400 && code < 500:
			return fmt.Errorf("%w: %v", email.ErrTransient, err)
		case code >= 500:
			return fmt.Errorf("%w: %v", email.ErrPermanent, err)
		}

		if sendErr.IsTemp() {
			return fmt.Errorf("%w: %v", email.ErrTransient, err)
		}

		// Код не распознан и временность не заявлена: разбираемся по причине.
		switch sendErr.Reason {
		case mail.ErrConnCheck, mail.ErrSMTPDataClose, mail.ErrWriteContent:
			// Проблемы самого соединения, а не письма. Доставка не
			// состоялась, повтор уместен.
			return fmt.Errorf("%w: %v", email.ErrTransient, err)
		case mail.ErrGetSender, mail.ErrGetRcpts, mail.ErrNoUnencoded:
			// Письмо составлено неправильно — повтор ничего не изменит.
			return fmt.Errorf("%w: %v", email.ErrPermanent, err)
		}

		// Причина неоднозначна. Считаем временной: неотправленное письмо
		// хуже лишней попытки, а потолок max_attempt всё равно ограничит.
		return fmt.Errorf("%w: %v", email.ErrTransient, err)
	}

	// Ошибка не от go-mail: конфигурация, TLS, аутентификация.
	if isPermanentConfigError(err) {
		return fmt.Errorf("%w: %v", email.ErrPermanent, err)
	}

	return fmt.Errorf("%w: %v", email.ErrTransient, err)
}

func isNetworkError(err error) bool {
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return true
	}

	if errors.Is(err, io.EOF) ||
		errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, net.ErrClosed) {
		return true
	}

	// Разрыв соединения приходит платформенно-зависимой ошибкой, единого
	// sentinel-значения для неё нет.
	text := strings.ToLower(err.Error())
	for _, marker := range []string{
		"connection reset",
		"connection refused",
		"broken pipe",
		"no such host",
		"i/o timeout",
		"network is unreachable",
		"tls handshake timeout",
	} {
		if strings.Contains(text, marker) {
			return true
		}
	}

	return false
}

// isPermanentConfigError распознаёт поломки настройки: ретраить их бесполезно,
// пока конфиг не поправят.
func isPermanentConfigError(err error) bool {
	text := strings.ToLower(err.Error())
	for _, marker := range []string{
		"authentication failed",
		"auth failed",
		"unsupported smtp auth",
		"certificate",
		"x509",
	} {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}
