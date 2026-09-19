// Package redis реализует collapse.Limiter поверх Redis.
//
// Состояние серии — одна hash-запись на ключ:
//
//	notif:collapse:<kind>:<recipient>:<source>
//	  ├ n        сколько событий подавлено после отправленного
//	  └ exp      unix-время конца окна
//
// Плюс индекс notif:collapse:index — sorted set тех же ключей со score = exp.
// Без него сводку было бы не найти: TTL уносит запись молча, а KEYS/SCAN по
// всей базе на каждом тике обхода — линейный проход по чужим ключам.
//
// Операции идут в Lua: «прочитать счётчик, решить, записать» между двумя
// командами клиента разъезжается — два сообщения одного чата на разных подах
// одновременно увидели бы пустой ключ и оба ушли бы пушем.
package redis

import (
	"context"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/RealTimeMap/RealTimeMap-backend/services/notification-service/internal/domain/collapse"
)

const (
	keyPrefix = "notif:collapse:"
	indexKey  = "notif:collapse:index"

	// keyTTLSlack — насколько запись переживает своё окно.
	//
	// Ключ не должен исчезнуть ровно в момент истечения окна: сводку
	// отправляет обход, который приходит с задержкой до одного тика. Без
	// запаса накопленный счётчик пропал бы до того, как его прочитали, и
	// серия из десяти сообщений закончилась бы тишиной.
	keyTTLSlack = 10 * time.Minute
)

// observeScript регистрирует событие и решает, отправлять ли его.
//
// Возвращает {send, suppressed}: send=1 — окна не было, начинаем новое и
// отправляем сразу; send=0 — окно идёт, увеличиваем счётчик.
var observeScript = redis.NewScript(`
local key    = KEYS[1]
local index  = KEYS[2]
local now    = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local ttl    = tonumber(ARGV[3])
local member = ARGV[4]

local exp = tonumber(redis.call('HGET', key, 'exp'))

if exp == nil or exp <= now then
  -- Окна нет или оно истекло: открываем новое, событие уходит сразу.
  redis.call('HSET', key, 'n', 0, 'exp', now + window)
  redis.call('PEXPIRE', key, ttl)
  redis.call('ZADD', index, now + window, member)
  return {1, 0}
end

-- Окно активно: копим, ничего не отправляя.
local n = redis.call('HINCRBY', key, 'n', 1)
redis.call('PEXPIRE', key, ttl)
return {0, n}
`)

// flushScript забирает счётчик и закрывает серию.
//
// Окно, которое ещё идёт, не трогается: возвращается 0, запись остаётся
// копить дальше. Иначе обход, пришедший в середине окна, оборвал бы серию и
// следующее сообщение снова ушло бы отдельным пушем.
var flushScript = redis.NewScript(`
local key   = KEYS[1]
local index = KEYS[2]
local now   = tonumber(ARGV[1])
local member = ARGV[2]

local exp = tonumber(redis.call('HGET', key, 'exp'))
if exp == nil then
  redis.call('ZREM', index, member)
  return 0
end

if exp > now then
  return 0
end

local n = tonumber(redis.call('HGET', key, 'n')) or 0
redis.call('DEL', key)
redis.call('ZREM', index, member)
return n
`)

type Limiter struct {
	rdb *redis.Client
}

func NewLimiter(rdb *redis.Client) *Limiter {
	return &Limiter{rdb: rdb}
}

func (l *Limiter) Observe(ctx context.Context, key collapse.Key, window time.Duration) (collapse.Decision, error) {
	member := key.String()

	res, err := observeScript.Run(ctx, l.rdb,
		[]string{keyPrefix + member, indexKey},
		time.Now().UnixMilli(),
		window.Milliseconds(),
		(window + keyTTLSlack).Milliseconds(),
		member,
	).Int64Slice()
	if err != nil {
		return collapse.Decision{}, err
	}
	if len(res) != 2 {
		return collapse.Decision{}, nil
	}

	return collapse.Decision{Send: res[0] == 1, Suppressed: int(res[1])}, nil
}

func (l *Limiter) Flush(ctx context.Context, key collapse.Key) (int, error) {
	member := key.String()

	n, err := flushScript.Run(ctx, l.rdb,
		[]string{keyPrefix + member, indexKey},
		time.Now().UnixMilli(),
		member,
	).Int64()
	if err != nil {
		return 0, err
	}
	return int(n), nil
}

// PendingKeys отдаёт ключи, чьё окно уже закрылось.
//
// Выборка по score из индекса, а не SCAN по префиксу: истёкших ключей всегда
// сильно меньше, чем живых, и обход не должен платить за общее их число.
func (l *Limiter) PendingKeys(ctx context.Context, now time.Time, limit int) ([]collapse.Key, error) {
	members, err := l.rdb.ZRangeByScore(ctx, indexKey, &redis.ZRangeBy{
		Min:   "-inf",
		Max:   strconv.FormatInt(now.UnixMilli(), 10),
		Count: int64(limit),
	}).Result()
	if err != nil {
		return nil, err
	}

	keys := make([]collapse.Key, 0, len(members))
	for _, m := range members {
		key, err := collapse.ParseKey(m)
		if err != nil {
			// Мусор в индексе не должен останавливать обход: выкидываем
			// запись и идём дальше, иначе одна битая строка блокирует
			// сводки всем остальным.
			l.rdb.ZRem(ctx, indexKey, m)
			continue
		}
		keys = append(keys, key)
	}
	return keys, nil
}
