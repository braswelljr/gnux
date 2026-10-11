//go:build !windows && !linux && !darwin

package ls

// deviceNumbers splits a dev_t using the traditional BSD layout: the major
// number in bits 8-15 and the minor number in the remaining bits.
func deviceNumbers(dev uint64) (major, minor uint64) {
	return (dev >> 8) & 0xff, dev&0xff | (dev>>8)&0xffff00
}
