package token

import "testing"

func TestModelAllows(t *testing.T) {
	tests := []struct {
		name     string
		settings Settings
		kind     Kind
		want     bool
	}{
		{
			name:     "устройство с настройками по умолчанию принимает всё",
			settings: DefaultSettings(),
			kind:     KindChatMessage,
			want:     true,
		},
		{
			name:     "общий выключатель перекрывает невыключенный тип",
			settings: Settings{Enabled: false},
			kind:     KindChatMessage,
			want:     false,
		},
		{
			name:     "выключенный тип не проходит",
			settings: Settings{Enabled: true, Muted: map[string]bool{"chat": true}},
			kind:     KindChatMessage,
			want:     false,
		},
		{
			name:     "выключение одного типа не задевает соседний",
			settings: Settings{Enabled: true, Muted: map[string]bool{"chat": true}},
			kind:     KindComment,
			want:     true,
		},
		{
			name:     "системная отправка без типа проходит мимо отписок",
			settings: Settings{Enabled: true, Muted: map[string]bool{"chat": true}},
			kind:     "",
			want:     true,
		},
		{
			name:     "системная отправка уважает общий выключатель",
			settings: Settings{Enabled: false},
			kind:     "",
			want:     false,
		},
		{
			name:     "неизвестный тип считается включённым",
			settings: Settings{Enabled: true, Muted: map[string]bool{"chat": true}},
			kind:     Kind("achievement"),
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Model{Settings: tt.settings}
			if got := m.Allows(tt.kind); got != tt.want {
				t.Errorf("Allows(%q) = %v, want %v", tt.kind, got, tt.want)
			}
		})
	}
}

// TestSettingsRoundTrip проверяет, что настройки переживают запись в jsonb и
// чтение обратно: Scan и Value — единственное, что стоит между картой в памяти
// и колонкой в базе.
func TestSettingsRoundTrip(t *testing.T) {
	original := Settings{Enabled: true, Muted: map[string]bool{"chat": true}}

	raw, err := original.Value()
	if err != nil {
		t.Fatalf("Value: %v", err)
	}

	var restored Settings
	if err := restored.Scan(raw); err != nil {
		t.Fatalf("Scan: %v", err)
	}

	if restored.Enabled != original.Enabled {
		t.Errorf("Enabled = %v, want %v", restored.Enabled, original.Enabled)
	}
	if !restored.Muted["chat"] {
		t.Errorf("Muted = %v, want chat muted", restored.Muted)
	}
}

// TestSettingsScanNull фиксирует поведение на NULL: устройство со сломанным
// settings не должно получать уведомления молча — Enabled=false честнее, чем
// ошибка чтения на каждой отправке.
func TestSettingsScanNull(t *testing.T) {
	var s Settings
	if err := s.Scan(nil); err != nil {
		t.Fatalf("Scan(nil): %v", err)
	}
	if s.Enabled {
		t.Error("Enabled = true после Scan(nil), want false")
	}
}
