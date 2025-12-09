package types

import (
	"database/sql/driver"
	"fmt"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/encoding/ewkb"
	"github.com/paulmach/orb/encoding/wkb"
)

// Point структура для геометрии PostGis
type Point struct {
	orb.Point
}

// Scan реализует интерфейс sql.Scanner для чтения из БД
func (p *Point) Scan(val interface{}) error {
	if val == nil {
		*p = Point{}
		return nil
	}

	b, ok := val.([]byte)
	if !ok {
		return fmt.Errorf("cannot scan %T into Point", val)
	}

	// Пробуем сначала EWKB (Extended WKB с SRID)
	point, _, err := ewkb.Unmarshal(b)
	if err != nil {
		// Если не получилось, пробуем обычный WKB
		point, err = wkb.Unmarshal(b)
		if err != nil {
			return fmt.Errorf("failed to unmarshal point: %w", err)
		}
	}

	if pt, ok := point.(orb.Point); ok {
		p.Point = pt
	}

	return nil
}

// Value реализует интерфейс driver.Valuer для записи в БД
// Использует EWKB формат с SRID 4326 для PostGIS
func (p Point) Value() (driver.Value, error) {
	// Используем EWKB с явным указанием SRID 4326
	hexString, err := ewkb.MarshalToHex(p.Point, 4326)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal point to EWKB: %w", err)
	}
	return hexString, nil
}
