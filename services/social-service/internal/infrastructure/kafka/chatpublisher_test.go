package kafka

import (
	"strings"
	"testing"
)

func TestTruncateKeepsShortTextIntact(t *testing.T) {
	const text = "привет"

	if got := truncate(text, previewLimit); got != text {
		t.Fatalf("expected %q unchanged, got %q", text, got)
	}
}

// Обрезка идёт по рунам: срез по байтам развалил бы кириллицу на середине
// символа, и в топик уехал бы битый UTF-8.
func TestTruncateCutsByRunesNotBytes(t *testing.T) {
	text := strings.Repeat("я", 300)

	got := truncate(text, previewLimit)

	if !strings.HasSuffix(got, "…") {
		t.Fatalf("expected ellipsis suffix, got %q", got)
	}
	if runes := []rune(got); len(runes) != previewLimit+1 {
		t.Fatalf("expected %d runes, got %d", previewLimit+1, len(runes))
	}
	if !isValidRunes(got) {
		t.Fatalf("truncate produced invalid utf-8: %q", got)
	}
}

func TestWithoutSenderDropsEchoRecipient(t *testing.T) {
	got := withoutSender([]uint{1, 2, 3}, 2)

	if len(got) != 2 {
		t.Fatalf("expected 2 recipients, got %v", got)
	}
	for _, id := range got {
		if id == 2 {
			t.Fatal("sender must not receive a push about own message")
		}
	}
}

func TestWithoutSenderEmptyWhenOnlySender(t *testing.T) {
	if got := withoutSender([]uint{7}, 7); len(got) != 0 {
		t.Fatalf("expected no recipients, got %v", got)
	}
}

func isValidRunes(s string) bool {
	for _, r := range s {
		if r == '�' {
			return false
		}
	}
	return true
}
