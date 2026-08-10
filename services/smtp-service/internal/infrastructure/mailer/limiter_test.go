package mailer

import (
	"sync"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

func TestLimiterAllowsWithinBurst(t *testing.T) {
	// Один слот в секунду, запас в три письма.
	drl := NewDomainRateLimiter(rate.Every(time.Second), 3)

	for i := 0; i < 3; i++ {
		if delay := drl.Reserve("example.com"); delay != 0 {
			t.Errorf("email %d delayed by %v, want immediate (within burst)", i, delay)
		}
	}
}

func TestLimiterDelaysBeyondBurst(t *testing.T) {
	drl := NewDomainRateLimiter(rate.Every(time.Second), 2)

	drl.Reserve("example.com")
	drl.Reserve("example.com")

	if delay := drl.Reserve("example.com"); delay <= 0 {
		t.Error("third email was allowed immediately, burst is not enforced")
	}
}

// Ключевое свойство: отложенное письмо не должно расходовать квоту.
//
// rate.Limiter резервирует слот безусловно, поэтому отклонённую бронь нужно
// отменять — иначе каждое откладывание тратило бы квоту впустую, и темп
// отправки просел бы кратно числу повторных проверок.
func TestLimiterDoesNotConsumeQuotaWhenPostponing(t *testing.T) {
	drl := NewDomainRateLimiter(rate.Every(100*time.Millisecond), 1)

	if delay := drl.Reserve("example.com"); delay != 0 {
		t.Fatalf("first email delayed by %v, want immediate", delay)
	}

	// Многократные проверки, пока слот недоступен: так ведёт себя воркер,
	// раз за разом подбирающий отложенное письмо из очереди.
	var last time.Duration
	for i := 0; i < 10; i++ {
		last = drl.Reserve("example.com")
		if last <= 0 {
			t.Fatalf("check %d unexpectedly allowed sending", i)
		}
	}

	// Задержка не должна нарастать: если бы брони копились, к десятой проверке
	// ожидание выросло бы до секунды вместо исходных 100 мс.
	if last > 300*time.Millisecond {
		t.Errorf("delay grew to %v after repeated checks — cancelled reservations are leaking quota", last)
	}
}

func TestLimiterSeparatesDomains(t *testing.T) {
	drl := NewDomainRateLimiter(rate.Every(time.Second), 1)

	if delay := drl.Reserve("example.com"); delay != 0 {
		t.Fatalf("first domain delayed by %v", delay)
	}
	// Лимит принадлежит домену получателя: перегрузка одного не должна
	// задерживать письма другим.
	if delay := drl.Reserve("another.com"); delay != 0 {
		t.Errorf("unrelated domain delayed by %v", delay)
	}
}

// Два воркера, впервые пишущие одному домену, не должны получить разные
// лимитеры: это удвоило бы фактический темп.
func TestLimiterCreatesSingleLimiterPerDomain(t *testing.T) {
	drl := NewDomainRateLimiter(rate.Every(time.Second), 1)

	const goroutines = 20
	limiters := make([]*rate.Limiter, goroutines)

	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			limiters[i] = drl.GetLimiter("example.com")
		}(i)
	}
	close(start)
	wg.Wait()

	for i, l := range limiters {
		if l != limiters[0] {
			t.Fatalf("goroutine %d got a different limiter for the same domain", i)
		}
	}
}
