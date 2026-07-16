package session

import (
	"testing"
)

func TestAccountInfo(t *testing.T) {
	info := AccountInfo{
		Phone:     "+8613800000000",
		Username:  "testuser",
		FirstName: "Test",
		IsActive:  true,
	}
	if info.Phone != "+8613800000000" {
		t.Error("phone mismatch")
	}
	if !info.IsActive {
		t.Error("expected active")
	}
}

func TestManagerListEmpty(t *testing.T) {
	mgr := NewManager()
	accs := mgr.List()
	if len(accs) != 0 {
		t.Errorf("expected 0 accounts, got %d", len(accs))
	}
}

func TestManagerGetNonexistent(t *testing.T) {
	mgr := NewManager()
	if a := mgr.Get("+999"); a != nil {
		t.Error("expected nil for nonexistent")
	}
}

func TestManagerRemoveNonexistent(t *testing.T) {
	mgr := NewManager()
	if mgr.Remove("+999") {
		t.Error("remove nonexistent should return false")
	}
}
