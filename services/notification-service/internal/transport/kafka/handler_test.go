package kafka

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	segmentio "github.com/segmentio/kafka-go"
	"go.uber.org/zap"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka/events"
	"github.com/RealTimeMap/RealTimeMap-backend/services/notification-service/internal/app/use_cases/notify"
	"github.com/RealTimeMap/RealTimeMap-backend/services/notification-service/internal/domain/collapse"
)

// notifierStub запоминает команды вместо отправки.
type notifierStub struct {
	cmds []notify.NotifyEventCommand
	err  error
}

func (s *notifierStub) Handle(_ context.Context, cmd notify.NotifyEventCommand) error {
	s.cmds = append(s.cmds, cmd)
	return s.err
}

func message(t *testing.T, eventType string, payload any) segmentio.Message {
	t.Helper()

	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	body, err := json.Marshal(events.RawEvent{
		Envelop: events.NewEnvelop(eventType),
		Payload: raw,
	})
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}

	return segmentio.Message{Topic: "test", Value: body}
}

func newHandler() (*Handler, *notifierStub) {
	stub := &notifierStub{}
	return NewHandler(stub, &stubEraser{}, zap.NewNop()), stub
}

func TestChatMessageNotifiesEveryRecipientExceptSender(t *testing.T) {
	h, stub := newHandler()

	msg := message(t, events.ChatMessageCreated, events.ChatMessagePayload{
		MessageID:    7,
		ChatID:       42,
		SenderID:     1,
		SenderName:   "Аня",
		Preview:      "привет",
		RecipientIDs: []uint{1, 2, 3},
	})

	if err := h.HandleMessage(context.Background(), msg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(stub.cmds) != 2 {
		t.Fatalf("expected 2 notifications, got %d", len(stub.cmds))
	}

	for _, cmd := range stub.cmds {
		if cmd.Key.RecipientID == 1 {
			t.Fatal("sender must not be notified about own message")
		}
		// Серия считается по чату: иначе каждое сообщение открывало бы своё
		// окно и схлопывание не срабатывало бы вовсе.
		if cmd.Key.SourceID != 42 || cmd.Key.Kind != collapse.KindChatMessage {
			t.Fatalf("unexpected collapse key: %+v", cmd.Key)
		}
		if cmd.Title != "Аня" {
			t.Fatalf("unexpected title: %q", cmd.Title)
		}
	}
}

func TestGroupChatUsesChatTitleAndPrefixesSender(t *testing.T) {
	h, stub := newHandler()

	msg := message(t, events.ChatMessageCreated, events.ChatMessagePayload{
		ChatID:       9,
		SenderID:     1,
		SenderName:   "Аня",
		ChatTitle:    "Поход",
		IsGroup:      true,
		Preview:      "выходим в семь",
		RecipientIDs: []uint{2},
	})

	if err := h.HandleMessage(context.Background(), msg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(stub.cmds) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(stub.cmds))
	}
	cmd := stub.cmds[0]
	if cmd.Title != "Поход" {
		t.Fatalf("group title must be the chat name, got %q", cmd.Title)
	}
	if cmd.Content != "Аня: выходим в семь" {
		t.Fatalf("group body must name the sender, got %q", cmd.Content)
	}
}

func TestCommentNotifiesParentAuthor(t *testing.T) {
	h, stub := newHandler()

	parent := uint(5)
	msg := message(t, events.CommentCreated, events.CommentPayload{
		CommentID:    11,
		UserID:       2,
		Username:     "Боря",
		EntityType:   entityMark,
		EntityID:     77,
		ParentUserID: &parent,
		Content:      "согласен",
	})

	if err := h.HandleMessage(context.Background(), msg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(stub.cmds) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(stub.cmds))
	}
	cmd := stub.cmds[0]
	if cmd.Key.RecipientID != parent {
		t.Fatalf("expected recipient %d, got %d", parent, cmd.Key.RecipientID)
	}
	// Серия по метке, а не по комментарию: десять ответов под одной меткой
	// должны схлопнуться в одну серию.
	if cmd.Key.SourceID != 77 {
		t.Fatalf("collapse must be keyed by mark, got source %d", cmd.Key.SourceID)
	}
}

func TestCommentSkippedWhenNoRecipient(t *testing.T) {
	cases := []struct {
		name    string
		payload events.CommentPayload
	}{
		{
			// Комментарий верхнего уровня: владелец метки в событии не едет,
			// уведомлять некого.
			name: "top level comment",
			payload: events.CommentPayload{
				UserID: 2, EntityType: entityMark, EntityID: 77,
			},
		},
		{
			name: "reply to self",
			payload: events.CommentPayload{
				UserID: 2, EntityType: entityMark, EntityID: 77,
				ParentUserID: ptr(uint(2)),
			},
		},
		{
			name: "comment on another entity",
			payload: events.CommentPayload{
				UserID: 2, EntityType: "post", EntityID: 77,
				ParentUserID: ptr(uint(5)),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h, stub := newHandler()

			if err := h.HandleMessage(context.Background(), message(t, events.CommentCreated, tc.payload)); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(stub.cmds) != 0 {
				t.Fatalf("expected no notifications, got %d", len(stub.cmds))
			}
		})
	}
}

func TestSubscriptionNotifiesTarget(t *testing.T) {
	h, stub := newHandler()

	msg := message(t, events.SubscriptionCreated, events.SubscriptionPayload{
		SubscriberID:   2,
		TargetID:       5,
		SubscriberName: "Боря",
	})

	if err := h.HandleMessage(context.Background(), msg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(stub.cmds) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(stub.cmds))
	}
	cmd := stub.cmds[0]
	if cmd.Key.RecipientID != 5 {
		t.Fatalf("expected target to be notified, got %d", cmd.Key.RecipientID)
	}
	// Подписчики схлопываются в общую серию на получателя.
	if cmd.Key.SourceID != 0 || cmd.Key.Kind != collapse.KindSubscriber {
		t.Fatalf("unexpected collapse key: %+v", cmd.Key)
	}
	if !strings.Contains(cmd.Content, "Боря") {
		t.Fatalf("expected subscriber name in body, got %q", cmd.Content)
	}
}

func TestUnknownEventTypeIsIgnored(t *testing.T) {
	h, stub := newHandler()

	msg := message(t, events.ProfileUpdated, events.ProfilePayload{UserID: 1})

	if err := h.HandleMessage(context.Background(), msg); err != nil {
		t.Fatalf("unknown events must be skipped silently, got %v", err)
	}
	if len(stub.cmds) != 0 {
		t.Fatalf("expected no notifications, got %d", len(stub.cmds))
	}
}

func TestMalformedBodyIsSkippedNotRetried(t *testing.T) {
	h, _ := newHandler()

	err := h.HandleMessage(context.Background(), segmentio.Message{Value: []byte("{oops")})
	if err == nil {
		t.Fatal("expected an error for malformed body")
	}
	// Skip, а не Retryable: повтор разберёт тело ровно так же и заклинит
	// партицию.
	if !strings.Contains(err.Error(), "skip") {
		t.Fatalf("expected skip verdict, got %v", err)
	}
}

func TestPreviewTruncatesOnWordBoundary(t *testing.T) {
	long := strings.Repeat("слово ", 40)

	got := preview(long, 20)

	if !strings.HasSuffix(got, "…") {
		t.Fatalf("expected ellipsis, got %q", got)
	}
	if len([]rune(got)) > 21 {
		t.Fatalf("expected at most 21 runes, got %d (%q)", len([]rune(got)), got)
	}
}

func ptr[T any](v T) *T { return &v }
