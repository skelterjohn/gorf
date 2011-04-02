// Copyright 2011 John Asmuth. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"os"
	"fmt"
	"flag"
	"gonicetrace.googlecode.com/hg/nicetrace"
)

var (
	LocalRoot = "."
	Usage bool
)

var UsageText = `Usage: gorf [flags] <command>
commands:
  scan <path>
  pkg <path> <old name> <new name>
  rename <path> [<type>.]<old name> <new name>
  move <old path> <new path> [<name>+]
  merge <old path> <new path>
  undo
`

func MakeErr(format string, args ...interface{}) (os.Error) {
	return os.NewError(fmt.Sprintf(format, args...))
}

func main() {
	defer nicetrace.Print()
	
	flag.StringVar(&LocalRoot, "r", ".", "Local package root")
	flag.BoolVar(&Usage, "?", false, "Print usage and quit")
	
	flag.Parse()
	
	cmds := map[string]func([]string) os.Error {
		"undo" : UndoCmd,
		"pkg" : PkgCmd,
		"rename" : RenameCmd,
		"move" : MoveCmd,
		"scan" : ScanCmd,
	}
	
	foo, ok := cmds[flag.Arg(0)]
	if ok && Usage {
		fmt.Println(Help(flag.Arg(0)))
		return
	}

	erf := func(format string, args ...interface{}) {
		fmt.Fprintf(os.Stderr, format, args...)
	}
			
	if !ok || Usage || len(flag.Args()) == 0 {
		erf(UsageText)
		erf("flags\n")
		flag.PrintDefaults()
		return
	}
	
	err := foo(flag.Args()[1:])
	if err != nil {
		erf("%v\n", err)
	}
}