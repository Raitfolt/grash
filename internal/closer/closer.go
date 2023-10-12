package closer

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"go.uber.org/zap"
)

type closer struct {
	mu    sync.Mutex
	funcs []Func
}

func New() *closer {
	return &closer{}
}

func (c *closer) Add(f Func) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.funcs = append(c.funcs, f)
}

func (c *closer) Close(ctx context.Context, log *zap.Logger) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var (
		msgs     = make([]string, 0, len(c.funcs))
		complete = make(chan struct{}, 1)
	)

	log.Info("ready to execute close funcs")

	go func() {
		var wg sync.WaitGroup
		wg.Add(len(c.funcs))

		for _, f := range c.funcs {
			go func() {
				f := f
				if err := f(ctx); err != nil {
					// TODO: set mutex
					msgs = append(msgs, fmt.Sprintf("[!] %v", err))
				}

				wg.Done()
			}()
		}
		wg.Wait()
		complete <- struct{}{}
	}()

	select {
	case <-complete:
		break
	case <-ctx.Done():
		return fmt.Errorf("shutdown cancelled: %v", ctx.Err())
	}

	if len(msgs) > 0 {
		return fmt.Errorf(
			"shutdown finished with error(s): \n%s",
			strings.Join(msgs, "\n"),
		)
	}

	return nil
}

type Func func(ctx context.Context) error
