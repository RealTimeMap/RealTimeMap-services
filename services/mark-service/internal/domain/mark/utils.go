package mark

import (
	"context"
	"math"
)

const earthRadius float64 = 6371008.8

type checkFn func(ctx context.Context, id uint) (bool, error)

func checkMarkExist(ctx context.Context, fn checkFn, id uint) error {
	exists, err := fn(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return ErrMarkNotFound(id)
	}
	return nil
}

func checkDistance(lat1, lon1, lat2, lon2 float64) float64 {
	f1, f2 := lat1*math.Pi/180, lat2*math.Pi/180
	d1 := (lat2 - lat1) * math.Pi / 180
	d2 := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(d1/2)*math.Sin(d1/2) +
		math.Cos(f1)*math.Cos(f2)*math.Sin(d2/2)*math.Sin(d2/2)

	return 2 * earthRadius * math.Asin(math.Sqrt(a))
}
