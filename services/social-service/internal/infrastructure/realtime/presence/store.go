// Package presence хранит онлайн-статусы пользователей в Redis.
//
// Онлайн — производная от факта наличия socket-соединения, а не отдельное поле в
// БД: пользователь онлайн, пока жив хотя бы один его сокет. Соединений может быть
// несколько (вкладки, устройства) и жить они могут на разных инстансах сервиса,
// поэтому состояние общее и лежит в Redis — том же, что использует Socket.IO
// adapter.
//
// Модель хранения: на пользователя один hash presence:conn:<userID>, поле = id
// сокета, значение = unix-время подключения. Онлайн = hash непустой.
//
// Почему hash с полями, а не счётчик INCR/DECR: счётчик не идемпотентен. Дубль
// disconnect уводит его в минус, а потерянный disconnect (kill -9 инстанса)
// навсегда оставляет пользователя «онлайн». С hash повторный HDEL того же поля —
// no-op, а осиротевшие поля убирает TTL ключа.
package presence

import (
	"context"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// keyPrefix — префикс ключей presence. Отдельный от префикса Socket.IO adapter
// (social:chat), чтобы не пересекаться с его служебными ключами.
const keyPrefix = "social:presence:conn:"

// connTTL — время жизни ключа соединений пользователя. Страховка от осиротевших
// записей: если инстанс упал не вызвав Disconnect, ключ протухнет сам и
// пользователь перестанет числиться онлайн. Продлевается на каждом Connect и
// через Refresh, поэтому живому соединению протухнуть не даёт.
const connTTL = 5 * time.Minute

// Store — presence-хранилище поверх Redis.
type Store struct {
	rdb *redis.Client
}

func NewStore(rdb *redis.Client) *Store {
	return &Store{rdb: rdb}
}

// Connect регистрирует соединение пользователя. Возвращает becameOnline=true,
// только если это первое живое соединение — то есть произошёл переход
// offline→online и об этом надо оповестить. Для второй вкладки вернётся false, и
// лишнего presence.online не будет.
//
// HSET возвращает число созданных полей, а HLEN после него даёт итоговое
// количество соединений: переход в онлайн = поле создано И оно единственное.
func (s *Store) Connect(ctx context.Context, userID uint, socketID string) (becameOnline bool, err error) {
	key := userKey(userID)

	pipe := s.rdb.TxPipeline()
	added := pipe.HSet(ctx, key, socketID, time.Now().Unix())
	total := pipe.HLen(ctx, key)
	pipe.Expire(ctx, key, connTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		return false, err
	}

	return added.Val() == 1 && total.Val() == 1, nil
}

// Disconnect снимает регистрацию соединения. Возвращает becameOffline=true,
// только если это было последнее живое соединение пользователя — переход
// online→offline. Идемпотентно: повторный вызов для того же сокета удалит 0 полей
// и вернёт false, так что дубль disconnect не породит второй presence.offline.
func (s *Store) Disconnect(ctx context.Context, userID uint, socketID string) (becameOffline bool, err error) {
	key := userKey(userID)

	pipe := s.rdb.TxPipeline()
	removed := pipe.HDel(ctx, key, socketID)
	left := pipe.HLen(ctx, key)
	if _, err := pipe.Exec(ctx); err != nil {
		return false, err
	}

	return removed.Val() == 1 && left.Val() == 0, nil
}

// Refresh продлевает TTL ключа соединений — вызывается периодически, пока сокет
// жив, иначе долгое соединение без реконнекта протухло бы по connTTL и
// пользователь пропал бы из онлайна, оставаясь подключённым.
func (s *Store) Refresh(ctx context.Context, userID uint) error {
	return s.rdb.Expire(ctx, userKey(userID), connTTL).Err()
}

// OnlineAmong фильтрует переданные id, оставляя только тех, кто сейчас онлайн.
// Нужен для presence.snapshot: подключившийся клиент должен сразу узнать, кто из
// его собеседников уже в сети, — по одним лишь событиям это не восстановить
// (события приходят только на будущие переходы).
//
// Проверяем пачкой через пайплайн: один RTT вместо N.
func (s *Store) OnlineAmong(ctx context.Context, userIDs []uint) ([]uint, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}

	pipe := s.rdb.Pipeline()
	cmds := make([]*redis.IntCmd, len(userIDs))
	for i, id := range userIDs {
		cmds[i] = pipe.HLen(ctx, userKey(id))
	}
	// redis.Nil здесь не ошибка: HLEN отсутствующего ключа возвращает 0. Любую
	// другую ошибку отдаём наверх — вызывающий трактует snapshot как best-effort.
	if _, err := pipe.Exec(ctx); err != nil && err != redis.Nil {
		return nil, err
	}

	online := make([]uint, 0, len(userIDs))
	for i, cmd := range cmds {
		if cmd.Val() > 0 {
			online = append(online, userIDs[i])
		}
	}
	return online, nil
}

func userKey(userID uint) string {
	return keyPrefix + strconv.FormatUint(uint64(userID), 10)
}
