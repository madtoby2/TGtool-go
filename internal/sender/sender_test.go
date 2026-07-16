package sender

import (
	"testing"
)

func TestResolveUsername(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"@testuser", "testuser"},
		{"https://t.me/testuser", "testuser"},
		{"https://t.me/testuser/123", "testuser"},
		{"testuser", "testuser"},
		{"  @testuser  ", "testuser"},
		{"", ""},
	}
	for _, tt := range tests {
		got := resolveUsername(tt.input)
		if got != tt.expected {
			t.Errorf("resolveUsername(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestResolveUsernameInviteLink(t *testing.T) {
	got := resolveUsername("https://t.me/+ABCdef123")
	// Should preserve the + part
	if got != "+ABCdef123" {
		t.Errorf("got %q, want +ABCdef123", got)
	}
}

func TestMinInt(t *testing.T) {
	if minInt(3, 5) != 3 {
		t.Error("minInt(3,5) should be 3")
	}
	if minInt(5, 2) != 2 {
		t.Error("minInt(5,2) should be 2")
	}
}
