package schedule

import (
	"testing"
)

func TestLoadBlackouts(t *testing.T) {
	blackouts, err := LoadBlackouts(3)
	if err != nil {
		t.Errorf("unexpected error: %e", err)
		return
	}

	if len(blackouts) < 2 {
		t.Errorf("loaded blackouts schedule is too short")
	}
}
