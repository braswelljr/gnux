//go:build darwin

package ls

// deviceNumbers splits a darwin dev_t (8-bit major, 24-bit minor) into its
// major and minor numbers.
func deviceNumbers(dev uint64) (major, minor uint64) {
	return (dev >> 24) & 0xff, dev & 0xffffff
}
