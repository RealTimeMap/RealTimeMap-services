package events

import "testing"

func TestParseUserDeleted(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantOK  bool
		wantErr bool
		wantID  uint
	}{
		{
			name:   "user.deleted",
			value:  `{"id":"1","type":"user.deleted","timestamp":"2026-09-26T10:00:00Z","payload":{"user_id":42,"email":"a@b.c","deleted_at":"2026-09-26T10:00:00Z"}}`,
			wantOK: true,
			wantID: 42,
		},
		{
			name:  "другое событие топика",
			value: `{"id":"1","type":"user.logged_in","payload":{"user_id":42}}`,
		},
		{
			name:    "битый конверт",
			value:   `{`,
			wantErr: true,
		},
		{
			name:    "нет user_id",
			value:   `{"type":"user.deleted","payload":{"email":"a@b.c"}}`,
			wantOK:  true,
			wantErr: true,
		},
		{
			name:    "битый payload",
			value:   `{"type":"user.deleted","payload":"oops"}`,
			wantOK:  true,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok, err := ParseUserDeleted([]byte(tt.value))
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if got.UserID != tt.wantID {
				t.Fatalf("UserID = %d, want %d", got.UserID, tt.wantID)
			}
		})
	}
}
