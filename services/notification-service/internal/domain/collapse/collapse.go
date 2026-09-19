package collapse

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Kind — класс схлопываемых уведомлений. Входит в ключ, чтобы серии разных
// типов не смешивались: сообщения в чате и комментарии к метке считаются
// независимо, даже если совпали id.
type Kind string

const (
	KindChatMessage Kind = "chat"
	KindComment     Kind = "comment"
	KindSubscriber  Kind = "subscriber"
)

// Key адресует серию: кому шлём и про что.
//
// SourceID — идентификатор источника серии, а не события: чат для сообщений,
// метка для комментариев. Для подписчиков источника нет — там серия общая на
// получателя, и SourceID остаётся нулевым.
type Key struct {
	Kind        Kind
	RecipientID uint
	SourceID    uint
}

func (k Key) String() string {
	return fmt.Sprintf("%s:%d:%d", k.Kind, k.RecipientID, k.SourceID)
}

// Decision — что делать с событием.
type Decision struct {
	// Send — отправлять ли уведомление прямо сейчас.
	Send bool

	// Suppressed — сколько событий накопилось в окне помимо отправленного.
	// Заполняется только вместе с Send для сводки: 0 означает обычное
	// уведомление об одном событии.
	Suppressed int
}

// Limiter — состояние серий. Реализация общая на инстансы (Redis).
type Limiter interface {
	// Observe регистрирует событие по ключу и говорит, отправлять ли его.
	// Первый вызов в окне возвращает Send=true, последующие — Send=false,
	Observe(ctx context.Context, key Key, window time.Duration) (Decision, error)

	// Flush забирает накопленные пропуски по ключу и обнуляет счётчик.
	// Возвращает 0, если сводка не нужна — окно ещё идёт или в нём не было
	Flush(ctx context.Context, key Key) (int, error)

	// PendingKeys отдаёт ключи с истёкшим окном и непустым счётчиком —
	// те, по которым пора отправить сводку.
	PendingKeys(ctx context.Context, now time.Time, limit int) ([]Key, error)
}

func ParseKey(s string) (Key, error) {
	parts := strings.Split(s, ":")
	if len(parts) != 3 {
		return Key{}, fmt.Errorf("malformed collapse key %q", s)
	}
	r, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil {
		return Key{}, fmt.Errorf("malformed recipient in key %q: %w", s, err)
	}
	src, err := strconv.ParseUint(parts[2], 10, 64)
	if err != nil {
		return Key{}, fmt.Errorf("malformed source in key %q: %w", s, err)
	}

	return Key{Kind: Kind(parts[0]), RecipientID: uint(r), SourceID: uint(src)}, nil
}
