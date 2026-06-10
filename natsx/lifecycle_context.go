package natsx

import (
	"context"
	"sync"
	"time"
)

type lifecycleContext struct {
	done chan struct{}
	once sync.Once
}

func newLifecycleContext() (context.Context, context.CancelFunc) {
	ctx := &lifecycleContext{done: make(chan struct{})}
	return ctx, ctx.cancel
}

func (c *lifecycleContext) Deadline() (time.Time, bool) {
	return time.Time{}, false
}

func (c *lifecycleContext) Done() <-chan struct{} {
	return c.done
}

func (c *lifecycleContext) Err() error {
	select {
	case <-c.done:
		return context.Canceled
	default:
		return nil
	}
}

func (c *lifecycleContext) Value(any) any {
	return nil
}

func (c *lifecycleContext) cancel() {
	c.once.Do(func() {
		close(c.done)
	})
}
