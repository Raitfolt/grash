package closer

import (
	"context"
	"fmt"
	"sync"

	"go.uber.org/zap"
)

// TODO add name to funcs
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

	var complete = make(chan struct{}, 1)

	log.Info("ready to execute close funcs")

	go func() {
		var wg sync.WaitGroup
		wg.Add(len(c.funcs))

		for _, f := range c.funcs {
			go func(f Func) {
				if err := f.F(ctx); err != nil {
					log.Error("closer", zap.String("error", err.Error()))
				}
				log.Info("closer", zap.String(f.Name, "was closed"))
				wg.Done()
			}(f)
		}

		wg.Wait()
		log.Info("graceful shutdown was done")
		complete <- struct{}{}
	}()

	select {
	case <-complete:
		break
	case <-ctx.Done():
		return fmt.Errorf("shutdown cancelled: %v", ctx.Err())
	}

	return nil
}

type Func struct {
	Name string
	F    func(ctx context.Context) error
}
