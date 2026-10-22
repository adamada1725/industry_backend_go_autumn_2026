package main

import (
	"context"
	"errors"
	"sync"
)

type Result[R any] struct {
	Value R
	Err   error
}

var ErrInvalidWorkers = errors.New("workers must be positive")

func ParallelMap[T any, R any](ctx context.Context, workers int, in []T, fn func(context.Context, T) (R, error)) ([]Result[R], error) {
	if workers <= 0 {
		return nil, ErrInvalidWorkers
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if len(in) == 0 {
		return []Result[R]{}, nil
	}

	count := min(workers, len(in))

	jobs := make(chan int)
	results := make([]Result[R], len(in))

	var wg sync.WaitGroup
	wg.Add(count)

	for range count {
		go func() {
			defer wg.Done()

			for i := range jobs {
				if ctx.Err() != nil {
					return
				}

				value, err := fn(ctx, in[i])

				if err != nil {
					results[i].Err = err
				} else {
					results[i].Value = value
				}
			}
		}()
	}

dispatch:
	for i := range in {
		if ctx.Err() != nil {
			break
		}

		select {
		case <-ctx.Done():
			break dispatch
		case jobs <- i:
		}
	}

	close(jobs)
	wg.Wait()

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return results, nil
}
