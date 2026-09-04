package ingest

import (
	"testing"
	"time"
)

func TestDue(t *testing.T) {
	now := time.Date(2026, 9, 4, 8, 0, 0, 0, time.UTC)
	minAge := 50 * time.Minute

	if !Due(time.Time{}, false, now, minAge) {
		t.Fatal("missing fetch is due")
	}
	if Due(now.Add(-10*time.Minute), true, now, minAge) {
		t.Fatal("fresh fetch is not due")
	}
	if !Due(now.Add(-50*time.Minute), true, now, minAge) {
		t.Fatal("stale fetch is due")
	}
	if Due(now.Add(time.Minute), true, now, minAge) {
		t.Fatal("future fetch is not due")
	}
}
