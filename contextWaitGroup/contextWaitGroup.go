package contextWaitGroup

import (
	"context"
	"os"
	"os/signal"
	"sync"
)

type CWG struct {
	sync.WaitGroup
	Ctx    context.Context
	Cancel context.CancelFunc
}

func New(parent context.Context) CWG {
	ctx, cancel := context.WithCancel(parent)
	return CWG{Ctx: ctx, Cancel: cancel}
}

func (c *CWG) WithSignal(signals ...os.Signal) (stop context.CancelFunc) {
	c.Ctx, stop = signal.NotifyContext(c.Ctx, signals...)
	return
}

func (c *CWG) Go(f func(context.Context)) {
	c.WaitGroup.Go(func() {
		f(c.Ctx)
	})
}
