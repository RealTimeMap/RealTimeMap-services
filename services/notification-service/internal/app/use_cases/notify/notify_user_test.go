package notify

import (
	"context"
	"testing"

	"go.uber.org/zap"

	"github.com/RealTimeMap/RealTimeMap-backend/services/notification-service/internal/domain/collapse"
	"github.com/RealTimeMap/RealTimeMap-backend/services/notification-service/internal/domain/notification"
	"github.com/RealTimeMap/RealTimeMap-backend/services/notification-service/internal/domain/token"
)

type getterStub struct {
	devices []token.Model
	err     error
}

func (s *getterStub) GetUserDevices(context.Context, uint) ([]token.Model, error) {
	return s.devices, s.err
}

// fcmStub запоминает токены, на которые ушла отправка.
type fcmStub struct {
	sent []string
}

func (s *fcmStub) Send(_ context.Context, mess notification.Message) error {
	s.sent = append(s.sent, mess.Token)
	return nil
}

func device(deviceID, tok string, settings token.Settings) token.Model {
	return token.Model{DeviceID: deviceID, Token: tok, Settings: settings}
}

// TestHandleRespectsPerDeviceSettings — ради этого настройки и привязаны к
// устройству: выключенные уведомления на одном не мешают доставке на другие.
func TestHandleRespectsPerDeviceSettings(t *testing.T) {
	tests := []struct {
		name    string
		devices []token.Model
		kind    token.Kind
		want    []string
	}{
		{
			name: "тип выключен на ПК — уведомление уходит на телефон",
			devices: []token.Model{
				device("pc", "tok-pc", token.Settings{Enabled: true, Muted: map[string]bool{"chat": true}}),
				device("phone", "tok-phone", token.DefaultSettings()),
			},
			kind: token.KindChatMessage,
			want: []string{"tok-phone"},
		},
		{
			name: "устройство с общим выключателем не получает ничего",
			devices: []token.Model{
				device("pc", "tok-pc", token.Settings{Enabled: false}),
				device("phone", "tok-phone", token.DefaultSettings()),
			},
			kind: token.KindComment,
			want: []string{"tok-phone"},
		},
		{
			name: "отписка от чата не мешает комментариям на том же устройстве",
			devices: []token.Model{
				device("pc", "tok-pc", token.Settings{Enabled: true, Muted: map[string]bool{"chat": true}}),
			},
			kind: token.KindComment,
			want: []string{"tok-pc"},
		},
		{
			name: "тип выключен везде — не уходит никуда",
			devices: []token.Model{
				device("pc", "tok-pc", token.Settings{Enabled: true, Muted: map[string]bool{"chat": true}}),
				device("phone", "tok-phone", token.Settings{Enabled: true, Muted: map[string]bool{"chat": true}}),
			},
			kind: token.KindChatMessage,
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fcm := &fcmStub{}
			h := NewUserNotifyHanlder(&getterStub{devices: tt.devices}, fcm, zap.NewNop())

			err := h.Handle(context.Background(), NotifyUserCommand{
				UserID: 1,
				Title:  "t",
				Kind:   tt.kind,
			})
			if err != nil {
				t.Fatalf("Handle: %v", err)
			}

			if len(fcm.sent) != len(tt.want) {
				t.Fatalf("отправлено на %v, want %v", fcm.sent, tt.want)
			}
			for i, tok := range tt.want {
				if fcm.sent[i] != tok {
					t.Errorf("отправка[%d] = %q, want %q", i, fcm.sent[i], tok)
				}
			}
		})
	}
}

// TestCollapseKindsMatchTokenKinds защищает неявную связь: EventNotifyHandler
// приводит collapse.Kind к token.Kind обычным приведением строки, и разошедшиеся
// справочники дали бы не ошибку компиляции, а молча не доставленные
// уведомления — отписка перестала бы совпадать с типом события.
func TestCollapseKindsMatchTokenKinds(t *testing.T) {
	pairs := []struct {
		collapse collapse.Kind
		token    token.Kind
	}{
		{collapse.KindChatMessage, token.KindChatMessage},
		{collapse.KindComment, token.KindComment},
		{collapse.KindSubscriber, token.KindSubscriber},
	}

	for _, p := range pairs {
		if string(p.collapse) != string(p.token) {
			t.Errorf("collapse.Kind %q не совпадает с token.Kind %q", p.collapse, p.token)
		}
		if !token.Kind(p.collapse).Valid() {
			t.Errorf("collapse.Kind %q не проходит валидацию token.Kind", p.collapse)
		}
	}
}
