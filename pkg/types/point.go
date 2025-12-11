package types

import (
	"database/sql/driver"
	"encoding/hex"
	"fmt"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/encoding/ewkb"
	"github.com/paulmach/orb/encoding/wkb"
)

// Point структура для геометрии PostGis
type Point struct {
	orb.Point
}

// TODO понять...
// Scan реализует интерфейс sql.Scanner для чтения из БД
func (p *Point) Scan(val interface{}) error {
	if val == nil {
		*p = Point{}
		return nil
	}

	var b []byte

	switch v := val.(type) {
	case []byte:
		// Проверяем, не hex ли это (начинается с цифры или буквы a-f)
		if len(v) > 0 && isHexString(v) {
			decoded, err := hex.DecodeString(string(v))
			if err != nil {
				return fmt.Errorf("failed to decode hex: %w", err)
			}
			b = decoded
		} else {
			b = v
		}
	case string:
		// Строка из PostGIS почти всегда hex-encoded
		decoded, err := hex.DecodeString(v)
		if err != nil {
			return fmt.Errorf("failed to decode hex string: %w", err)
		}
		b = decoded
	default:
		return fmt.Errorf("cannot scan %T into Point", val)
	}

	// Пробуем EWKB (с SRID)
	point, _, err := ewkb.Unmarshal(b)
	if err != nil {
		// Fallback на обычный WKB
		point, err = wkb.Unmarshal(b)
		if err != nil {
			return fmt.Errorf("failed to unmarshal point: %w", err)
		}
	}

	if pt, ok := point.(orb.Point); ok {
		p.Point = pt
	} else {
		return fmt.Errorf("geometry is not a point: %T", point)
	}

	return nil
}

// Проверяет, является ли []byte hex-строкой
func isHexString(b []byte) bool {
	if len(b) == 0 {
		return false
	}
	for _, c := range b {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
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
