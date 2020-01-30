package validation

import (
	"testing"
	"time"
)

func TestIsTimeWithinWindow(t *testing.T) {
	now := time.Now()
	before := now.AddDate(0, 0, -1)

	withinWindow := IsTimeWithinWindow(before, 200000)
	if !withinWindow {
		t.Errorf("Time was not within window when it was expected to be")
	}

	withoutWindow := IsTimeWithinWindow(before, 1000)
	if withoutWindow {
		t.Errorf("Time was within window when it wasn't expect to be")
	}
}
