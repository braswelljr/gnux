package core

import "bufio"

// FlushOutput flushes a tool's buffered standard output at the end of a run,
// as GNU's close_stdout does: a write error is diagnosed ("write error: No
// space left on device") and turns a successful run into ErrFailed, while a
// reader that has gone away (a closed pipe) is a quiet stop. Defer it with
// the run's named error result:
//
//	func (t *Tool) Run(args []string) (err error) {
//		out := bufio.NewWriter(t.streams.Out)
//		defer core.FlushOutput(out, t.diag, &err)
func FlushOutput(w *bufio.Writer, d *Diag, err *error) {
	ferr := w.Flush()
	if ferr == nil || IsClosedPipe(ferr) {
		return
	}
	d.Errorf("write error: %s", ErrText(ferr))
	if *err == nil {
		*err = ErrFailed
	}
}
