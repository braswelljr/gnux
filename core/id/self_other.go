//go:build !unix

package id

import "os/user"

// selfIdentity reads the current user from the account database. Without
// Unix process credentials there is no separate real and effective ID: on
// Windows the IDs are security identifiers (SIDs).
func selfIdentity() (Identity, error) {
	u, err := user.Current()
	if err != nil {
		return Identity{}, err
	}
	groups, _ := u.GroupIds()
	return Identity{UID: u.Uid, GID: u.Gid, EUID: u.Uid, EGID: u.Gid, Groups: groups}, nil
}
