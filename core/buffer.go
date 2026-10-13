package core

import (
	"bufio"
	"io"
	"strconv"
)

// BufferSize is the size of the buffers engines read and write through. At
// bufio's default of 4096 bytes a tool spends a large share of its time in
// read and write calls; 64 KiB makes that share negligible and is still small
// next to any input worth buffering.
const BufferSize = 64 << 10

// NewReader returns a buffered reader of BufferSize bytes over r. Engines read
// their input through it rather than through bufio.NewReader.
func NewReader(r io.Reader) *bufio.Reader {
	return bufio.NewReaderSize(r, BufferSize)
}

// NewWriter returns a buffered writer of BufferSize bytes over w. Engines
// write their output through it rather than through bufio.NewWriter, and flush
// it before returning (see FlushOutput).
func NewWriter(w io.Writer) *bufio.Writer {
	return bufio.NewWriterSize(w, BufferSize)
}

// AppendInt appends n in decimal to dst, right-aligned with spaces in a field
// of at least width bytes, as C's "%*d" prints it. It is what a per-line
// counter (cat -n, nl, uniq -c) is written with: formatting through fmt for
// every line costs more than copying the line does.
func AppendInt(dst []byte, n int64, width int) []byte {
	var digits [20]byte // the longest int64, sign included
	text := strconv.AppendInt(digits[:0], n, 10)
	for pad := width - len(text); pad > 0; pad-- {
		dst = append(dst, ' ')
	}
	return append(dst, text...)
}

// FlushIfIdle flushes out when in has no more input buffered, that is, when
// the engine's next read will wait for whoever produces the input. A filter
// calls it after each record it has written, which makes it answer a slow
// producer (a terminal, tail -f, a build log) record by record, and costs
// nothing on bulk input: a buffer of BufferSize bytes runs dry once per
// BufferSize bytes read, which is when the output would be written anyway.
//
// A failed flush is not reported here. The error stays with out, where the
// engine's next write or its final flush finds it.
func FlushIfIdle(out *bufio.Writer, in *bufio.Reader) {
	if in.Buffered() == 0 {
		_ = out.Flush()
	}
}
