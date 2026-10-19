// Package dfx wires the dfx command (GNU df) to its engine.
package dfx

import (
	"slices"
	"strings"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/df"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the dfx tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:  Meta(),
		GNU:   "df",
		Flags: Register,
		Run:   run,
	}
}

// Meta returns the documentation metadata for dfx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    df.Name,
		SeeAlso: []string{"dux", "statx", "lsx"},
		Use:     df.Name + " [OPTION]... [FILE]...",
		Short:   "Report file system space usage.",
		Long: "dfx shows, for each mounted file system, its size, the space used and " +
			"available, the percentage used and where it is mounted. With FILE operands it " +
			"shows only the file systems holding them. Sizes are in 1024-byte blocks by " +
			"default; -h prints them like 23G (--si/-H in powers of 1000), -k and -m use KiB " +
			"and MiB, and -B SIZE any unit. Use% is used/(used+available), rounded up, so " +
			"space reserved for the superuser shows as used. -i reports inodes instead of " +
			"blocks, -T adds the file system type, -P uses the POSIX format, and " +
			"--output=FIELDS chooses the columns (source, fstype, itotal, iused, iavail, " +
			"ipcent, size, used, avail, pcent, file, target). -t TYPE and -x TYPE select or " +
			"exclude file system types, -l leaves out network file systems, and --total adds " +
			"a grand total. Without -a, pseudo file systems (proc, sysfs, devfs, ...), file " +
			"systems with no blocks and duplicate mounts of the same device are left out; -a " +
			"shows them all, with '-' for any that cannot be examined. The block size " +
			"defaults to DF_BLOCK_SIZE, BLOCK_SIZE or BLOCKSIZE when set. GNU df only " +
			"runs where coreutils is installed; dfx reports the mounted file systems on " +
			"Linux, macOS and FreeBSD, and on Windows lists every drive (C:\\, D:\\, network " +
			"drives) with the same columns, quietly skipping a card reader or DVD drive with " +
			"no medium. --sync and --no-sync are accepted and have no effect, and -v is " +
			"ignored, as in GNU.",
		Permissions: "Needs no special permission; a mount point the user cannot examine is " +
			"left out (or shown with '-' under -a).",
		UseCases: []string{
			"Checking how much free space is left before a large download, build or backup.",
			"Finding which disk is filling up on a server.",
			"Checking which file system a directory lives on, and its type.",
			"Running out of inodes despite free space: checking with -i.",
			"Alerting from a script when a disk passes a usage threshold.",
		},
		Examples: []docmeta.Example{
			{
				Description: "See at a glance how full every disk is:",
				Bash:        "dfx -h",
				PowerShell:  "dfx -h",
				Output: "Filesystem      Size  Used Avail Use% Mounted on\n" +
					"/dev/nvme0n1p2  469G  301G  145G  68% /\n" +
					"/dev/nvme0n1p1  511M  6.1M  505M   2% /boot/efi\n" +
					"/dev/sda1       1.8T  1.6T  163G  91% /mnt/backup",
			},
			{
				Description: "Before a 20 GB download, check the free space where it will land:",
				Bash:        "dfx -h ~/Downloads",
				PowerShell:  "dfx -h $HOME\\Downloads",
				Output:      "Filesystem      Size  Used Avail Use% Mounted on\n/dev/nvme0n1p2  469G  301G  145G  68% /",
			},
			{
				Description: "Show only real local disks, with their file system types:",
				Bash:        "dfx -hlT",
				PowerShell:  "dfx -hlT",
				Output:      "Filesystem     Type  Size  Used Avail Use% Mounted on\n/dev/nvme0n1p2 ext4  469G  301G  145G  68% /",
			},
			{
				Description: "A server says 'No space left on device' but df shows free space: check the inodes:",
				Bash:        "dfx -i /var",
				PowerShell:  "dfx -i C:\\",
				Output:      "Filesystem      Inodes   IUsed IFree IUse% Mounted on\n/dev/sda2      1310720 1310720     0  100% /",
			},
			{
				Description: "Alert from a cron job when the root disk passes 90%:",
				Bash:        "[ \"$(dfx --output=pcent / | tailx -n 1 | trx -dc 0-9)\" -gt 90 ] && echox 'root disk above 90%'",
				PowerShell:  "if ([int]((dfx --output=pcent C:\\ | tailx -n 1) -replace '\\D','') -gt 90) { 'C: above 90%' }",
			},
			{
				Description: "Print just the available space, in megabytes, for a script:",
				Bash:        "dfx -BM --output=avail /data | tailx -n 1",
				PowerShell:  "dfx -BM --output=avail D:\\ | tailx -n 1",
				Output:      "  48213M",
			},
			{
				Description: "See which device and file system type a directory is on:",
				Bash:        "dfx --output=source,fstype,target /var/lib/docker",
				PowerShell:  "dfx --output=source,fstype,target C:\\Users",
				Output:      "Filesystem     Type Mounted on\n/dev/nvme0n1p2 ext4 /",
			},
			{
				Description: "Total the space across all local disks:",
				Bash:        "dfx -hl --total",
				PowerShell:  "dfx -hl --total",
			},
			{
				Description: "Check only the network shares:",
				Bash:        "dfx -h -t nfs -t cifs",
				PowerShell:  "dfx -h",
			},
			{
				Description: "Leave temporary file systems out of the report:",
				Bash:        "dfx -h -x tmpfs -x devtmpfs",
				PowerShell:  "dfx -h",
			},
			{
				Description: "Produce stable, parseable output with the POSIX format:",
				Bash:        "dfx -P | tailx -n +2",
				PowerShell:  "dfx -P | tailx -n +2",
			},
			{
				Description: "Show disk space in the portable gnux shell, the same on every OS:",
				Bash:        "gnux shell -c 'dfx -h . && dux -sh .'",
				PowerShell:  "gnux shell -c 'dfx -h . && dux -sh .'",
			},
		},
	}
}

// unitValue is the shared value of -B, -h, -H, -k and -m: whichever is
// given last decides the unit, as in GNU.
type unitValue struct {
	spec *string
	set  string // the block size the switch stands for; "" means -B's argument
}

func (v *unitValue) String() string { return "" }

// Type is what --help shows as the value placeholder.
func (v *unitValue) Type() string {
	if v.set == "" {
		return "SIZE"
	}
	return "bool"
}

// IsBoolFlag lets the unit switches take no argument.
func (v *unitValue) IsBoolFlag() bool { return v.set != "" }

func (v *unitValue) Set(s string) error {
	if v.set != "" {
		s = v.set
	}
	*v.spec = s
	return nil
}

// Register installs the dfx flags on fs.
func Register(fs *pflag.FlagSet) {
	unit := new(string)
	flag := func(name, short, set, usage string) *pflag.Flag {
		f := fs.VarPF(&unitValue{spec: unit, set: set}, name, short, usage)
		if set != "" {
			f.NoOptDefVal = "true"
		}
		return f
	}
	fs.BoolP("all", "a", false, "include pseudo, duplicate, inaccessible file systems")
	flag("block-size", "B", "", "scale sizes by SIZE before printing them; e.g., '-BM' prints sizes in units of 1,048,576 bytes")
	flag("human-readable", "h", "human-readable", "print sizes in powers of 1024 (e.g., 1023M)")
	flag("si", "H", "si", "print sizes in powers of 1000 (e.g., 1.1G)")
	fs.BoolP("inodes", "i", false, "list inode information instead of block usage")
	flag("kilobytes", "k", "1K", "like --block-size=1K")
	flag("megabytes", "m", "1M", "like --block-size=1M (obsolescent)").Hidden = true
	fs.BoolP("local", "l", false, "limit listing to local file systems")
	fs.Bool("no-sync", false, "do not invoke sync before getting usage info (default)")
	fs.StringSlice("output", nil, "use the output format defined by `FIELD_LIST`, or print all fields if FIELD_LIST is omitted")
	// A bare --output means every field; a blank (which pflag requires to be
	// non-empty) stands for "no list" and is trimmed away.
	fs.Lookup("output").NoOptDefVal = " "
	fs.BoolP("portability", "P", false, "use the POSIX output format")
	fs.Bool("sync", false, "invoke sync before getting usage info (no effect)")
	fs.Bool("total", false, "elide all entries insignificant to available space, and produce a grand total")
	fs.StringArrayP("type", "t", nil, "limit listing to file systems of type `TYPE`")
	fs.StringArrayP("type-f", "F", nil, "same as --type")
	fs.Lookup("type-f").Hidden = true
	fs.BoolP("print-type", "T", false, "print file system type")
	fs.StringArrayP("exclude-type", "x", nil, "limit listing to file systems not of type `TYPE`")
	fs.BoolP("sysv", "v", false, "(ignored)")

	// Long names GNU does not have: accepted, never an abbreviation.
	tool.OwnNames(fs, "kilobytes", "sysv")
}

// buildOptions folds the parsed flags into df.Options.
func buildOptions(fs *pflag.FlagSet) df.Options {
	on := func(name string) bool {
		b, _ := fs.GetBool(name)
		return b
	}
	o := df.Options{
		All:         on("all"),
		Inodes:      on("inodes"),
		Local:       on("local"),
		Portability: on("portability"),
		PrintType:   on("print-type"),
		Total:       on("total"),
		OutputSet:   fs.Changed("output"),
	}
	o.BlockSize = *fs.Lookup("block-size").Value.(*unitValue).spec
	types, _ := fs.GetStringArray("type")
	more, _ := fs.GetStringArray("type-f")
	o.Types = slices.Concat(types, more)
	o.ExcludeTypes, _ = fs.GetStringArray("exclude-type")
	out, _ := fs.GetStringSlice("output")
	for _, f := range out {
		if f = strings.TrimSpace(f); f != "" {
			o.Output = append(o.Output, f)
		}
	}
	return o
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	return df.New(buildOptions(fs), s).Run(args)
}
