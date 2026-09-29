package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/logger"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka/events"
	segmentio "github.com/segmentio/kafka-go"
)

func bugMessage(t *testing.T, userID *uint) segmentio.Message {
	t.Helper()
	body, err := json.Marshal(events.NewBugCreated(events.BugCreatedPayload{
		BugID:      318,
		UserID:     userID,
		Title:      "Метка исчезает после смены масштаба",
		Desc:       "Поставила метку у кафе, приблизила карту — метка пропала.",
		Tag:        "ui",
		Platform:   "iPhone 15",
		OS:         "iOS 18.2",
		Resolution: "1179x2556",
		Build:      "1.0.4 (212)",
		CreatedAt:  time.Date(2026, 9, 26, 14, 8, 0, 0, time.UTC),
	}))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return segmentio.Message{Topic: "feedback-service.events", Value: body}
}

func TestBugCreatedRendersForEachRecipient(t *testing.T) {
	h, enq := newRenderingHandler(t, usersWith(7, "marina.k", "m@example.com"))
	h.WithBugReports([]string{"dev1@example.com", " dev2@example.com "}, "https://admin.example.com/")

	author := uint(7)
	if err := h.HandleMessage(context.Background(), bugMessage(t, &author)); err != nil {
		t.Fatalf("handle: %v", err)
	}

	if len(enq.rendered) != 2 {
		t.Fatalf("rendered %d emails, want 2", len(enq.rendered))
	}
	out := enq.rendered[0]
	for _, want := range []string{"BR-318", "Метка исчезает", "@marina.k", "iPhone 15 · iOS 18.2", "1179x2556", "ИНТЕРФЕЙС", "https://admin.example.com/bugs/318"} {
		if !strings.Contains(out.HTML, want) {
			t.Errorf("body has no %q", want)
		}
	}
	if !strings.Contains(out.Subject, "BR-318") {
		t.Errorf("subject %q has no bug code", out.Subject)
	}
}

func TestBugCreatedKeysAreUniquePerRecipient(t *testing.T) {
	enq := &recordingEnqueuer{}
	h := NewHandler(enq, &stubUsers{}, &recordingEraser{}, "https://realtimemap.ru", logger.NewNop()).
		WithBugReports([]string{"a@example.com", "b@example.com"}, "https://admin.example.com")

	if err := h.HandleMessage(context.Background(), bugMessage(t, nil)); err != nil {
		t.Fatalf("handle: %v", err)
	}

	if len(enq.calls) != 2 {
		t.Fatalf("queued %d emails, want 2", len(enq.calls))
	}
	if enq.calls[0].IdempotencyKey == enq.calls[1].IdempotencyKey {
		t.Error("recipients share an idempotency key: the second email would be dropped as duplicate")
	}
	if got := enq.calls[0].Data["author"]; got != "Аноним" {
		t.Errorf("author = %v, want Аноним", got)
	}
}

func TestBugCreatedWithoutRecipientsIsIgnored(t *testing.T) {
	enq := &recordingEnqueuer{}
	h := NewHandler(enq, &stubUsers{}, &recordingEraser{}, "https://realtimemap.ru", logger.NewNop())

	if err := h.HandleMessage(context.Background(), bugMessage(t, nil)); err != nil {
		t.Fatalf("handle: %v", err)
	}
	if len(enq.calls) != 0 {
		t.Errorf("queued %d emails without configured recipients", len(enq.calls))
	}
}

// Недоступный UserService не должен останавливать письмо.
func TestBugCreatedSurvivesUserLookupFailure(t *testing.T) {
	h, enq := newRenderingHandler(t, &stubUsers{err: errors.New("unavailable")})
	h.WithBugReports([]string{"dev@example.com"}, "https://admin.example.com")

	author := uint(7)
	if err := h.HandleMessage(context.Background(), bugMessage(t, &author)); err != nil {
		t.Fatalf("handle: %v", err)
	}
	if len(enq.rendered) != 1 || !strings.Contains(enq.rendered[0].HTML, "Пользователь #7") {
		t.Error("email not rendered with fallback author")
	}
}
