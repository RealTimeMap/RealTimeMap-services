package kafka

import (
	"context"
	"encoding/json"
	"testing"

	segmentio "github.com/segmentio/kafka-go"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka/events"
)

// Сквозной контракт: события собираются конструкторами, которыми их публикует
// social-service, и проходят через реальный разбор хендлера.
//
// Тест ловит рассинхрон сторон — переименованное поле или съехавший тип
// события. Без него ошибка вылезла бы только в проде молчащими пушами:
// хендлер не падает на незнакомом типе, он его пропускает.

func TestChatEventFromProducerIsUnderstoodByHandler(t *testing.T) {
	h, stub := newHandler()

	// Ровно то, что кладёт в топик ChatPublisher.
	event := events.NewChatMessageCreated(events.ChatMessagePayload{
		MessageID:    7,
		ChatID:       42,
		SenderID:     1,
		SenderName:   "Аня",
		Preview:      "привет",
		RecipientIDs: []uint{2, 3},
	})

	body, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal produced event: %v", err)
	}

	if err := h.HandleMessage(context.Background(), segmentio.Message{Value: body}); err != nil {
		t.Fatalf("handler rejected a valid produced event: %v", err)
	}

	if len(stub.cmds) != 2 {
		t.Fatalf("expected 2 notifications, got %d", len(stub.cmds))
	}
	if stub.cmds[0].Title != "Аня" {
		t.Fatalf("sender name did not survive the wire: %q", stub.cmds[0].Title)
	}
	if stub.cmds[0].Key.SourceID != 42 {
		t.Fatalf("chat id did not survive the wire: %d", stub.cmds[0].Key.SourceID)
	}
}

func TestSubscriptionEventFromProducerIsUnderstoodByHandler(t *testing.T) {
	h, stub := newHandler()

	event := events.NewSubscriptionCreated(events.SubscriptionPayload{
		SubscriberID:   2,
		TargetID:       5,
		SubscriberName: "Боря",
	})

	body, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal produced event: %v", err)
	}

	if err := h.HandleMessage(context.Background(), segmentio.Message{Value: body}); err != nil {
		t.Fatalf("handler rejected a valid produced event: %v", err)
	}

	if len(stub.cmds) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(stub.cmds))
	}
	if got := stub.cmds[0].Key.RecipientID; got != 5 {
		t.Fatalf("target id did not survive the wire: %d", got)
	}
}

// CommentPublisher в comment-service собирает payload этим конструктором —
// проверяем, что уведомление строится именно из него.
func TestCommentEventFromProducerIsUnderstoodByHandler(t *testing.T) {
	h, stub := newHandler()

	parentUserID := uint(5)
	event := events.NewCommentCreated(events.NewCommentPayload(
		11,            // commentID
		2,             // userID
		77,            // entityID
		entityMark,    // entityType
		"Боря",        // username
		ptr(uint(10)), // parentID
		&parentUserID, // parentUserID
		"согласен",    // content
	))

	body, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal produced event: %v", err)
	}

	if err := h.HandleMessage(context.Background(), segmentio.Message{Value: body}); err != nil {
		t.Fatalf("handler rejected a valid produced event: %v", err)
	}

	if len(stub.cmds) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(stub.cmds))
	}
	if got := stub.cmds[0].Key.RecipientID; got != parentUserID {
		t.Fatalf("parent author did not survive the wire: %d", got)
	}
}
