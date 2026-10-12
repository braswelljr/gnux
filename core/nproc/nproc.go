// Package nproc implements the engine behind nprocx, a GNU-compatible
// implementation of the GNU nproc command: print the number of processing
// units available.
package nproc

import (
	"fmt"
	"math"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/braswelljr/gnux/core"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "nprocx"

// Options mirrors the GNU nproc flag surface.
type Options struct {
	// All (--all) prints the number of installed processors, ignoring the
	// OMP_NUM_THREADS and OMP_THREAD_LIMIT variables.
	All bool
	// Ignore (--ignore=N) subtracts N from the count, never going below 1.
	Ignore uint64
}

// Nproc is the engine. Construct it with New.
type Nproc struct {
	Options

	// Available is the number of processing units the process may use, and
	// Installed the number in the machine; both default to runtime.NumCPU,
	// which is what Go can see without cgo. Getenv defaults to os.Getenv.
	Available func() int
	Installed func() int
	Getenv    func(string) string

	streams core.Streams
}

// New builds an nproc engine writing to s.
func New(opts Options, s core.Streams) *Nproc {
	return &Nproc{
		Options:   opts,
		Available: runtime.NumCPU,
		Installed: runtime.NumCPU,
		Getenv:    os.Getenv,
		streams:   s,
	}
}

// Run prints the count. nproc takes no operands.
func (n *Nproc) Run(args []string) error {
	if len(args) > 0 {
		return core.Usagef("extra operand '%s'", args[0])
	}
	count := n.count()
	if n.Ignore < count {
		count -= n.Ignore
	} else {
		count = 1
	}
	fmt.Fprintln(n.streams.Out, count)
	return nil
}

// count applies GNU's rules: --all is the installed count; otherwise
// OMP_NUM_THREADS, when set, replaces the available count, and
// OMP_THREAD_LIMIT caps either.
func (n *Nproc) count() uint64 {
	if n.All {
		return positive(n.Installed())
	}
	limit := ompThreads(n.Getenv("OMP_THREAD_LIMIT"))
	if limit == 0 {
		limit = math.MaxUint64
	}
	if threads := ompThreads(n.Getenv("OMP_NUM_THREADS")); threads != 0 {
		return min(threads, limit)
	}
	return min(positive(n.Available()), limit)
}

// ompThreads parses an OpenMP thread count: the first number of a
// comma-separated list, surrounded by optional blanks. Anything else, like
// an unset variable, is 0.
func ompThreads(s string) uint64 {
	s = strings.TrimLeft(s, " \t\n\v\f\r")
	end := 0
	for end < len(s) && s[end] >= '0' && s[end] <= '9' {
		end++
	}
	if end == 0 {
		return 0
	}
	rest := strings.TrimLeft(s[end:], " \t\n\v\f\r")
	if rest != "" && rest[0] != ',' {
		return 0
	}
	v, err := strconv.ParseUint(s[:end], 10, 64)
	if err != nil {
		return math.MaxUint64
	}
	return v
}

func positive(n int) uint64 {
	if n < 1 {
		return 1
	}
	return uint64(n)
}
