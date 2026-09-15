package main

import (
	"database/sql"
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type IndexEntry struct {
	ID         [32]byte
	MinLon     float64
	MinLat     float64
	MaxLon     float64
	MaxLat     float64
	Offset     int64
	PointCount int32
	_          int32 // Padding for 8-byte alignment
}

type Point struct {
	Lon float64
	Lat float64
}

type ForestData struct {
	ID     string
	Points []Point
}

func main() {
	dbURL := "postgres://oentike:oentike@127.0.0.1:54321/oentike?sslmode=disable"
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		log.Fatalf("DB connect failed: %v", err)
	}
	defer db.Close()

	fmt.Println("1. Fetching all forests from PostGIS...")
	start := time.Now()
	
	// Query all forests. We use ST_GeometryN(..., 1) to get the main polygon.
	query := `
		SELECT f.id, ST_X(dp.geom) as lon, ST_Y(dp.geom) as lat 
		FROM forest_units f, 
		LATERAL ST_DumpPoints(ST_ExteriorRing(ST_GeometryN(ST_Transform(f.geom, 4326), 1))) dp 
		ORDER BY f.id;
	`
	rows, err := db.Query(query)
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}
	defer rows.Close()

	forests := make(map[string]*ForestData)
	var forestOrder []string

	for rows.Next() {
		var id string
		var lon, lat float64
		if err := rows.Scan(&id, &lon, &lat); err != nil {
			log.Fatalf("Scan failed: %v", err)
		}
		
		fd, exists := forests[id]
		if !exists {
			fd = &ForestData{ID: id}
			forests[id] = fd
			forestOrder = append(forestOrder, id)
		}
		fd.Points = append(fd.Points, Point{Lon: lon, Lat: lat})
	}

	fmt.Printf("Fetched %d forests in %v\n", len(forestOrder), time.Since(start))
	fmt.Println("2. Calculating AABB and building Zero-Copy flat file...")

	f, err := os.Create("forests.bin")
	if err != nil {
		log.Fatalf("Failed to create file: %v", err)
	}
	defer f.Close()

	// 1. Write Header: Magic 'OENT' + Count
	f.Write([]byte{'O', 'E', 'N', 'T'})
	binary.Write(f, binary.LittleEndian, int32(len(forestOrder)))

	// 2. Reserve space for Index Array
	indexSize := int64(len(forestOrder)) * 80 // 80 bytes per IndexEntry
	indexStart := int64(8)
	payloadStart := indexStart + indexSize
	
	f.Seek(payloadStart, 0) // Move cursor to where payload begins

	var indices []IndexEntry
	currentOffset := payloadStart

	for _, id := range forestOrder {
		fd := forests[id]
		
		// Calculate AABB (Bounding Box)
		minLon, minLat := math.MaxFloat64, math.MaxFloat64
		maxLon, maxLat := -math.MaxFloat64, -math.MaxFloat64
		
		for _, p := range fd.Points {
			if p.Lon < minLon { minLon = p.Lon }
			if p.Lon > maxLon { maxLon = p.Lon }
			if p.Lat < minLat { minLat = p.Lat }
			if p.Lat > maxLat { maxLat = p.Lat }
		}

		// Prepare Index Entry
		var idBytes [32]byte
		copy(idBytes[:], id)

		entry := IndexEntry{
			ID:         idBytes,
			MinLon:     minLon,
			MinLat:     minLat,
			MaxLon:     maxLon,
			MaxLat:     maxLat,
			Offset:     currentOffset,
			PointCount: int32(len(fd.Points)),
		}
		indices = append(indices, entry)

		// Write payload points
		binary.Write(f, binary.LittleEndian, fd.Points)
		currentOffset += int64(len(fd.Points) * 16)
	}

	// 3. Go back and write the Index Array
	f.Seek(indexStart, 0)
	binary.Write(f, binary.LittleEndian, indices)

	fmt.Printf("Export complete. File size: %d bytes.\n", currentOffset)
}
