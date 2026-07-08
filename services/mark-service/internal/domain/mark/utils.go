package mark

import "context"

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
