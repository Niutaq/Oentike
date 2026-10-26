package spatial

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"
)

// Synthetic, deterministic WGS84 polygons. No database, network or local export.
func fixture(cells, vertices int) []byte {
	var buf bytes.Buffer
	buf.WriteString("OENT")
	binary.Write(&buf, binary.LittleEndian, int32(cells))
	for i := 0; i < cells; i++ {
		var id [32]byte
		copy(id[:], fmt.Sprintf("cell-%04d", i))
		x, y := float64(i%100)*0.1, float64(i/100)*0.1
		binary.Write(&buf, binary.LittleEndian, IndexEntry{ID: id, MinLon: (x + 0.04) - 0.04, MinLat: (y + 0.04) - 0.04, MaxLon: (x + 0.04) + 0.04, MaxLat: (y + 0.04) + 0.04, Offset: int64(8 + cells*80 + i*vertices*16), PointCount: int32(vertices)})
	}
	for i := 0; i < cells; i++ {
		x, y := float64(i%100)*0.1+0.04, float64(i/100)*0.1+0.04
		for j := 0; j < vertices; j++ {
			a := float64(j) * 2 * math.Pi / float64(vertices)
			binary.Write(&buf, binary.LittleEndian, Point{x + 0.04*math.Cos(a), y + 0.04*math.Sin(a)})
		}
	}
	return buf.Bytes()
}

func openFixture(t testing.TB, data []byte) *Engine {
	t.Helper()
	path := filepath.Join(t.TempDir(), "forests.bin")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	e, err := NewEngine(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := e.Close(); err != nil {
			t.Error(err)
		}
	})
	return e
}

func TestEngineFindCell(t *testing.T) {
	e := openFixture(t, fixture(429, 64))
	for _, tc := range []struct {
		name string
		x, y float64
		id   string
	}{
		{"first", 0.04, 0.04, "cell-0000"}, {"last", 2.84, 0.44, "cell-0428"},
		{"outside", -10, -10, ""}, {"inside box outside polygon", 0.001, 0.001, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			id, found := e.FindCell(tc.x, tc.y)
			if id != tc.id || found != (tc.id != "") {
				t.Fatalf("got (%q,%v), want %q", id, found, tc.id)
			}
		})
	}
}

var benchmarkID string
var benchmarkFound bool

func BenchmarkEngineFindCell(b *testing.B) {
	for _, n := range []int{429, 4290} {
		b.Run(fmt.Sprintf("cells=%d", n), func(b *testing.B) {
			e := openFixture(b, fixture(n, 128))
			for _, tc := range []struct {
				name string
				x, y float64
			}{
				{"first", 0.04, 0.04}, {"last", float64((n-1)%100)*0.1 + 0.04, float64((n-1)/100)*0.1 + 0.04},
				{"miss", -10, -10}, {"box-only", 0.001, 0.001},
			} {
				b.Run(tc.name, func(b *testing.B) {
					b.ReportAllocs()
					for i := 0; i < b.N; i++ {
						benchmarkID, benchmarkFound = e.FindCell(tc.x, tc.y)
					}
				})
			}
		})
	}
}

func TestRejectMalformedFiles(t *testing.T) {
	valid := fixture(1, 4)
	cases := map[string][]byte{"empty": {}, "short": []byte("OENT"), "truncated index": valid[:20], "truncated polygon": valid[:len(valid)-1]}
	mutate := func(name string, edit func([]byte)) { data := bytes.Clone(valid); edit(data); cases[name] = data }
	mutate("magic", func(d []byte) { d[0] = 'X' })
	mutate("negative count", func(d []byte) { binary.LittleEndian.PutUint32(d[4:8], math.MaxUint32) })
	mutate("negative offset", func(d []byte) { binary.LittleEndian.PutUint64(d[72:80], math.MaxUint64) })
	mutate("offset into header", func(d []byte) { binary.LittleEndian.PutUint64(d[72:80], 0) })
	mutate("too few vertices", func(d []byte) { binary.LittleEndian.PutUint32(d[80:84], 2) })
	mutate("oversized polygon", func(d []byte) { binary.LittleEndian.PutUint32(d[80:84], math.MaxUint32) })
	mutate("nan bounds", func(d []byte) { binary.LittleEndian.PutUint64(d[40:48], math.Float64bits(math.NaN())) })
	mutate("nan vertex", func(d []byte) { binary.LittleEndian.PutUint64(d[88:96], math.Float64bits(math.NaN())) })
	mutate("vertex outside box", func(d []byte) { binary.LittleEndian.PutUint64(d[88:96], math.Float64bits(170)) })
	mutate("empty id", func(d []byte) { clear(d[8:40]) })
	overlap := fixture(2, 4)
	binary.LittleEndian.PutUint64(overlap[152:160], binary.LittleEndian.Uint64(overlap[72:80]))
	cases["overlapping polygons"] = overlap
	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "bad.bin")
			if err := os.WriteFile(path, data, 0600); err != nil {
				t.Fatal(err)
			}
			e, err := NewEngine(path)
			if err == nil {
				e.Close()
				t.Fatal("accepted malformed file")
			}
		})
	}
}

func TestBoundaryAndClose(t *testing.T) {
	e := openFixture(t, fixture(1, 4))
	if _, ok := e.FindCell(0.08, 0.04); !ok {
		t.Fatal("vertex must be included")
	}
	for _, p := range []Point{{math.NaN(), 0}, {0, math.Inf(1)}} {
		if _, ok := e.FindCell(p.Lon, p.Lat); ok {
			t.Fatal("nonfinite query accepted")
		}
	}
	if err := e.Close(); err != nil {
		t.Fatal(err)
	}
	if _, ok := e.FindCell(0.04, 0.04); ok {
		t.Fatal("query after close")
	}
	if err := e.Close(); err != nil {
		t.Fatal(err)
	}
}

func FuzzDecodeCells(f *testing.F) {
	f.Add(fixture(1, 4))
	f.Add([]byte("OENT"))
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, data []byte) {
		cells, err := decodeCells(data)
		if err != nil {
			return
		}
		e := Engine{cells: cells}
		e.FindCell(0.04, 0.04)
	})
}

// Measures mmap + full validation with a warm filesystem cache, not disk latency.
func BenchmarkEngineOpen(b *testing.B) {
	for _, n := range []int{429, 4290} {
		b.Run(fmt.Sprintf("cells=%d", n), func(b *testing.B) {
			data := fixture(n, 128)
			path := filepath.Join(b.TempDir(), "forests.bin")
			if err := os.WriteFile(path, data, 0600); err != nil {
				b.Fatal(err)
			}
			b.SetBytes(int64(len(data)))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				e, err := NewEngine(path)
				if err != nil {
					b.Fatal(err)
				}
				if err := e.Close(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
