//go:build unix

package hashsum

// escapeTriggers are the characters that make a file name be escaped on its
// line. On Unix a backslash is an ordinary file name character, so GNU's rule
// applies in full.
const escapeTriggers = "\\\n\r"
