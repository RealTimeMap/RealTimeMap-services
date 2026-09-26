package postgres

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/services/smtp-service/internal/domain/email"
)

// Удаление аккаунта стирает письма на его адрес вместе с журналом событий и
// не задевает чужие. Адрес в событии может отличаться регистром от
// сохранённого.
func TestIntegrationDeleteByRecipient(t *testing.T) {
	repo, events, db, run := setupRepo(t)
	ctx := context.Background()
	now := time.Now().UTC()

	victim := "victim-" + run + "@example.com"
	other := "other-" + run + "@example.com"

	ids := map[string]string{}
	for i, to := range []string{victim, victim, other} {
		e := newEmail(run, "del-"+string(rune('a'+i)), 0, 0, now)
		e.ToEmail = to
		if _, err := repo.Create(ctx, e); err != nil {
			t.Fatalf("create: %v", err)
		}
		if err := events.Append(ctx, &email.Event{EmailID: e.ID, EventType: email.EventQueued, OccurredTime: now}); err != nil {
			t.Fatalf("append event: %v", err)
		}
		ids[e.ID.String()] = to
	}

	deleted, err := repo.DeleteByRecipient(ctx, strings.ToUpper(victim))
	if err != nil {
		t.Fatalf("DeleteByRecipient: %v", err)
	}
	if deleted != 2 {
		t.Fatalf("deleted = %d, want 2", deleted)
	}

	for id, to := range ids {
		var emails, evs int64
		db.Model(&email.Email{}).Where("id = ?", id).Count(&emails)
		db.Unscoped().Model(&email.Event{}).Where("email_id = ?", id).Count(&evs)

		want := int64(1)
		if to == victim {
			want = 0
		}
		if emails != want || evs != want {
			t.Errorf("%s (%s): emails=%d events=%d, want %d", id, to, emails, evs, want)
		}
	}

	// Повторная доставка события — не ошибка и ничего не удаляет.
	if deleted, err := repo.DeleteByRecipient(ctx, victim); err != nil || deleted != 0 {
		t.Fatalf("repeat: deleted=%d err=%v", deleted, err)
	}
}
