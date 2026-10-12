// Package installx wires the installx command (GNU install) to its engine.
package installx

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/install"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the installx tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:  Meta(),
		GNU:   "install",
		Flags: Register,
		Run:   run,
	}
}

// Meta returns the documentation metadata for installx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    install.Name,
		SeeAlso: []string{"cpx", "mkdirx", "touchx"},
		Use:     install.Name + " [OPTION]... SOURCE... DEST",
		Short:   "Copy files into place and set their mode, owner and group.",
		Long: "installx copies SOURCE to DEST, or several SOURCE files into the existing " +
			"directory DEST (or into -t DIRECTORY), and then sets each copy's permission " +
			"mode (755 unless -m gives another, octal or symbolic) and, with -o and -g, its " +
			"owner and group. An existing DEST is removed and replaced, not rewritten, so a " +
			"running program keeps its copy. With -d every operand is a directory to create, " +
			"with its missing parents; the mode applies to the named directories, and parents " +
			"get 755 minus the umask. -D creates the leading directories of DEST (or the -t " +
			"directory) first. -T treats DEST as a file even if it is a directory. -C leaves a " +
			"destination alone when it already has the same content, mode, owner and group. " +
			"-p gives each copy its source's modification time (the access time is set to " +
			"the same value). -v reports each file and directory. -s is accepted, but " +
			"stripping symbol tables needs a strip program, so it only warns that files are " +
			"installed unstripped. -c is accepted and ignored, as in GNU. -D also answers to " +
			"the gnux long name --create-leading. A symbolic -m starts from no permissions " +
			"(u=rwx,go=rx is 755), as GNU install does.",
		Permissions: "Needs read permission on each SOURCE and write and search permission " +
			"on the destination directory. Setting the owner with -o needs root, and setting " +
			"the group with -g needs ownership of the file and membership of the group. Off " +
			"Unix, -o and -g accept numeric IDs only and fail with 'Operation not supported', " +
			"only the read-only attribute of a mode is representable, and -C therefore always " +
			"reinstalls.",
		UseCases: []string{
			"Installing a built binary into a bin directory with the right mode in one step.",
			"Writing a Makefile install target that works the same on Linux, macOS and Windows.",
			"Creating a directory tree with an exact mode with -d.",
			"Deploying configuration files readable by their owner only with -m 600.",
			"Placing a file into a path whose directories may not exist yet with -D.",
			"Avoiding needless rewrites (and changed timestamps) of unchanged files with -C.",
			"Setting ownership of system files during packaging with -o and -g.",
			"Logging what a deployment copied with -v.",
		},
		Examples: []docmeta.Example{
			{Description: "Install a binary with mode 755:", Bash: "installx bin/gnux /usr/local/bin/gnux", PowerShell: "installx bin\\gnux.exe C:\\tools\\gnux.exe"},
			{Description: "Install several files into a directory:", Bash: "installx a.sh b.sh ~/bin", PowerShell: "installx a.ps1 b.ps1 $HOME\\bin"},
			{Description: "Install a configuration file readable by its owner only:", Bash: "installx -m 600 app.conf ~/.config/app/app.conf", PowerShell: "installx -m 600 app.conf $HOME\\.config\\app\\app.conf"},
			{Description: "Use a symbolic mode:", Bash: "installx -m u=rw,go=r README.md /opt/app/README.md", PowerShell: "installx -m 'u=rw,go=r' README.md C:\\app\\README.md"},
			{Description: "Create directories with a given mode:", Bash: "installx -d -m 750 /srv/app/logs /srv/app/data", PowerShell: "installx -d -m 750 C:\\srv\\logs C:\\srv\\data"},
			{Description: "Create the leading directories of the destination:", Bash: "installx -D app.conf out/etc/app/app.conf", PowerShell: "installx -D app.conf out\\etc\\app\\app.conf"},
			{Description: "Install into a target directory, creating it first:", Bash: "installx -D -t out/bin build/*", PowerShell: "installx -D -t out\\bin (Get-ChildItem build -Name)"},
			{Description: "Report what is installed:", Bash: "installx -v -t ~/bin tool1 tool2", PowerShell: "installx -v -t $HOME\\bin tool1 tool2"},
			{Description: "Leave unchanged files untouched:", Bash: "installx -C -m 644 docs/*.md /usr/share/doc/app/", PowerShell: "installx -C -m 644 (Get-ChildItem docs\\*.md -Name) C:\\app\\doc"},
			{Description: "Keep the source's modification time:", Bash: "installx -p data.db backup/data.db", PowerShell: "installx -p data.db backup\\data.db"},
			{Description: "Set the owner and group (as root):", Bash: "sudo installx -o root -g wheel -m 755 gnux /usr/local/bin/", PowerShell: "installx -m 755 gnux.exe C:\\tools\\"},
			{Description: "Install to a file name even if a directory of that name exists (an error then):", Bash: "installx -T build/app dist/app", PowerShell: "installx -T build\\app.exe dist\\app.exe"},
			{Description: "An install step in the portable gnux shell:", Bash: "gnux shell -e -c 'installx -D -m 644 site/index.html public/index.html'", PowerShell: "gnux shell -e -c 'installx -D -m 644 site/index.html public/index.html'"},
		},
	}
}

// Register installs the installx flags on fs.
func Register(fs *pflag.FlagSet) {
	fs.BoolP("copy", "c", false, "ignored; for compatibility with old Unix versions of install")
	fs.BoolP("compare", "C", false, "compare content of source and destination files, and if no change to content, ownership, and permissions, do not modify the destination at all")
	fs.BoolP("directory", "d", false, "treat all arguments as directory names; create all components of the specified directories")
	fs.BoolP("create-leading", "D", false, "create all leading components of DEST except the last, or all components of --target-directory, then copy SOURCE to DEST")
	fs.StringP("group", "g", "", "set group ownership, instead of process' current group, to `GROUP`")
	fs.StringP("mode", "m", "", "set permission mode (as in chmod) to `MODE`, instead of rwxr-xr-x")
	fs.StringP("owner", "o", "", "set ownership (super-user only) to `OWNER`")
	fs.BoolP("preserve-timestamps", "p", false, "apply access/modification times of SOURCE files to corresponding destination files")
	fs.BoolP("strip", "s", false, "strip symbol tables (not supported: warns and installs unstripped)")
	fs.StringP("target-directory", "t", "", "copy all SOURCE arguments into `DIRECTORY`")
	fs.BoolP("no-target-directory", "T", false, "treat DEST as a normal file")
	fs.BoolP("verbose", "v", false, "print the name of each created file or directory")
}

// buildOptions folds the parsed flags into install.Options.
func buildOptions(fs *pflag.FlagSet) install.Options {
	on := func(name string) bool {
		b, _ := fs.GetBool(name)
		return b
	}
	str := func(name string) string {
		v, _ := fs.GetString(name)
		return v
	}
	return install.Options{
		Directory:          on("directory"),
		CreateLeading:      on("create-leading"),
		Mode:               str("mode"),
		Owner:              str("owner"),
		Group:              str("group"),
		PreserveTimestamps: on("preserve-timestamps"),
		Strip:              on("strip"),
		TargetDirectory:    str("target-directory"),
		NoTargetDirectory:  on("no-target-directory"),
		Verbose:            on("verbose"),
		Compare:            on("compare"),
	}
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	return install.New(buildOptions(fs), s).Run(args)
}
