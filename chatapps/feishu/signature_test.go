package feishu

import (
	"testing"
)

func TestCalculateHMACSHA256(t *testing.T) {
	tests := []struct {
		name    string
		message string
		key     string
		wantLen int
	}{
		{
			name:    "basic_signature",
			message: "test_message",
			key:     "test_key",
			wantLen: 44, // Base64 encoded SHA256 is always 44 chars
		},
		{
			name:    "empty_message",
			message: "",
			key:     "test_key",
			wantLen: 44,
		},
		{
			name:    "empty_key",
			message: "test_message",
			key:     "",
			wantLen: 44,
		},
		{
			name:    "complex_message",
			message: `{"type":"im.message.receive_v1","data":{"message_id":"123"}}`,
			key:     "feishu_encrypt_key_2026",
			wantLen: 44,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateHMACSHA256(tt.message, tt.key)
			if len(got) != tt.wantLen {
				t.Errorf("calculateHMACSHA256() length = %v, want %v", len(got), tt.wantLen)
			}

			// Verify determinism
			got2 := calculateHMACSHA256(tt.message, tt.key)
			if got != got2 {
				t.Error("calculateHMACSHA256() is not deterministic")
			}
		})
	}
}

func TestSecureCompare(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
		want bool
	}{
		{
			name: "equal_strings",
			a:    "abc123",
			b:    "abc123",
			want: true,
		},
		{
			name: "different_strings",
			a:    "abc123",
			b:    "abc124",
			want: false,
		},
		{
			name: "different_lengths",
			a:    "abc123",
			b:    "abc1234",
			want: false,
		},
		{
			name: "empty_strings",
			a:    "",
			b:    "",
			want: true,
		},
		{
			name: "base64_signatures",
			a:    "dGVzdF9zaWduYXR1cmU=",
			b:    "dGVzdF9zaWduYXR1cmU=",
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := secureCompare(tt.a, tt.b); got != tt.want {
				t.Errorf("secureCompare() = %v, want %v", got, tt.want)
			}
		})
	}
}
