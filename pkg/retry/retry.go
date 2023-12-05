package retry

import (
	"context"
	"time"
)

func Retry(ctx context.Context, call func(ctx context.Context) error) error {
	t := time.NewTicker(time.Second)
	defer t.Stop()

	var err error

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-t.C:
			gctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			err = call(gctx)
			if err == nil {
				return nil
			}
		}
	}
}
