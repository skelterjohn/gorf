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
  var <path> [package] <old name> <new name>
  pkg <path> <old name> <new name>
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
		"var" : VarCmd,
		"const" : ConstCmd,
		"func" : FuncCmd,
		"type" : TypeCmd,
		"field" : FieldCmd,
		"scan" : ScanCmd,
	}
	
	foo, ok := cmds[flag.Arg(0)]

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