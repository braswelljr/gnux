//go:build darwin

package sysstat

// DeviceNumbers splits a darwin dev_t (8-bit major, 24-bit minor).
func DeviceNumbers(dev uint64) (major, minor uint64) {
	return (dev >> 24) & 0xff, dev & 0xffffff
}
