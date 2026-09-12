package bdl

import "testing"

func TestUnitID(t *testing.T) {
	got := UnitID("06", "12")
	if got != "nadl-06-12" {
		t.Fatalf("got %q", got)
	}
}

func TestDisplayName(t *testing.T) {
	if DisplayName("Janów Lubelski") != "Nadleśnictwo Janów Lubelski" {
		t.Fatal(DisplayName("Janów Lubelski"))
	}
	if DisplayName("  ") != "Nadleśnictwo" {
		t.Fatal(DisplayName("  "))
	}
}
