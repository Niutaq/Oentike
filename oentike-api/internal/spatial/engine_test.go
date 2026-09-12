package spatial

import (
	"testing"
)

// To run this test, make sure forests.bin is accessible (e.g. at ../../../forests.bin)
func TestEngine_FindCell(t *testing.T) {
	// Wymaga wygenerowanego wcześniej pliku forests.bin w głównym katalogu Oentike
	engine, err := NewEngine("../../forests.bin")
	if err != nil {
		t.Skipf("Skipping test, missing forests.bin: %v", err)
	}
	defer engine.Close()

	tests := []struct {
		name     string
		lon      float64
		lat      float64
		wantID   string
		wantFound bool
	}{
		{"Center of Janow (nadl-05-31)", 22.37771, 50.70819, "nadl-05-31", true},
		{"Warsaw / Chojnow (nadl-17-02)", 21.0122, 52.2297, "nadl-17-02", true},
		{"Baltic Sea (Not in any forest)", 19.0, 55.0, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, gotFound := engine.FindCell(tt.lon, tt.lat)
			if gotFound != tt.wantFound {
				t.Errorf("FindCell() gotFound = %v, want %v", gotFound, tt.wantFound)
			}
			if gotID != tt.wantID {
				t.Errorf("FindCell() gotID = %v, want %v", gotID, tt.wantID)
			}
		})
	}
}

func BenchmarkEngine_FindCell(b *testing.B) {
	engine, err := NewEngine("../../forests.bin")
	if err != nil {
		b.Skipf("Skipping benchmark, missing forests.bin: %v", err)
	}
	defer engine.Close()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Benchmark punktu wewnątrz lasu Janowskiego (pesymistyczny przypadek, przechodzi AABB)
		_, _ = engine.FindCell(22.25, 50.60)
	}
}
