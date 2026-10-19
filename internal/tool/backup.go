package tool

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
)

// fromEnvironment is --backup given without a CONTROL: GNU then reads
// VERSION_CONTROL. It is what the help shows as the value-less default.
const fromEnvironment = "$VERSION_CONTROL"

// RegisterBackup installs GNU's backup options, shared by cpx, mvx, lnx and
// installx: --backup[=CONTROL], -b (the same without a CONTROL) and
// -S, --suffix=SUFFIX (which asks for backups too, as in GNU).
func RegisterBackup(fs *pflag.FlagSet) {
	fs.String("backup", "", "make a backup of each existing destination file; `CONTROL` is none (off), simple (never), existing (nil) or numbered (t)")
	fs.Lookup("backup").NoOptDefVal = fromEnvironment
	fs.BoolP("backup-default", "b", false, "like --backup but does not accept an argument")
	fs.StringP("suffix", "S", "", "override the usual backup suffix, '~', with `SUFFIX`")
	OwnNames(fs, "backup-default")
}

// Backups reports whether the backup options ask for backups at all.
func Backups(fs *pflag.FlagSet) bool {
	b, _ := fs.GetBool("backup-default")
	return b || fs.Changed("backup") || fs.Changed("suffix")
}

// BackupOptions folds the options RegisterBackup installed into a
// core.Backup, as GNU does: without a CONTROL the VERSION_CONTROL variable
// chooses (existing when it is unset), and without -S the
// SIMPLE_BACKUP_SUFFIX variable gives the suffix. getenv reads the
// environment.
func BackupOptions(fs *pflag.FlagSet, getenv func(string) string) (core.Backup, error) {
	if !Backups(fs) {
		return core.Backup{}, nil
	}
	var (
		b   core.Backup
		err error
	)
	if word, _ := fs.GetString("backup"); fs.Changed("backup") && word != fromEnvironment && word != "" {
		b.Control, err = core.ParseBackupControl("backup type", word)
	} else {
		b.Control, err = core.ParseBackupControl("$VERSION_CONTROL", getenv("VERSION_CONTROL"))
	}
	if err != nil {
		return core.Backup{}, err
	}
	if fs.Changed("suffix") {
		b.Suffix, _ = fs.GetString("suffix")
	} else {
		b.Suffix = getenv("SIMPLE_BACKUP_SUFFIX")
	}
	return b, nil
}
