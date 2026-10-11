// Command gnux is a multi-call binary carrying GNU-compatible implementations of
// the core utilities, for every platform. See package cmd for how a tool is selected.
package main

//go:generate go run ./tools/gendocs documentation
//go:generate go run ./tools/gendocs web/content/docs mdx

import (
	"os"

	"github.com/braswelljr/gnux/cmd"
	"github.com/braswelljr/gnux/core"
)

func main() {
	os.Exit(cmd.Main(os.Args, core.Stdio()))
}
