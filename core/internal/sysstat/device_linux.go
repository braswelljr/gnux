//go:build linux

package sysstat

// DeviceNumbers splits a Linux dev_t into its major and minor numbers.
func DeviceNumbers(dev uint64) (major, minor uint64) {
	return (dev>>8)&0xfff | (dev>>32)&^0xfff, dev&0xff | (dev>>12)&^0xff
}
