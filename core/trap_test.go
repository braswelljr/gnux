package core_test

import (
	"strings"
	"sync"
	"testing"

	"github.com/braswelljr/gnux/core"
)

// caught runs fn and returns the *core.Panic it raised, or nil.
func caught(fn func()) (p *core.Panic) {
	defer func() {
		if r := recover(); r != nil {
			p, _ = r.(*core.Panic)
		}
	}()
	fn()
	return nil
}

func TestTrapCarriesAPanicToTheCaller(t *testing.T) {
	p := caught(func() {
		var trap core.Trap
		var wg sync.WaitGroup
		for i := range 4 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer trap.Catch()
				if i == 2 {
					panic("worker failed")
				}
			}()
		}
		wg.Wait()
		if !trap.Caught() {
			t.Error("Caught() = false after a goroutine panicked")
		}
		trap.Release()
		t.Error("Release returned after a goroutine panicked")
	})
	if p == nil || p.Value != "worker failed" {
		t.Fatalf("caught %+v, want the worker's panic", p)
	}
	// The stack is the worker's, where the bug is, not the caller's.
	if !strings.Contains(string(p.Stack), "TestTrapCarriesAPanicToTheCaller.func") {
		t.Errorf("the stack does not show where the panic happened:\n%s", p.Stack)
	}
	if got := p.Error(); got != "panic: worker failed" {
		t.Errorf("Error() = %q", got)
	}
}

func TestTrapWithoutAPanic(t *testing.T) {
	var trap core.Trap
	func() { defer trap.Catch() }()
	if trap.Caught() {
		t.Error("Caught() = true with no panic")
	}
	trap.Release() // must not panic
}

// A panic that already crossed one goroutine keeps its first stack when it
// crosses another (a pipeline stage running xargsx -P).
func TestTrapKeepsTheFirstStack(t *testing.T) {
	inner := &core.Panic{Value: "deep", Stack: []byte("the original stack")}
	p := caught(func() {
		var trap core.Trap
		func() {
			defer trap.Catch()
			panic(inner)
		}()
		trap.Release()
	})
	if p != inner {
		t.Fatalf("caught %+v, want the same *Panic", p)
	}
}
