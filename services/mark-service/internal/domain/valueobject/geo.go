package valueobject

import "github.com/mmcloughlin/geohash"

const GeohashPersistence = 5

const (
	geoHashLatStep = 0.0439
	geoHashLonStep = 0.0439
)

type Point struct {
	Lon float64
	Lat float64
}

type BoundingBox struct {
	LeftTop     Point
	RightBottom Point
}

func (b BoundingBox) GeoHashes() []string {
	minLat := b.RightBottom.Lat
	maxLat := b.LeftTop.Lat
	minLon := b.LeftTop.Lon
	maxLon := b.RightBottom.Lon

	latCells := int((maxLat-minLat)/geoHashLatStep) + 2
	lonCells := int((maxLon-minLon)/geoHashLonStep) + 2
	size := latCells * lonCells

	seen := make(map[string]struct{}, size)

	latStep := geoHashLatStep * 0.95
	lonStep := geoHashLonStep * 0.95

	maxLat += geoHashLatStep * 0.1
	maxLon += geoHashLonStep * 0.1

	for lat := minLat; lat <= maxLat; lat += latStep {
		for lon := minLon; lon <= maxLon; lon += lonStep {
			hash := geohash.EncodeWithPrecision(lat, lon, GeohashPersistence)
			seen[hash] = struct{}{}
		}
	}

	result := make([]string, 0, len(seen))
	for hash := range seen {
		result = append(result, hash)
	}
	return result
}
