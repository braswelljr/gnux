package core

import (
	"fmt"
	"runtime/debug"
	"sync"
)

// Panic is a panic that happened on one goroutine and is raised again on
// another: the value it was raised with, and the stack of where it happened.
type Panic struct {
	Value any
	Stack []byte
}

func (p *Panic) Error() string {
	return fmt.Sprintf("panic: %v", p.Value)
}

// Trap brings a panic from the goroutines an engine starts back to the
// goroutine that called the engine. A panic on a goroutine of its own cannot
// be recovered by anyone else: it ends the whole process, and with it the
// shell pipeline, the playground or the program the tool was one call in.
//
// Each goroutine defers Catch, before any defer that tells the engine it has
// finished, so that the panic is recorded by the time the engine stops
// waiting. The engine calls Release once its goroutines have ended:
//
//	var trap core.Trap
//	var wg sync.WaitGroup
//	wg.Add(1)
//	go func() {
//		defer wg.Done()
//		defer trap.Catch()
//		work()
//	}()
//	wg.Wait()
//	trap.Release()
//
// The zero value is ready to use.
type Trap struct {
	mu    sync.Mutex
	first *Panic
}

// Catch recovers a panic on the calling goroutine and keeps the first one
// caught for Release. It must be deferred directly.
func (t *Trap) Catch() {
	r := recover()
	if r == nil {
		return
	}
	p, ok := r.(*Panic)
	if !ok {
		p = &Panic{Value: r, Stack: debug.Stack()}
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.first == nil {
		t.first = p
	}
}

// Caught reports whether a goroutine has panicked, for an engine that should
// stop handing out work once one has.
func (t *Trap) Caught() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.first != nil
}

// Release raises, on the calling goroutine, the first panic Catch recovered,
// as a *Panic. It does nothing when there was none.
func (t *Trap) Release() {
	t.mu.Lock()
	p := t.first
	t.mu.Unlock()
	if p != nil {
		panic(p)
	}
}
