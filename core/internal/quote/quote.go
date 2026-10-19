// Package quote renders file names in gnulib's shell-escape quoting style,
// the style GNU stat and du use: a name is left alone when a shell would
// read it back unchanged, single-quoted otherwise, with control characters
// written as $'\n' outside the quotes.
package quote

import "github.com/braswelljr/gnux/core/internal/quotearg"

// ShellEscape quotes name for a POSIX shell.
func ShellEscape(name string) string {
	return quotearg.Quote(name, quotearg.Options{Style: quotearg.ShellEscape})
}
