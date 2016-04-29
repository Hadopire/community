package provisionner

import (
	"io"
	"sync"

	"github.com/Nanocloud/community/nanocloud/broadcaster"
)

type ProvFunc func(io.Writer)

type Provisionner struct {
	fn   ProvFunc
	cond *sync.Cond
	done bool

	b broadcaster.Broadcaster
}

func New(fn ProvFunc) *Provisionner {
	cond := sync.NewCond(&sync.Mutex{})

	return &Provisionner{
		fn:   fn,
		cond: cond,
	}
}

func (p *Provisionner) _run() {
	p.fn(&p.b)

	p.cond.L.Lock()
	p.done = true
	p.cond.Broadcast()
	p.cond.L.Unlock()
}
func (p *Provisionner) Run() {
	go p._run()
}

func (p *Provisionner) Wait() {
	p.cond.L.Lock()
	if !p.done {
		p.cond.Wait()
	}
	p.cond.L.Unlock()
}

func (p *Provisionner) AddOutput(w io.Writer) {
	p.b.Add(w)
}
