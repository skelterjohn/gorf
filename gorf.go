// Copyright 2011 John Asmuth. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"io"
	"strings"
	"path/filepath"
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
  changes
  clear
  move <old path> <new path> [<name>+]
  moveall <old path> <new path>
  merge <old path> <new path>
  pkg <path> <old name> <new name>
  rename <path> [<type>.]<old name> <new name>
  scan <path>
  undo
`

func MakeErr(format string, args ...interface{}) (os.Error) {
	return os.NewError(fmt.Sprintf(format, args...))
}

func main() {
	defer nicetrace.Print()
	
	var err os.Error
	

	erf := func(format string, args ...interface{}) {
		fmt.Fprintf(os.Stderr, format, args...)
	}
	defer func() {
		if err != nil {
			erf("%v\n", err)
		}
	}()
	
	flag.StringVar(&LocalRoot, "r", ".", "Local package root")
	flag.BoolVar(&Usage, "?", false, "Print usage and quit")
	
	flag.Parse()
	
	norollCmds := map[string]func([]string) os.Error {
		"undo" : UndoCmd,
		"scan" : ScanCmd,
		"changes" : ChangesCmd,
		"clear" : ClearCmd,
	}
	
	rollCmds := map[string]func([]string) os.Error {
		
		"pkg" : PkgCmd,
		"rename" : RenameCmd,
		"move" : MoveCmd,
		"moveall" : MoveAllCmd,
		"merge" : MergeCmd,
	}
	
	foo, ok := norollCmds[flag.Arg(0)]
	if ok && Usage {
		fmt.Print(Help(flag.Arg(0)))
		return
	}
	if !ok {
		foo, ok = rollCmds[flag.Arg(0)]
		err = RollbackUndos()
		if err != nil {
			return
		}
		
		if ok {
			var out io.Writer
			out, err = os.Create(filepath.Join(LocalRoot, ".change.0.gorfc"))
			if err != nil {
				return
			}
			cmdline := strings.Join(flag.Args(), " ")
			fmt.Fprintf(out, "%s\n", cmdline)
		}
		//out.Close()
	}
			
	if !ok || Usage || len(flag.Args()) == 0 {
		erf(UsageText)
		erf("flags\n")
		flag.PrintDefaults()
		return
	}
	
	err = foo(flag.Args()[1:])
	
}