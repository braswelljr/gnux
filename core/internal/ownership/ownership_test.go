package ownership

import (
	"os/user"
	"strconv"
	"testing"
)

func TestParseSpec(t *testing.T) {
	tests := []struct {
		spec      string
		uid, gid  int
		userNamed bool
		err       string
	}{
		{"", Unset, Unset, false, ""},
		{"+1000", 1000, Unset, true, ""},
		{"+1000:+50", 1000, 50, true, ""},
		{":+50", Unset, 50, false, ""},
		{"+1000.+50", 1000, 50, true, ""}, // the obsolete separator
		{"no-such-user-gnux", 0, 0, false, "invalid user"},
		{"+1000:no-such-group-gnux", 0, 0, false, "invalid group"},
		{"+99999999999", 0, 0, false, "invalid user"},
		{"+2147480001:", 0, 0, false, "invalid spec"}, // no such user, so no login group
	}
	for _, tt := range tests {
		uid, gid, named, err := ParseSpec(tt.spec)
		switch {
		case tt.err != "":
			if err == nil || err.Error() != tt.err {
				t.Errorf("ParseSpec(%q) error = %v, want %q", tt.spec, err, tt.err)
			}
		case err != nil || uid != tt.uid || gid != tt.gid || named != tt.userNamed:
			t.Errorf("ParseSpec(%q) = %d, %d, %v, %v; want %d, %d, %v", tt.spec, uid, gid, named, err, tt.uid, tt.gid, tt.userNamed)
		}
	}
}

func TestParseSpecNames(t *testing.T) {
	me, err := user.Current()
	if err != nil || me.Username == "" {
		t.Skip("no current user to look up")
	}
	wantUID, _ := strconv.Atoi(me.Uid)
	wantGID, gerr := strconv.Atoi(me.Gid)
	if gerr != nil {
		t.Skip("the current user has no numeric group")
	}
	uid, gid, named, err := ParseSpec(me.Username)
	if err != nil || uid != wantUID || gid != Unset || !named {
		t.Fatalf("ParseSpec(%q) = %d, %d, %v, %v", me.Username, uid, gid, named, err)
	}
	// OWNER: takes the owner's login group too.
	if uid, gid, _, err = ParseSpec(me.Username + ":"); err != nil || uid != wantUID || gid != wantGID {
		t.Fatalf("ParseSpec(%q:) = %d, %d, %v; want %d, %d", me.Username, uid, gid, err, wantUID, wantGID)
	}
}

func TestLookupNumbers(t *testing.T) {
	if gid, err := LookupGroup("+7"); err != nil || gid != 7 {
		t.Fatalf("LookupGroup(+7) = %d, %v", gid, err)
	}
	if _, err := LookupGroup("-1"); err == nil {
		t.Fatal("LookupGroup(-1) was accepted")
	}
	if uid, _, err := LookupUser("+0"); err != nil || uid != 0 {
		t.Fatalf("LookupUser(+0) = %d, %v", uid, err)
	}
}
