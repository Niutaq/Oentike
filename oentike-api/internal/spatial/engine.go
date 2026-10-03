// Package spatial reads the legacy OENT single-ring polygon format.
// Files must remain immutable while open. Close must not race with FindCell.
package spatial

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"syscall"
)

// IndexEntry is the 80-byte little-endian wire record (including padding).
type IndexEntry struct {
	ID                             [32]byte
	MinLon, MinLat, MaxLon, MaxLat float64
	Offset                         int64
	PointCount                     int32
	_                              int32
}
type Point struct{ Lon, Lat float64 }
type cell struct {
	id                             string
	minLon, minLat, maxLon, maxLat float64
	points                         []byte
}
type Engine struct {
	fileData []byte
	cells    []cell
}

func NewEngine(path string) (*Engine, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open spatial db: %w", err)
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() < 8 || uint64(info.Size()) > uint64(^uint(0)>>1) {
		return nil, fmt.Errorf("invalid spatial db size or file type")
	}
	data, err := syscall.Mmap(int(f.Fd()), 0, int(info.Size()), syscall.PROT_READ, syscall.MAP_PRIVATE)
	if err != nil {
		return nil, fmt.Errorf("mmap spatial db: %w", err)
	}
	cells, err := decodeCells(data)
	if err != nil {
		syscall.Munmap(data)
		return nil, err
	}
	return &Engine{fileData: data, cells: cells}, nil
}

func number(data []byte) float64 { return math.Float64frombits(binary.LittleEndian.Uint64(data)) }
func finite(v float64) bool      { return !math.IsNaN(v) && !math.IsInf(v, 0) }

// Decode only the index. Polygon coordinates stay in the mapped file; explicit
// endian decoding avoids unsafe pointers, native layout and alignment assumptions.
func decodeCells(data []byte) ([]cell, error) {
	if len(data) < 8 || string(data[:4]) != "OENT" {
		return nil, fmt.Errorf("invalid OENT header")
	}
	count := uint64(binary.LittleEndian.Uint32(data[4:8]))
	if count > uint64((len(data)-8)/80) {
		return nil, fmt.Errorf("truncated OENT index")
	}
	payload := uint64(8) + count*80
	cells := make([]cell, 0, int(count))
	nextOffset := payload
	for i := 0; i < int(count); i++ {
		record := data[8+i*80 : 8+(i+1)*80]
		id := bytes.TrimRight(record[:32], "\x00")
		if len(id) == 0 {
			return nil, fmt.Errorf("cell %d: empty ID", i)
		}
		for _, c := range id {
			if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-') {
				return nil, fmt.Errorf("cell %d: invalid ID", i)
			}
		}
		c := cell{id: string(id), minLon: number(record[32:40]), minLat: number(record[40:48]), maxLon: number(record[48:56]), maxLat: number(record[56:64])}
		if !finite(c.minLon) || !finite(c.maxLon) || !finite(c.minLat) || !finite(c.maxLat) || c.minLon > c.maxLon || c.minLat > c.maxLat || c.minLon < -180 || c.maxLon > 180 || c.minLat < -90 || c.maxLat > 90 {
			return nil, fmt.Errorf("cell %d: invalid WGS84 bounds", i)
		}
		offset := binary.LittleEndian.Uint64(record[64:72])
		points := uint64(binary.LittleEndian.Uint32(record[72:76]))
		if offset != nextOffset || offset > uint64(len(data)) || points < 3 || points > (uint64(len(data))-offset)/16 {
			return nil, fmt.Errorf("cell %d: invalid polygon range", i)
		}
		c.points = data[offset : offset+points*16]
		nextOffset = offset + points*16
		for j := 0; j < len(c.points); j += 16 {
			x, y := number(c.points[j:j+8]), number(c.points[j+8:j+16])
			if !finite(x) || !finite(y) || x < c.minLon || x > c.maxLon || y < c.minLat || y > c.maxLat {
				return nil, fmt.Errorf("cell %d: vertex outside bounds", i)
			}
		}
		cells = append(cells, c)
	}
	return cells, nil
}

// Close is idempotent. Queries after Close return no match.
func (e *Engine) Close() error {
	if e.fileData == nil {
		return nil
	}
	if err := syscall.Munmap(e.fileData); err != nil {
		return err
	}
	e.fileData = nil
	e.cells = nil
	return nil
}

// FindCell returns the first matching ring in file order. Boundary points count
// as inside. The legacy format cannot express polygon holes or multipolygons.
func (e *Engine) FindCell(lon, lat float64) (string, bool) {
	if !finite(lon) || !finite(lat) {
		return "", false
	}
	for i := range e.cells {
		c := &e.cells[i]
		if lon < c.minLon || lon > c.maxLon || lat < c.minLat || lat > c.maxLat {
			continue
		}
		if contains(lon, lat, c.points) {
			return c.id, true
		}
	}
	return "", false
}

func contains(x, y float64, points []byte) bool {
	inside := false
	j := len(points) - 16
	for i := 0; i < len(points); i += 16 {
		xi, yi := number(points[i:i+8]), number(points[i+8:i+16])
		xj, yj := number(points[j:j+8]), number(points[j+8:j+16])
		// Exact boundary predicate; no arbitrary geographic epsilon.
		if (x-xi)*(yj-yi) == (y-yi)*(xj-xi) && x >= math.Min(xi, xj) && x <= math.Max(xi, xj) && y >= math.Min(yi, yj) && y <= math.Max(yi, yj) {
			return true
		}
		if (yi > y) != (yj > y) && x < (xj-xi)*(y-yi)/(yj-yi)+xi {
			inside = !inside
		}
		j = i
	}
	return inside
}
