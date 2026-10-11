//go:build linux

package ls

// deviceNumbers splits a Linux dev_t into its major and minor numbers.
func deviceNumbers(dev uint64) (major, minor uint64) {
	major = (dev>>8)&0xfff | (dev>>32)&^0xfff
	minor = dev&0xff | (dev>>12)&^0xff
	return major, minor
}
