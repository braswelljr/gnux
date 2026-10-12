//go:build !linux && !darwin

package sysstat

// DeviceNumbers splits a dev_t with the traditional BSD layout: the major
// number in bits 8-15 and the minor number in the rest.
func DeviceNumbers(dev uint64) (major, minor uint64) {
	return (dev >> 8) & 0xff, dev&0xff | (dev>>8)&0xffff00
}
