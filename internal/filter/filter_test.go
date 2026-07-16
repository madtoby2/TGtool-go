package filter

import (
	"strings"
	"testing"
)

func TestGeneratePhoneRange(t *testing.T) {
	phones := GeneratePhoneRange("+8613800000000", 5)
	if len(phones) != 5 {
		t.Fatalf("len = %d, want 5", len(phones))
	}
	if phones[0] != "+8613800000000" {
		t.Errorf("phones[0] = %s, want +8613800000000", phones[0])
	}
	if phones[4] != "+8613800000004" {
		t.Errorf("phones[4] = %s, want +8613800000004", phones[4])
	}
}

func TestGeneratePhoneRangeSingle(t *testing.T) {
	phones := GeneratePhoneRange("13800000000", 1)
	if len(phones) != 1 {
		t.Fatal("len should be 1")
	}
	if phones[0] != "13800000000" {
		t.Errorf("got %s", phones[0])
	}
}

func TestGeneratePhoneRangeZero(t *testing.T) {
	phones := GeneratePhoneRange("+8613800000000", 0)
	if len(phones) != 0 {
		t.Errorf("len = %d, want 0", len(phones))
	}
}

func TestGeneratePhoneRangeWithoutPrefix(t *testing.T) {
	phones := GeneratePhoneRange("12345678901", 3)
	if len(phones) != 3 {
		t.Fatal("len should be 3")
	}
	if !strings.Contains(phones[0], "12345678901") {
		t.Errorf("got %s", phones[0])
	}
}
