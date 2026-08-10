package mailer

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// DomainRateLimiter ограничивает темп отправки в разрезе домена получателя.
type DomainRateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex

	defaultLimit rate.Limit
	defaultBurst int
}

func NewDomainRateLimiter(defaultLimit rate.Limit, defaultBurst int) *DomainRateLimiter {
	return &DomainRateLimiter{
		limiters:     make(map[string]*rate.Limiter),
		defaultLimit: defaultLimit,
		defaultBurst: defaultBurst,
	}
}

func (drl *DomainRateLimiter) GetLimiter(domain string) *rate.Limiter {
	drl.mu.RLock()
	limiter, exists := drl.limiters[domain]
	drl.mu.RUnlock()
	if exists {
		return limiter
	}

	drl.mu.Lock()
	defer drl.mu.Unlock()

	// Проверяем повторно: между освобождением RLock и захватом Lock лимитер
	// мог создать другой воркер, а два лимитера на домен удвоили бы темп.
	if limiter, exists = drl.limiters[domain]; exists {
		return limiter
	}

	limiter = rate.NewLimiter(drl.defaultLimit, drl.defaultBurst)
	drl.limiters[domain] = limiter

	return limiter
}

// Reserve возвращает, сколько ждать перед отправкой на домен.
func (drl *DomainRateLimiter) Reserve(domain string) time.Duration {
	reservation := drl.GetLimiter(domain).Reserve()

	// Лимитер не может выдать слот в принципе (нулевой burst).
	if !reservation.OK() {
		return 0
	}

	delay := reservation.Delay()
	if delay > 0 {
		reservation.Cancel()
	}

	return delay
}
