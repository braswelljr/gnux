//go:build !unix

package shred

import "os"

// appendOnly cannot be told off Unix; Windows has no append-only descriptors
// that shredding could meet as standard output.
func appendOnly(*os.File) bool { return false }

// syncDirectory does nothing: Windows cannot open a directory to flush it,
// and its renames are journaled by NTFS.
func syncDirectory(string) {}
