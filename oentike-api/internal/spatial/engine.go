package spatial

import (
	"bytes"
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

// IndexEntry reflects exactly 80 bytes in memory (8-byte aligned)
type IndexEntry struct {
	ID         [32]byte
	MinLon     float64
	MinLat     float64
	MaxLon     float64
	MaxLat     float64
	Offset     int64
	PointCount int32
	_          int32 // Padding
}

type Point struct {
	Lon float64
	Lat float64
}

type Engine struct {
	fileData []byte
	indices  []IndexEntry
}

// NewEngine maps the binary file to RAM and overlays the index struct.
func NewEngine(filepath string) (*Engine, error) {
	mf, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open spatial db: %w", err)
	}
	defer mf.Close()

	info, err := mf.Stat()
	if err != nil {
		return nil, err
	}

	// MAP_SHARED rzuca plik bezpośrednio w przestrzeń adresową procesu (Zero-Copy)
	data, err := syscall.Mmap(int(mf.Fd()), 0, int(info.Size()), syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		return nil, fmt.Errorf("mmap failed: %w", err)
	}

	// Walidacja Magic Bytes "OENT"
	if string(data[0:4]) != "OENT" {
		return nil, fmt.Errorf("invalid magic bytes in spatial db")
	}

	// Czytamy int32 z offsetu 4
	countPtr := (*int32)(unsafe.Pointer(&data[4]))
	count := *countPtr

	// Nasza tablica indeksów zaczyna się od bajtu 8.
	// Rzutujemy ją na tablicę struktur (Zero-Copy overlay).
	indices := unsafe.Slice((*IndexEntry)(unsafe.Pointer(&data[8])), count)

	return &Engine{
		fileData: data,
		indices:  indices,
	}, nil
}

// Close unmaps the memory.
func (e *Engine) Close() error {
	if e.fileData != nil {
		return syscall.Munmap(e.fileData)
	}
	return nil
}

// FindCell szuka pierwszego lasu, do którego wpada punkt (lat, lon).
func (e *Engine) FindCell(lon, lat float64) (string, bool) {
	for i := 0; i < len(e.indices); i++ {
		idx := &e.indices[i]

		// 1. AABB (Frustum Culling) - Odrzucamy 99% lasów w 4 operacjach!
		if lon < idx.MinLon || lon > idx.MaxLon || lat < idx.MinLat || lat > idx.MaxLat {
			continue
		}

		// 2. Jeśli punkt jest wewnątrz Bounding Boxa, rzutujemy wielokąt i robimy Ray-Casting
		points := unsafe.Slice((*Point)(unsafe.Pointer(&e.fileData[idx.Offset])), idx.PointCount)

		if rayCasting(lon, lat, points) {
			// Czyścimy ID z pustych bajtów (null bytes z tablicy 32 bajtowej)
			cleanID := string(bytes.Trim(idx.ID[:], "\x00"))
			return cleanID, true
		}
	}

	return "", false
}

func rayCasting(x, y float64, points []Point) bool {
	inside := false
	numPoints := len(points)
	j := numPoints - 1

	for i := 0; i < numPoints; i++ {
		xi, yi := points[i].Lon, points[i].Lat
		xj, yj := points[j].Lon, points[j].Lat

		intersect := ((yi > y) != (yj > y)) && (x < (xj-xi)*(y-yi)/(yj-yi)+xi)
		if intersect {
			inside = !inside
		}
		j = i
	}
	return inside
}
