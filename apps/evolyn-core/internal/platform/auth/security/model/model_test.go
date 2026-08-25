package model

import "testing"

func TestNewSIDUniquenessAndFormat(t *testing.T) {
	seen := make(map[string]struct{})
	for i := 0; i < 200; i++ {
		sid, err := NewSID()
		if err != nil {
			t.Fatalf("new sid: %v", err)
		}
		if len(sid) != 32 {
			t.Fatalf("sid length = %d, want 32 (hex of 16 bytes)", len(sid))
		}
		for _, c := range sid {
			if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
				t.Fatalf("sid contains non-hex char %q", c)
			}
		}
		seen[sid] = struct{}{}
	}
	if len(seen) != 200 {
		t.Fatalf("sid collisions: %d unique of 200", len(seen))
	}
}
