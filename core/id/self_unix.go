//go:build unix

package id

import (
	"strconv"
	"syscall"
)

// selfIdentity reads the process's real and effective IDs and its
// supplementary groups.
func selfIdentity() (Identity, error) {
	groups, err := syscall.Getgroups()
	if err != nil {
		return Identity{}, err
	}
	who := Identity{
		UID:  strconv.Itoa(syscall.Getuid()),
		GID:  strconv.Itoa(syscall.Getgid()),
		EUID: strconv.Itoa(syscall.Geteuid()),
		EGID: strconv.Itoa(syscall.Getegid()),
	}
	for _, g := range groups {
		who.Groups = append(who.Groups, strconv.Itoa(g))
	}
	return who, nil
}
