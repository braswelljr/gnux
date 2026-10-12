//go:build !unix

package hashsum

// escapeTriggers are the characters that make a file name be escaped on its
// line. Off Unix a backslash is the path separator, so C:\dir\file is written
// as it is; a name with a newline is still escaped, backslashes included, so
// every line reads back the same.
const escapeTriggers = "\n\r"
