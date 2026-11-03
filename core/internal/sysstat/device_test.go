//go:build linux || darwin

package sysstat

import (
	"testing"

	"golang.org/x/sys/unix"
)

func TestDeviceNumbers(t *testing.T) {
	for _, d := range [][2]uint32{{8, 1}, {0, 0}, {253, 17}, {1, 3}} {
		major, minor := DeviceNumbers(unix.Mkdev(d[0], d[1]))
		if major != uint64(d[0]) || minor != uint64(d[1]) {
			t.Errorf("DeviceNumbers(Mkdev(%d, %d)) = %d, %d", d[0], d[1], major, minor)
		}
	}
}
